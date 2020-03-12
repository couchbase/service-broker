package provisioners

import (
	"encoding/json"
	"fmt"
	"strings"

	v1 "github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/evanphx/json-patch"
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
func getTemplateBindings(serviceID, planID string) (*v1.CouchbaseServiceBrokerConfigBinding, error) {
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

// getTemplate returns the template corresponding to a template name.
func getTemplate(name string) (*v1.CouchbaseServiceBrokerConfigTemplate, error) {
	for index, template := range config.Config().Spec.Templates {
		if template.Name == name {
			return &config.Config().Spec.Templates[index], nil
		}
	}

	return nil, fmt.Errorf("unable to locate template for %s", name)
}

// resolveSource gets a parameter source from either metadata or a JSON path into user specified parameters.
func resolveSource(source *v1.CouchbaseServiceBrokerConfigTemplateParameterSource, entry *registry.Entry, parameters *runtime.RawExtension, useDefaults bool) (interface{}, error) {
	// Use a default if set.
	var value interface{}

	if useDefaults && source.Default != nil {
		switch {
		case source.Default.String != nil:
			value = *source.Default.String
		case source.Default.Bool != nil:
			value = *source.Default.Bool
		case source.Default.Int != nil:
			value = *source.Default.Int
		case source.Default.Object != nil:
			if err := json.Unmarshal(source.Default.Object.Raw, &value); err != nil {
				glog.Infof("unmarshal of source default failed: %v", err)
				return nil, err
			}
		default:
			return nil, errors.NewConfigurationError("undefined source default parameter")
		}

		glog.Infof("using source default %v", value)
	}

	switch {
	case source.Registry != nil:
		// Metadata parameters are explicitly mapped to values associated
		// with the request.
		v, ok, err := entry.GetUser(*source.Registry)
		if err != nil {
			return nil, err
		}

		if !ok {
			glog.Infof("registry key %s not set, skipping", *source.Registry)
			break
		}

		value = v

	case source.Parameter != nil:
		// Parameters reference parameters specified by the client in response to
		// the schema advertised to the client.
		parameter := source.Parameter

		// Parameters reference parameters specified by the client in response to
		// the schema advertised to the client.  Try to extract the parameter from
		// the supplied parameters.
		if parameters == nil || parameters.Raw == nil {
			glog.Infof("client parameters not set, skipping")
			break
		}

		var parametersUnstructured interface{}
		if err := json.Unmarshal(parameters.Raw, &parametersUnstructured); err != nil {
			glog.Infof("unmarshal of client parameters failed: %v", err)
			return nil, err
		}

		glog.Infof("interrogating path %s", parameter.Path)

		pointer, err := jsonpointer.New(parameter.Path)
		if err != nil {
			glog.Infof("failed to parse JSON pointer: %v", err)
			return nil, err
		}

		v, _, err := pointer.Get(parametersUnstructured)
		if err != nil {
			glog.Infof("client parameter not set, skipping: %v", err)
			break
		}

		value = v
	default:
		return nil, fmt.Errorf("source parameter type not specified")
	}

	glog.Infof("using source value %v", value)

	return value, nil
}

// resolveParameter applies parameter lookup rules and tries to return a value.
func resolveParameter(parameter *v1.CouchbaseServiceBrokerConfigTemplateParameter, entry *registry.Entry, parameters *runtime.RawExtension, useDefaults bool) (interface{}, error) {
	values := make([]interface{}, len(parameter.Sources))

	// Collect any sources requested.
	for index := range parameter.Sources {
		value, err := resolveSource(&parameter.Sources[index], entry, parameters, useDefaults)
		if err != nil {
			return nil, err
		}

		if parameter.Required && value == nil {
			glog.Infof("source unset but parameter is required")
			return nil, errors.NewParameterError("parameter %s is required", parameter.Name)
		}

		values[index] = value
	}

	// Default to the first value before optional mutation.
	var value interface{}

	if len(values) != 0 {
		value = values[0]
	}

	// Mutate the sources into a finished value if requested.
	if parameter.Mutation != nil {
		if parameter.Mutation.Format != nil {
			// Error checking needs to happen.
			value = fmt.Sprintf(*parameter.Mutation.Format, values...)
		}
	}

	return value, nil
}

// renderTemplate accepts a template defined in the configuration and applies any
// request or metadata parameters to it.
func renderTemplate(template *v1.CouchbaseServiceBrokerConfigTemplate, entry *registry.Entry, parameters *runtime.RawExtension) (*v1.CouchbaseServiceBrokerConfigTemplate, error) {
	glog.Infof("rendering template %s", template.Name)

	// We will be modifying the template in place, so first clone it as the
	// config is immutable.
	t := template.DeepCopy()

	// Now for the fun bit.  Work through each defined parameter and apply it to
	// the object.  This basically works like JSON patch++, automatically filling
	// in parent objects and arrays as necessary.
	for index, parameter := range t.Parameters {
		value, err := resolveParameter(&t.Parameters[index], entry, parameters, true)
		if err != nil {
			return nil, err
		}

		// Set the registry entry if defined.
		if parameter.Destination.Registry != nil {
			glog.Infof("setting registry entry %s to %v", *parameter.Destination.Registry, value)

			strValue, ok := value.(string)
			if !ok {
				return nil, errors.NewConfigurationError("parameter %s is not a string", parameter.Name)
			}

			if err := entry.SetUser(*parameter.Destination.Registry, strValue); err != nil {
				return nil, errors.NewConfigurationError(err.Error())
			}
		}

		// Set each destination path using JSON patch.
		if t.Template == nil {
			continue
		}

		patches := []string{}

		for _, path := range parameter.Destination.Paths {
			valueJSON, err := json.Marshal(value)
			if err != nil {
				glog.Infof("marshal of value failed: %v", err)
				return nil, err
			}

			patches = append(patches, fmt.Sprintf(`{"op":"add","path":"%s","value":%s}`, path, string(valueJSON)))
		}

		patchSet := "[" + strings.Join(patches, ",") + "]"

		glog.Infof("applying patchset %s", patchSet)

		patch, err := jsonpatch.DecodePatch([]byte(patchSet))
		if err != nil {
			glog.Infof("decode of JSON patch failed: %v", err)
			return nil, err
		}

		object, err := patch.Apply(t.Template.Raw)
		if err != nil {
			glog.Infof("apply of JSON patch failed: %v", err)
			return nil, err
		}

		t.Template.Raw = object
	}

	if t.Template != nil {
		glog.Infof("rendered template %s", string(t.Template.Raw))
	}

	return t, nil
}
