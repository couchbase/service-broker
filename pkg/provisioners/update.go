package provisioners

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/couchbase/service-broker/pkg/api"
	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/evanphx/json-patch"
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Updater caches various data associated with updating a service instance.
type Updater struct {
	resourceType ResourceType

	// request is the incoming client requesst.
	request *api.UpdateServiceInstanceRequest

	// resources is a list of resources that need to be updated as a result
	// of any required update operations.
	resources []*unstructured.Unstructured
}

// NewUpdater returns a new controler capable of updaing a service instance.
func NewUpdater(resourceType ResourceType, request *api.UpdateServiceInstanceRequest) (*Updater, error) {
	u := &Updater{
		resourceType: resourceType,
		request:      request,
	}

	return u, nil
}

// Prepare pre-processes the registry and templates.
func (u *Updater) Prepare(entry *registry.Entry) error {
	// Use the cached versions, as the request parameters may not be set.
	serviceID, ok, err := entry.GetString(registry.ServiceID)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("unable to lookup service instance service ID")
	}

	planID, ok, err := entry.GetString(registry.PlanID)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("unable to lookup service instance plan ID")
	}

	// Collate and render our templates.
	glog.Infof("looking up bindings for service %s, plan %s", serviceID, planID)

	templates, err := getTemplateBinding(u.resourceType, serviceID, planID)
	if err != nil {
		return err
	}

	// Prepare the client code
	client := config.Clients().Dynamic()

	for _, templateName := range templates.Templates {
		glog.Infof("getting resource for template %s", templateName)

		// Lookup the template, the name may be dynamic e.g. based on instance
		// ID so render it first before getting from the API.
		template, err := getTemplate(templateName)
		if err != nil {
			return err
		}

		// Updates from multiple service instances or bindings will
		// inevitably lead to split-brain, with values changing at
		// random.
		if template.Singleton {
			glog.Info("template is a singleton, ignoring update")
			continue
		}

		// No parameters, nothing can change.
		if len(template.Parameters) == 0 {
			glog.Info("template not paramterized, ignoring update")
			continue
		}

		t, err := renderTemplate(template, entry)
		if err != nil {
			return err
		}

		newJSON := t.Template.Raw

		// Unmarshal the object so we can derive the kind and name.
		newObject := &unstructured.Unstructured{}
		if err := json.Unmarshal(newJSON, newObject); err != nil {
			return err
		}

		gvk := newObject.GroupVersionKind()

		mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		// The namespace defaults to that configured in the object, if not
		// specified we use the namespace defined in the context (where the
		// service instance or binding is created).
		namespace := newObject.GetNamespace()
		if namespace == "" {
			n, ok, err := entry.GetString(registry.Namespace)
			if err != nil {
				return err
			}

			if !ok {
				return fmt.Errorf("unable to lookup namespace")
			}

			namespace = n
		}

		glog.Infof("using namespace %s", namespace)

		// Get the current resource.
		// We will extract the annotation that contains the JSON we generated
		// when creating the resource, then compare it against the JSON when
		// we re-render the resource.  If those two differ then make a merge
		// patch and apply it to the current resource.  This way we can and and
		// remove configuration in response to parameter changes and also
		// preserve any mutations that have been applied by Kubernetes or any
		// other controller.
		currentObject, err := client.Resource(mapping.Resource).Namespace(namespace).Get(newObject.GetName(), metav1.GetOptions{})
		if err != nil {
			glog.Infof("failed to get resource %s/%s %s", newObject.GetAPIVersion(), newObject.GetKind(), newObject.GetName())
			return err
		}

		originalJSONString, ok, _ := unstructured.NestedString(currentObject.Object, "metadata", "annotations", v1.ResourceAnnotation)
		if !ok {
			return fmt.Errorf("failed to lookup original resource")
		}

		originalJSON := []byte(originalJSONString)

		originalObject := &unstructured.Unstructured{}
		if err := json.Unmarshal(originalJSON, originalObject); err != nil {
			return err
		}

		glog.Infof("original resource: %s", string(originalJSON))
		glog.Infof("new resource: %s", string(newJSON))

		// jsonpatch.Equal is broken, so use reflection.
		if reflect.DeepEqual(originalObject, newObject) {
			glog.Infof("resource unchanged")
			continue
		}

		mergePatch, err := jsonpatch.CreateMergePatch(originalJSON, newJSON)
		if err != nil {
			return err
		}

		glog.Infof("marge patch: %s", string(mergePatch))

		currentJSON, err := json.Marshal(currentObject)
		if err != nil {
			return err
		}

		glog.Infof("current resource: %s", string(currentJSON))

		mergedJSON, err := jsonpatch.MergePatch(currentJSON, mergePatch)
		if err != nil {
			return err
		}

		mergedObject := &unstructured.Unstructured{}
		if err := json.Unmarshal(mergedJSON, mergedObject); err != nil {
			return err
		}

		// Update the resource annotation with our new idealized representation
		// of what we asked for, so future updates will diff against the right
		// things.
		if err := unstructured.SetNestedField(mergedObject.Object, string(newJSON), "metadata", "annotations", v1.ResourceAnnotation); err != nil {
			return err
		}

		glog.Infof("merged resource: %s", string(mergedJSON))

		u.resources = append(u.resources, mergedObject)
	}

	return nil
}

// run performs asynchronous update tasks.
func (u *Updater) run() error {
	glog.Info("updating resources")

	// Prepare the client code
	client := config.Clients().Dynamic()

	for _, resource := range u.resources {
		glog.Infof("updating resource %s/%s %s", resource.GetAPIVersion(), resource.GetKind(), resource.GetName())

		gvk := resource.GroupVersionKind()

		mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		if _, err := client.Resource(mapping.Resource).Namespace(resource.GetNamespace()).Update(resource, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// Run performs asynchronous update tasks.
func (u *Updater) Run(entry *registry.Entry) {
	if err := operation.Complete(entry, u.run()); err != nil {
		glog.Infof("failed to delete instance")
	}
}
