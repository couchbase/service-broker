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

	"github.com/golang/glog"
)

// getServiceAndPlanNames translates from GUIDs to human readable names used in configuration.
func getServiceAndPlanNames(serviceID, planID string) (string, string, error) {
	for _, service := range config.Config().Spec.Catalog.Services {
		if service.ID == serviceID {
			for _, plan := range service.Plans {
				if plan.ID == planID {
					return service.Name, plan.Name, nil
				}
			}

			return "", "", fmt.Errorf("%w: unable to locate plan for ID %s", ErrResourceReferenceMissing, planID)
		}
	}

	return "", "", fmt.Errorf("%w: unable to locate service for ID %s", ErrResourceReferenceMissing, serviceID)
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

	return nil, fmt.Errorf("%w: unable to locate template bindings for service plan %s/%s", ErrResourceReferenceMissing, service, plan)
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
		return nil, fmt.Errorf("%w: illegal binding type %s", ErrUndefinedType, string(t))
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

// renderTemplate accepts a template defined in the configuration and applies any
// request or metadata parameters to it.
func renderTemplate(template *v1.ConfigurationTemplate, entry *registry.Entry, data interface{}) (*v1.ConfigurationTemplate, error) {
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
	if object, err = recurseRenderTemplate(object, entry, data); err != nil {
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
