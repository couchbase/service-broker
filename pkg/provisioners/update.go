package provisioners

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Updater caches various data associated with updating a service instance.
type Updater struct {
	resourceType ResourceType

	// request is the incomiong client requesst.
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

func (u *Updater) PrepareResources(entry *registry.Entry) error {
	// Use the cached versions, as the request parameters may not be set.
	serviceID, ok := entry.Get(registry.ServiceID)
	if !ok {
		return fmt.Errorf("unable to lookup service instance service ID")
	}

	planID, ok := entry.Get(registry.PlanID)
	if !ok {
		return fmt.Errorf("unable to lookup service instance plan ID")
	}

	namespace, ok := entry.Get(registry.Namespace)
	if !ok {
		return fmt.Errorf("unable to lookup namespace")
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

		if len(template.Parameters) == 0 {
			glog.Info("template not paramterized, ignoring update")
			continue
		}

		t, err := renderTemplate(template, entry)
		if err != nil {
			return err
		}

		// Unmarshal the object so we can derive the kind and name.
		object := &unstructured.Unstructured{}
		if err := json.Unmarshal(t.Template.Raw, object); err != nil {
			return err
		}

		gvk := object.GroupVersionKind()

		mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		// Get the resource.
		objectCurr, err := client.Resource(mapping.Resource).Namespace(namespace).Get(object.GetName(), metav1.GetOptions{})
		if err != nil {
			glog.Infof("failed to get resource %s/%s %s", object.GetAPIVersion(), object.GetKind(), object.GetName())
			return err
		}

		objectRaw, err := json.Marshal(objectCurr)
		if err != nil {
			return err
		}

		glog.Infof("current resource: %s", string(objectRaw))

		// Apply the parameters.  Only affect parameters that are defined
		// in the request, so be sure not to apply any defaults as they may
		// cause the resource to do something that was not intended.
		objectRaw, err = patchObject(objectRaw, template.Parameters, entry, false)
		if err != nil {
			return err
		}

		// Commit the resource if it has changed
		objectNew := &unstructured.Unstructured{}
		if err := json.Unmarshal(objectRaw, objectNew); err != nil {
			return err
		}

		if reflect.DeepEqual(objectCurr, objectNew) {
			glog.Infof("resource unchanged")
			continue
		}

		glog.Infof("new resource: %s", string(objectRaw))

		u.resources = append(u.resources, objectNew)
	}

	return nil
}

// run performs asynchronous update tasks.
func (u *Updater) run(entry *registry.Entry) error {
	glog.Info("updating resources")

	namespace, ok := entry.Get(registry.Namespace)
	if !ok {
		return fmt.Errorf("unable to lookup namespace")
	}

	// Prepare the client code
	client := config.Clients().Dynamic()

	for _, resource := range u.resources {
		glog.Infof("updating resource %s/%s %s", resource.GetAPIVersion(), resource.GetKind(), resource.GetName())

		gvk := resource.GroupVersionKind()

		mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		if _, err := client.Resource(mapping.Resource).Namespace(namespace).Update(resource, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// Run performs asynchronous update tasks.
func (u *Updater) Run(entry *registry.Entry) {
	if err := operation.Complete(entry, u.run(entry)); err != nil {
		glog.Infof("failed to delete instance")
	}
}
