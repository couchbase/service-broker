// Copyright 2020-2021 Couchbase, Inc.
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

package config

import (
	"errors"
	"fmt"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
)

// ErrConfigurationInvalid is a generic configuration error.
var ErrConfigurationInvalid = errors.New("configuration is invalid")

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

// validate does any validation that cannot be performed by the JSON schema
// included in the CRD.
func validate(config *v1.ServiceBrokerConfig) error {
	// Check that service offerings and plans are bound properly to configuration.
	for _, service := range config.Spec.Catalog.Services {
		for _, plan := range service.Plans {
			// Each service plan must have a service binding.
			binding := getBindingForServicePlan(config, service.Name, plan.Name)
			if binding == nil {
				return fmt.Errorf("%w: service plan '%s' for offering '%s' does not have a binding", ErrConfigurationInvalid, plan.Name, service.Name)
			}

			// Only bindable service plans may have templates for bindings.
			bindable := service.Bindable
			if plan.Bindable != nil {
				bindable = *plan.Bindable
			}

			if !bindable && binding.ServiceBinding != nil {
				return fmt.Errorf("%w: service plan '%s' for offering '%s' not bindable, but binding '%s' defines service binding configuarion", ErrConfigurationInvalid, plan.Name, service.Name, binding.Name)
			}

			if bindable && binding.ServiceBinding == nil {
				return fmt.Errorf("%w: service plan '%s' for offering '%s' bindable, but binding '%s' does not define service binding configuarion", ErrConfigurationInvalid, plan.Name, service.Name, binding.Name)
			}
		}
	}

	// Check that configuration bindings are properly configured.
	for _, binding := range config.Spec.Bindings {
		// Bindings cannot do nothing.
		if len(binding.ServiceInstance.Registry) == 0 && len(binding.ServiceInstance.Templates) == 0 {
			return fmt.Errorf("%w: binding '%s' does nothing for service instances", ErrConfigurationInvalid, binding.Name)
		}

		if binding.ServiceBinding != nil {
			if len(binding.ServiceBinding.Registry) == 0 && len(binding.ServiceBinding.Templates) == 0 {
				return fmt.Errorf("%w: binding '%s' does nothing for service bindings", ErrConfigurationInvalid, binding.Name)
			}
		}

		// Binding templates must exist.
		for _, template := range binding.ServiceInstance.Templates {
			if getTemplateByName(config, template) == nil {
				return fmt.Errorf("%w: template '%s', referenced by binding '%s' service instance, must exist", ErrConfigurationInvalid, template, binding.Name)
			}
		}

		if binding.ServiceBinding != nil {
			for _, template := range binding.ServiceBinding.Templates {
				if getTemplateByName(config, template) == nil {
					return fmt.Errorf("%w: template '%s', referenced by binding '%s' service binding, must exist", ErrConfigurationInvalid, template, binding.Name)
				}
			}
		}
	}

	return nil
}
