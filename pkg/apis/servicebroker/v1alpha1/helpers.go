// Copyright 2021 Couchbase, Inc.
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

package v1alpha1

import (
	"fmt"
)

// GetServiceAndPlanNames translates from GUIDs to human readable names used in configuration.
func (config *ServiceBrokerConfig) GetServiceAndPlanNames(serviceID, planID string) (string, string, error) {
	for _, service := range config.Spec.Catalog.Services {
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

// GetTemplateBindings returns the template bindings associated with a creation request's
// service and plan IDs.
func (config *ServiceBrokerConfig) GetTemplateBindings(serviceID, planID string) (*ConfigurationBinding, error) {
	service, plan, err := config.GetServiceAndPlanNames(serviceID, planID)
	if err != nil {
		return nil, err
	}

	for index, binding := range config.Spec.Bindings {
		if binding.Service == service && binding.Plan == plan {
			return &config.Spec.Bindings[index], nil
		}
	}

	return nil, fmt.Errorf("%w: unable to locate template bindings for service plan %s/%s", ErrResourceReferenceMissing, service, plan)
}
