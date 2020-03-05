package provisioners

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/config"

	"github.com/evanphx/json-patch"
	"github.com/go-openapi/jsonpointer"
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/runtime"
)

// getNamespace returns the namespace to provision resources in.  This is the namespace
// the broker lives in by default, however when operating as a kubernetes cluster service
// broker then this information is passed as request context.
func getNamespace(context *runtime.RawExtension) (string, error) {
	if context != nil {
		var ctx interface{}

		if err := json.Unmarshal(context.Raw, &ctx); err != nil {
			glog.Errorf("unmarshal of client context failed: %v", err)
			return "", err
		}

		pointer, err := jsonpointer.New("/namespace")
		if err != nil {
			glog.Errorf("failed to parse JSON pointer: %v", err)
			return "", err
		}

		v, _, err := pointer.Get(ctx)
		if err == nil {
			namespace, ok := v.(string)
			if ok {
				return namespace, nil
			}

			glog.Errorf("request context namespace not a string")

			return "", fmt.Errorf("request context namespace not a string")
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

// resolveParameter applies parameter lookup rules and tries to return a value.
func resolveParameter(parameter *v1.CouchbaseServiceBrokerConfigTemplateParameter, instanceID string, parameters *runtime.RawExtension, useDefaults bool) (interface{}, error) {
	// You cannot have multiple parameter sources.  Metadata takes
	// precedence over request parameters.
	var value interface{}

	switch {
	case parameter.Source.Metadata != nil:
		// Metadata parameters are explicitly mapped to values associated
		// with the request.
		switch *parameter.Source.Metadata {
		case "id":
			glog.Infof("using metadata id %s", instanceID)
			value = instanceID
		default:
			return nil, fmt.Errorf("undefined metadata parameter %s", *parameter.Source.Metadata)
		}
	case parameter.Source.Parameter != nil:
		// Parameters reference parameters specified by the client in response to
		// the schema advertised to the client.  First set the default then try
		// override if the parameter is specified by the client.
		if useDefaults && parameter.Source.Parameter.Default != nil {
			switch {
			case parameter.Source.Parameter.Default.String != nil:
				value = *parameter.Source.Parameter.Default.String
			case parameter.Source.Parameter.Default.Bool != nil:
				value = *parameter.Source.Parameter.Default.Bool
			case parameter.Source.Parameter.Default.Int != nil:
				value = *parameter.Source.Parameter.Default.Int
			case parameter.Source.Parameter.Default.Object != nil:
				if err := json.Unmarshal(parameter.Source.Parameter.Default.Object.Raw, &value); err != nil {
					glog.Errorf("unmarshal of default parameter failed: %v", err)
					return nil, err
				}
			default:
				return nil, fmt.Errorf("undefined default parameter")
			}

			glog.Infof("using parameter default %v", value)
		}
		// Parameters reference parameters specified by the client in response to
		// the schema advertised to the client.  Try to extract the parameter from
		// the supplied parameters.
		if parameters.Raw == nil {
			glog.Infof("client parameters not set, skipping")
			break
		}

		var parametersUnstructured interface{}
		if err := json.Unmarshal(parameters.Raw, &parametersUnstructured); err != nil {
			glog.Errorf("unmarshal of client parameters failed: %v", err)
			return nil, err
		}

		glog.Infof("interrogating path %s", parameter.Source.Parameter.Path)

		pointer, err := jsonpointer.New(parameter.Source.Parameter.Path)
		if err != nil {
			glog.Errorf("failed to parse JSON pointer: %v", err)
			return nil, err
		}

		v, _, err := pointer.Get(parametersUnstructured)
		if err != nil {
			glog.Infof("client parameter not set, skipping: %v", err)
			break
		}

		glog.Infof("using parameter %v", v)

		value = v
	default:
		return nil, fmt.Errorf("source parameter type not specified")
	}

	// Set the requested paths in the template if the value is valid.
	// We require that all parent elements in the template be populated,
	// even if that means an empty object, exactly like the JSON patch
	// specification.
	if value == nil {
		if parameter.Required {
			glog.Errorf("value unset but parameter is required")
			return nil, fmt.Errorf("parameter %s is required", parameter.Name)
		}

		glog.Infof("value unset, skipping")

		return nil, nil
	}

	// Apply any necessary source filters.  Only one can be applied at any time.
	switch {
	case parameter.Source.Prefix != nil:
		v, ok := value.(string)
		if !ok {
			glog.Errorf("prefix is set but value is not a string")
			return nil, fmt.Errorf("prefix is set but value is not a string")
		}

		value = *parameter.Source.Prefix + v
	case parameter.Source.Suffix != nil:
		v, ok := value.(string)
		if !ok {
			glog.Errorf("suffix is set but value is not a string")
			return nil, fmt.Errorf("suffix is set but value is not a string")
		}

		value = v + *parameter.Source.Suffix
	case parameter.Source.Format != nil:
		value = fmt.Sprintf(*parameter.Source.Format, value)
	}

	return value, nil
}

// renderTemplate accepts a template defined in the configuration and applies any
// request or metadata parameters to it.
func renderTemplate(template *v1.CouchbaseServiceBrokerConfigTemplate, instanceID string, parameters *runtime.RawExtension) (*v1.CouchbaseServiceBrokerConfigTemplate, error) {
	glog.Infof("rendering template %s", template.Name)

	// We will be modifying the template in place, so first clone it as the
	// config is immutable.
	t := template.DeepCopy()

	// Now for the fun bit.  Work through each defined parameter and apply it to
	// the object.  This basically works like JSON patch++, automatically filling
	// in parent objects and arrays as necessary.
	for index, parameter := range t.Parameters {
		value, err := resolveParameter(&t.Parameters[index], instanceID, parameters, true)
		if err != nil {
			return nil, err
		}

		if value == nil {
			continue
		}

		// Set each destination path using JSON patch.
		patches := []string{}

		for _, path := range parameter.Destination.Paths {
			valueJSON, err := json.Marshal(value)
			if err != nil {
				glog.Errorf("marshal of value failed: %v", err)
				return nil, err
			}

			patches = append(patches, fmt.Sprintf(`{"op":"add","path":"%s","value":%s}`, path, string(valueJSON)))
		}

		patchSet := "[" + strings.Join(patches, ",") + "]"

		glog.Infof("applying patchset %s", patchSet)

		patch, err := jsonpatch.DecodePatch([]byte(patchSet))
		if err != nil {
			glog.Errorf("decode of JSON patch failed: %v", err)
			return nil, err
		}

		object, err := patch.Apply(t.Template.Raw)
		if err != nil {
			glog.Errorf("apply of JSON patch failed: %v", err)
			return nil, err
		}

		t.Template.Raw = object
	}

	glog.Infof("rendered template %s", string(t.Template.Raw))

	return t, nil
}
