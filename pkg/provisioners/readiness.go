package provisioners

import (
	"fmt"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/registry"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// conditionReady waits for a condition on a resource to report as ready.
func conditionReady(entry *registry.Entry, condition *v1.ConfigurationReadinessCheckCondition) error {
	namespace, err := resolveString(&condition.Namespace, entry)
	if err != nil {
		return err
	}

	name, err := resolveString(&condition.Name, entry)
	if err != nil {
		return err
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

	object, err := client.Resource(mapping.Resource).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	conditions, ok, _ := unstructured.NestedSlice(object.Object, "status", "conditions")
	if !ok {
		return fmt.Errorf("resource %s/%s %s contains no status conditions", condition.APIVersion, condition.Kind, name)
	}

	for _, c := range conditions {
		o, ok := c.(map[string]interface{})
		if !ok {
			return fmt.Errorf("resource %s/%s %s conditions are not objects", condition.APIVersion, condition.Kind, name)
		}

		t, ok, _ := unstructured.NestedString(o, "type")
		if !ok {
			return fmt.Errorf("resource %s/%s %s conditions contains no type", condition.APIVersion, condition.Kind, name)
		}

		if t != condition.Type {
			continue
		}

		status, ok, _ := unstructured.NestedString(o, "status")
		if !ok {
			return fmt.Errorf("resource %s/%s %s conditions contains no status", condition.APIVersion, condition.Kind, name)
		}

		if status != condition.Status {
			return fmt.Errorf("resource %s/%s %s %s condition %s is, expected %s", condition.APIVersion, condition.Kind, name, condition.Type, status, condition.Status)
		}

		return nil
	}

	return fmt.Errorf("resource %s/%s %s doesn't contain the condition %s", condition.APIVersion, condition.Kind, name, condition.Type)
}

// Ready processes any readiness checks and returns nil on success.  For now this is intended to
// be called from the service instance polling code.  In the future we may allow waits within the
// template rendering path.
func Ready(t ResourceType, entry *registry.Entry, serviceID, planID string) error {
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
			return fmt.Errorf("readiness check %s check type undefined", readinessCheck.Name)
		}
	}

	return nil
}
