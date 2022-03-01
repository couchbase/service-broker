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

package provisioners

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/pkg/util"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// conditionUnreadyError is returned when a condition is not matched for acceptable
// reasons, e.g. doesn't exist or doesn't match.
type conditionUnreadyError struct {
	message string
}

// newConditionUnreadyError returns a new condition unready error.
func newConditionUnreadyError(message string, arguments ...interface{}) error {
	return &conditionUnreadyError{message: fmt.Sprintf(message, arguments...)}
}

// IsConditionUnreadyError checks if the error is due to a condition being unready.
func IsConditionUnreadyError(e error) bool {
	if _, ok := e.(*conditionUnreadyError); !ok {
		return false
	}

	return true
}

// Error returns the condition unready error string.
func (e *conditionUnreadyError) Error() string {
	return e.message
}

// conditionReady waits for a condition on a resource to report as ready.  Returns nil on success and
// an error otherwise.
func conditionReady(entry *registry.Entry, condition *v1.ConfigurationReadinessCheckCondition) error {
	namespaceRaw, err := renderTemplateString(condition.Namespace, entry, nil)
	if err != nil {
		return err
	}

	namespace, ok := namespaceRaw.(string)
	if !ok {
		return errors.NewConfigurationError("condition resource namespace not a string %v", namespaceRaw)
	}

	nameRaw, err := renderTemplateString(condition.Name, entry, nil)
	if err != nil {
		return err
	}

	name, ok := nameRaw.(string)
	if !ok {
		return errors.NewConfigurationError("condition resource name not a string %v", nameRaw)
	}

	gv, err := schema.ParseGroupVersion(condition.APIVersion)
	if err != nil {
		return err
	}

	gvk := gv.WithKind(condition.Kind)

	mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	client := config.Clients().Dynamic()

	var object *unstructured.Unstructured

	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		object, err = client.Resource(mapping.Resource).Get(context.TODO(), name, metav1.GetOptions{})
	} else {
		object, err = client.Resource(mapping.Resource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	}

	if err != nil {
		return err
	}

	conditions, ok, _ := unstructured.NestedSlice(object.Object, "status", "conditions")
	if !ok {
		return newConditionUnreadyError("resource %s/%s %s contains no status conditions", condition.APIVersion, condition.Kind, name)
	}

	for _, c := range conditions {
		o, ok := c.(map[string]interface{})
		if !ok {
			return newConditionUnreadyError("resource %s/%s %s conditions are not objects", condition.APIVersion, condition.Kind, name)
		}

		t, ok, _ := unstructured.NestedString(o, "type")
		if !ok {
			return newConditionUnreadyError("resource %s/%s %s conditions contains no type", condition.APIVersion, condition.Kind, name)
		}

		if t != condition.Type {
			continue
		}

		status, ok, _ := unstructured.NestedString(o, "status")
		if !ok {
			return newConditionUnreadyError("resource %s/%s %s conditions contains no status", condition.APIVersion, condition.Kind, name)
		}

		if status != condition.Status {
			return newConditionUnreadyError("resource %s/%s %s %s condition %s is, expected %s", condition.APIVersion, condition.Kind, name, condition.Type, status, condition.Status)
		}

		return nil
	}

	return newConditionUnreadyError("resource %s/%s %s doesn't contain the condition %s", condition.APIVersion, condition.Kind, name, condition.Type)
}

// Ready processes any readiness checks and returns nil on success.  For now this is intended to
// be called from the service instance polling code.  In the future we may allow waits within the
// template rendering path.  Returns nil on success and an error otherwise.
func Ready(t ResourceType, entry *registry.Entry, serviceID, planID string) error {
	// Only do this for provisioning operations, it makes no sense to check for
	// readiness when deprovisioning and we expect updates to maintain service
	// availability.
	op, ok, err := entry.GetString(registry.Operation)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("%w: service instance missing operation", ErrRegistryEntryMissing)
	}

	if operation.Type(op) != operation.TypeProvision {
		return err
	}

	// Collate and render our templates.
	templates, err := getTemplateBinding(t, serviceID, planID)
	if err != nil {
		return err
	}

	for _, readinessCheck := range templates.ReadinessChecks {
		switch {
		case readinessCheck.Condition != nil:
			if err := conditionReady(entry, readinessCheck.Condition); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: readiness check %s check type undefined", ErrResourceAttributeMissing, readinessCheck.Name)
		}
	}

	return nil
}

// barrier waits for a readiness check to complete before continuing.
func barrier(readinessCheck v1.ConfigurationReadinessCheck, entry *registry.Entry) error {
	doCheck := func() error {
		switch {
		case readinessCheck.Condition != nil:
			if err := conditionReady(entry, readinessCheck.Condition); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: readiness check %s check type undefined", ErrResourceAttributeMissing, readinessCheck.Name)
		}

		return nil
	}

	timeout := time.Minute
	if readinessCheck.Timeout != nil {
		timeout = readinessCheck.Timeout.Duration
	}

	return util.WaitFor(doCheck, timeout)
}
