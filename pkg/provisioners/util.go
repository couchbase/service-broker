// Copyright 2020 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provisioners

import (
	"encoding/json"
	"fmt"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/log"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/go-openapi/jsonpointer"
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/runtime"
)

// GetNamespace returns the namespace to provision resources in.  This is the namespace
// the broker lives in by default, however when operating as a kubernetes cluster service
// broker then this information is passed as request context.
func GetNamespace(context *runtime.RawExtension) (string, error) {
	if context != nil {
		var ctx interface{}

		if err := json.Unmarshal(context.Raw, &ctx); err != nil {
			glog.Infof("unmarshal of client context failed: %v", err)
			return "", err
		}

		pointer, err := jsonpointer.New("/namespace")
		if err != nil {
			glog.Infof("failed to parse JSON pointer: %v", err)
			return "", err
		}

		v, _, err := pointer.Get(ctx)
		if err == nil {
			namespace, ok := v.(string)
			if ok {
				return namespace, nil
			}

			glog.Infof("request context namespace not a string")

			return "", errors.NewParameterError("request context namespace not a string")
		}
	}

	return config.Namespace(), nil
}

// getServiceAndPlanNames translates from GUIDs to human readable names used in configuration.
func getServiceAndPlanNames(serviceID, planID string) (string, string, error) {
	for _, service := range config.Config().Spec.Catalog.Services {
		if service.ID == serviceID {
			for _, plan := range service.Plans {
				if plan.ID == planID {
					return service.Name, plan.Name, nil
				}
			}

			return "", "", fmt.Errorf("unable to locate plan for ID %s", planID)
		}
	}

	return "", "", fmt.Errorf("unable to locate service for ID %s", serviceID)
}

// getTemplateBindings returns the template bindings associated with a creation request's
// service and plan IDs.
func getTemplateBindings(serviceID, planID string) (*v1.ConfigurationBinding, error) {
	service, plan, err := getServiceAndPlanNames(serviceID, planID)
	if err != nil {
		return nil, err
	}

	for index, binding := range config.Config().Spec.Bindings {
		if binding.Service == service && binding.Plan == plan {
			return &config.Config().Spec.Bindings[index], nil
		}
	}

	return nil, fmt.Errorf("unable to locate template bindings for service plan %s/%s", service, plan)
}

// getTemplateBinding returns the binding associated with a specific resource type.
func getTemplateBinding(t ResourceType, serviceID, planID string) (*v1.ServiceBrokerTemplateList, error) {
	bindings, err := getTemplateBindings(serviceID, planID)
	if err != nil {
		return nil, err
	}

	var templates *v1.ServiceBrokerTemplateList

	switch t {
	case ResourceTypeServiceInstance:
		templates = &bindings.ServiceInstance
	case ResourceTypeServiceBinding:
		templates = bindings.ServiceBinding
	default:
		return nil, fmt.Errorf("illegal binding type %s", string(t))
	}

	if templates == nil {
		return nil, errors.NewConfigurationError("missing bindings for type %s", string(t))
	}

	return templates, nil
}

// getTemplate returns the template corresponding to a template name.
func getTemplate(name string) (*v1.ConfigurationTemplate, error) {
	for index, template := range config.Config().Spec.Templates {
		if template.Name == name {
			return &config.Config().Spec.Templates[index], nil
		}
	}

	return nil, errors.NewConfigurationError("unable to locate template for %s", name)
}

// resolveParameter attempts to find the parameter path in the provided JSON.
func resolveParameter(path string, entry *registry.Entry) (interface{}, bool, error) {
	var parameters interface{}

	ok, err := entry.Get(registry.Parameters, &parameters)
	if err != nil {
		return nil, false, err
	}

	if !ok {
		return nil, false, fmt.Errorf("unable to lookup parameters")
	}

	pointer, err := jsonpointer.New(path)
	if err != nil {
		glog.Infof("failed to parse JSON pointer: %v", err)
		return nil, false, err
	}

	value, _, err := pointer.Get(parameters)
	if err != nil {
		return nil, false, nil
	}

	return value, true, nil
}

// resolveAccessor looks up a registry or parameter.
func resolveAccessor(accessor *v1.Accessor, entry *registry.Entry) (interface{}, error) {
	var value interface{}

	switch {
	case accessor.Registry != nil:
		v, ok, err := entry.GetUser(*accessor.Registry)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, nil
		}

		value = v

	case accessor.Parameter != nil:
		v, ok, err := resolveParameter(*accessor.Parameter, entry)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, nil
		}

		value = v
	default:
		return nil, fmt.Errorf("accessor must have one method defined")
	}

	glog.Infof("resolved parameter value %v", value)

	return value, nil
}

// resolveString looks up a string value.
func resolveString(str *v1.String, entry *registry.Entry) (string, error) {
	if str.String != nil {
		return *str.String, nil
	}

	value, err := resolveAccessor(&str.Accessor, entry)
	if err != nil {
		return "", err
	}

	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value %v is not a string", value)
	}

	return stringValue, nil
}

// renderTemplate accepts a template defined in the configuration and applies any
// request or metadata parameters to it.
func renderTemplate(template *v1.ConfigurationTemplate, entry *registry.Entry) (*v1.ConfigurationTemplate, error) {
	glog.Infof("rendering template %s", template.Name)

	if template.Template == nil || template.Template.Raw == nil {
		return nil, errors.NewConfigurationError("template %s is not defined", template.Name)
	}

	glog.V(log.LevelDebug).Infof("template source: %s", string(template.Template.Raw))

	// We will be modifying the template in place, so first clone it as the
	// config is immutable.
	t := template.DeepCopy()

	var object interface{}
	if err := json.Unmarshal(t.Template.Raw, &object); err != nil {
		return nil, err
	}

	var err error
	if object, err = recurseRenderTemplate(object, entry); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}

	t.Template.Raw = raw

	glog.Infof("rendered template %s", string(t.Template.Raw))

	return t, nil
}
