package config

import (
	"fmt"
	"strings"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
)

// getBindingForServicePlan looks up a configuration binding for a named service plan.
func getBindingForServicePlan(config *v1.ServiceBrokerConfig, serviceName, planName string) *v1.ConfigurationBinding {
	for index, binding := range config.Spec.Bindings {
		if binding.Service == serviceName && binding.Plan == planName {
			return &config.Spec.Bindings[index]
		}
	}

	return nil
}

// getTemplateByName looks up a configuration template for a named template.
func getTemplateByName(config *v1.ServiceBrokerConfig, templateName string) *v1.ConfigurationTemplate {
	for index, template := range config.Spec.Templates {
		if template.Name == templateName {
			return &config.Spec.Templates[index]
		}
	}

	return nil
}

// recurseDetectCycles does a depth-first search of the tree of the dependency graph
// of templates and looks for infinite loops.  Ancestors may be pre-populated or nil
// if the parameters don't beloing to a template.
func recurseDetectCycles(config *v1.ServiceBrokerConfig, ancestors []string, parameters []v1.ConfigurationParameter) error {
	for _, parameter := range parameters {
		// Only concerned with parameter source templates
		if parameter.Source == nil || parameter.Source.Template == nil {
			continue
		}

		// Perform cycle detection by seeing if the referenced template is
		// its own ancestor.
		child := *parameter.Source.Template

		for _, ancestor := range ancestors {
			if ancestor == child {
				chain := strings.Join(ancestors, " -> ") + " -> " + child
				return fmt.Errorf("dependency cycle detected in template chain %s", chain)
			}
		}

		// Look up the template ready for depth first recursion.
		template := getTemplateByName(config, child)
		if template == nil {
			return fmt.Errorf("parameter %s references non existent tempate %s", parameter.Name, child)
		}

		// Push the new ancestor onto the stack and recurse.
		ancestors = append(ancestors, child)
		if err := recurseDetectCycles(config, ancestors, template.Parameters); err != nil {
			return err
		}

		// Pop the new ancestor off the stack and continue.
		popOne := 1
		ancestors = ancestors[:len(ancestors)-popOne]
	}

	return nil
}

// detectCyclesTemplateList detects dependency cycles in a configuration binding
// list, first for any configuration parameters containing template snippets, and
// then for the templates themselves.
func detectCyclesTemplateList(config *v1.ServiceBrokerConfig, templates *v1.ServiceBrokerTemplateList) error {
	if err := recurseDetectCycles(config, nil, templates.Parameters); err != nil {
		return err
	}

	for _, templateName := range templates.Templates {
		template := getTemplateByName(config, templateName)

		if err := recurseDetectCycles(config, []string{template.Name}, template.Parameters); err != nil {
			return err
		}
	}

	return nil
}

// detectCycles does a recursive lookup of templates to ensure there are no
// infinite loops lurking in bad configuration.
func detectCycles(config *v1.ServiceBrokerConfig, binding *v1.ConfigurationBinding) error {
	if err := detectCyclesTemplateList(config, &binding.ServiceInstance); err != nil {
		return err
	}

	if binding.ServiceBinding != nil {
		if err := detectCyclesTemplateList(config, binding.ServiceBinding); err != nil {
			return err
		}
	}

	return nil
}

// validate does any validation that cannot be performed by the JSON schema
// included in the CRD.
func validate(config *v1.ServiceBrokerConfig) error {
	// Check that service offerings and plans are bound properly to configuration.
	for _, service := range config.Spec.Catalog.Services {
		for _, plan := range service.Plans {
			// Each service plan must have a service binding.
			binding := getBindingForServicePlan(config, service.Name, plan.Name)
			if binding == nil {
				return fmt.Errorf("service plan %s for offering %s does not have a configuration binding", plan.Name, service.Name)
			}

			// Only bindable service plans may have templates for bindings.
			bindable := service.Bindable
			if plan.Bindable != nil {
				bindable = *plan.Bindable
			}

			if !bindable && binding.ServiceBinding != nil {
				return fmt.Errorf("service plan %s for offering %s not bindable, but configuration binding %s defines service binding configuarion", plan.Name, service.Name, binding.Name)
			}

			if bindable && binding.ServiceBinding == nil {
				return fmt.Errorf("service plan %s for offering %s bindable, but configuration binding %s does not define service binding configuarion", plan.Name, service.Name, binding.Name)
			}
		}
	}

	// Check that configuration bindings are properly configured.
	for index, binding := range config.Spec.Bindings {
		// Bindings cannot do nothing.
		if len(binding.ServiceInstance.Parameters) == 0 && len(binding.ServiceInstance.Templates) == 0 {
			return fmt.Errorf("configuration binding %s does nothing for service instances", binding.Name)
		}

		if binding.ServiceBinding != nil {
			if len(binding.ServiceBinding.Parameters) == 0 && len(binding.ServiceBinding.Templates) == 0 {
				return fmt.Errorf("configuration binding %s does nothing for service bindings", binding.Name)
			}
		}

		// Binding templates must exist.
		for _, template := range binding.ServiceInstance.Templates {
			if getTemplateByName(config, template) == nil {
				return fmt.Errorf("template %s referenced by configuration %s service instance must exist", template, binding.Name)
			}
		}

		if binding.ServiceBinding != nil {
			for _, template := range binding.ServiceBinding.Templates {
				if getTemplateByName(config, template) == nil {
					return fmt.Errorf("template %s referenced by configuration %s service binding must exist", template, binding.Name)
				}
			}
		}

		// Templates must not contain cycles.
		if err := detectCycles(config, &config.Spec.Bindings[index]); err != nil {
			return err
		}
	}

	return nil
}
