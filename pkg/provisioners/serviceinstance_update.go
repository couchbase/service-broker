package provisioners

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/evanphx/json-patch"
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ServiceInstanceUpdater caches various data associated with updating a service instance.
type ServiceInstanceUpdater struct {
	// registry is the instance registry.
	registry *registry.Entry

	// instanceID is the unique instance ID requested by the client.
	instanceID string

	// namespace is the namespace in which the instance resides.
	namespace string

	// request is the incomiong client requesst.
	request *api.UpdateServiceInstanceRequest

	// resources is a list of resources that need to be updated as a result
	// of any required update operations.
	resources []*unstructured.Unstructured
}

// NewServiceInstanceUpdater returns a new controler capable of updaing a service instance.
func NewServiceInstanceUpdater(registry *registry.Entry, instanceID string, request *api.UpdateServiceInstanceRequest) (*ServiceInstanceUpdater, error) {
	namespace, err := GetNamespace(request.Context)
	if err != nil {
		return nil, err
	}

	u := &ServiceInstanceUpdater{
		registry:   registry,
		instanceID: instanceID,
		namespace:  namespace,
		request:    request,
	}

	return u, nil
}

func (u *ServiceInstanceUpdater) PrepareResources() error {
	// Use the cached versions, as the request parameters may not be set.
	planID, ok := u.registry.Get(registry.PlanID)
	if !ok {
		return fmt.Errorf("unable to lookup service instance plan ID")
	}

	// Collate and render our templates.
	glog.Infof("looking up bindings for service %s, plan %s", u.request.ServiceID, planID)

	templateBindings, err := getTemplateBindings(u.request.ServiceID, planID)
	if err != nil {
		return err
	}

	if templateBindings.ServiceInstance == nil {
		return nil
	}

	// Prepare the client code
	client := config.Clients().Dynamic()

	for _, templateName := range templateBindings.ServiceInstance.Templates {
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

		t, err := renderTemplate(template, u.registry, u.request.Parameters)
		if err != nil {
			return err
		}

		if t.Template == nil || t.Template.Raw == nil {
			glog.Info("template not set, ignoring update")
			continue
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
		objectCurr, err := client.Resource(mapping.Resource).Namespace(u.namespace).Get(object.GetName(), metav1.GetOptions{})
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
		for index, parameter := range template.Parameters {
			value, err := resolveParameter(&template.Parameters[index], u.registry, u.request.Parameters, false)
			if err != nil {
				return err
			}

			if value == nil {
				continue
			}

			patches := []string{}

			for _, path := range parameter.Destination.Paths {
				valueJSON, err := json.Marshal(value)
				if err != nil {
					glog.Infof("marshal of value failed: %v", err)
					return err
				}

				patches = append(patches, fmt.Sprintf(`{"op":"add","path":"%s","value":%s}`, path, string(valueJSON)))
			}

			patchSet := "[" + strings.Join(patches, ",") + "]"

			glog.Infof("applying patchset %s", patchSet)

			patch, err := jsonpatch.DecodePatch([]byte(patchSet))
			if err != nil {
				glog.Infof("decode of JSON patch failed: %v", err)
				return err
			}

			objectRaw, err = patch.Apply(objectRaw)
			if err != nil {
				glog.Infof("apply of JSON patch failed: %v", err)
				return err
			}
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
func (u *ServiceInstanceUpdater) run() error {
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

		if _, err := client.Resource(mapping.Resource).Namespace(u.namespace).Update(resource, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// Run performs asynchronous update tasks.
func (u *ServiceInstanceUpdater) Run() {
	if err := operation.Complete(u.registry, u.run()); err != nil {
		glog.Infof("failed to delete instance")
	}
}
