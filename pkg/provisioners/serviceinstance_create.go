package provisioners

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/golang/glog"

	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ServiceInstanceCreator caches various data associated with provisioning.
type ServiceInstanceCreator struct {
	// registry is the instance registry.
	registry *registry.Entry

	// instanceID is the unique instance ID requested by the client.
	instanceID string

	// request is the raw request made by the client.
	request *api.CreateServiceInstanceRequest

	// namespace is the namespace to provision resources into.
	namespace string

	// templates contains the list of rendered templates.  Used as a cache
	// between the synchronous and asynchronous phases of provisioning.
	templates []*v1.CouchbaseServiceBrokerConfigTemplate
}

// NewServiceInstanceCreator initializes all the data required for
// provisioning a service instance.
func NewServiceInstanceCreator(registry *registry.Entry, instanceID string, request *api.CreateServiceInstanceRequest) (*ServiceInstanceCreator, error) {
	namespace, err := GetNamespace(request.Context)
	if err != nil {
		return nil, err
	}

	provisioner := &ServiceInstanceCreator{
		registry:   registry,
		instanceID: instanceID,
		request:    request,
		namespace:  namespace,
	}

	return provisioner, nil
}

// renderTemplate applies any requested parameters to the template.
func (p *ServiceInstanceCreator) renderTemplate(template *v1.CouchbaseServiceBrokerConfigTemplate) error {
	t, err := renderTemplate(template, p.registry, p.request.Parameters)
	if err != nil {
		return err
	}

	p.templates = append(p.templates, t)

	return nil
}

// createResource instantiates rendered template resources.
func (p *ServiceInstanceCreator) createResource(template *v1.CouchbaseServiceBrokerConfigTemplate) error {
	if template.Template == nil {
		glog.Infof("template has no associated object, skipping")
		return nil
	}

	// Unmarshal into instructured JSON.
	object := &unstructured.Unstructured{}
	if err := json.Unmarshal(template.Template.Raw, object); err != nil {
		glog.Errorf("unmarshal of template failed: %v", err)
		return err
	}

	glog.Infof("creating resource %s/%s %s", object.GetAPIVersion(), object.GetKind(), object.GetName())

	// First we need to set up owner references so that we can garbage collect the
	// cluster easily.
	ownerReference := p.registry.GetOwnerReference()
	object.SetOwnerReferences([]metav1.OwnerReference{ownerReference})

	// Prepare the client code
	gvk := object.GroupVersionKind()

	mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	client := config.Clients().Dynamic()

	// Create the object
	if _, err := client.Resource(mapping.Resource).Namespace(p.namespace).Create(object, metav1.CreateOptions{}); err != nil {
		// When the object already exists and it is marked as a singleton we need to
		// update the owner references to include this new serivce instance so it
		// will not be garbage collected when an existing service instance is removed.
		if k8s_errors.IsAlreadyExists(err) && template.Singleton {
			glog.Infof("singleton resource already exists, adding owner reference")

			existing, err := client.Resource(mapping.Resource).Namespace(p.namespace).Get(object.GetName(), metav1.GetOptions{})
			if err != nil {
				glog.Errorf("unable to get existing singleton resource: %v", err)
				return err
			}

			owners, found, err := unstructured.NestedSlice(existing.Object, "metadata", "ownerReferences")
			if err != nil {
				glog.Errorf("unable to get owner references for object: %v", err)
				return err
			}

			if !found {
				glog.Errorf("owner references unexpectedly missing")
				return fmt.Errorf("owner references unexpectedly missing")
			}

			unstructuredOwnerReference, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ownerReference)
			if err != nil {
				glog.Errorf("failed to convert owner reference to unstructured: %v", err)
				return err
			}

			owners = append(owners, unstructuredOwnerReference)
			if err := unstructured.SetNestedSlice(existing.Object, owners, "metadata", "ownerReferences"); err != nil {
				glog.Errorf("unable to patch owner references for object: %v", err)
				return err
			}

			if _, err := client.Resource(mapping.Resource).Namespace(p.namespace).Update(existing, metav1.UpdateOptions{}); err != nil {
				glog.Errorf("unable to update singleton resource owner references: %v", err)
				return err
			}

			return nil
		}

		return err
	}

	return nil
}

// prepareServiceInstance does provisional synchronous tasks before provisioning.  This does
// basic template collection and rendering.
func (p *ServiceInstanceCreator) PrepareServiceInstance() error {
	glog.Infof("looking up bindings for service %s, plan %s", p.request.ServiceID, p.request.PlanID)

	// Collate and render our templates.
	templateBindings, err := getTemplateBindings(p.request.ServiceID, p.request.PlanID)
	if err != nil {
		return err
	}

	if templateBindings.ServiceInstance == nil {
		return nil
	}

	// Render any parameters.  As they are not associated with any template they
	// can only ever be committed to the registry.
	glog.Infof("rendering parameters for binding %s", templateBindings.Name)

	for index := range templateBindings.ServiceInstance.Parameters {
		parameter := &templateBindings.ServiceInstance.Parameters[index]

		value, err := resolveParameter(parameter, p.registry, p.request.Parameters, true)
		if err != nil {
			return err
		}

		if parameter.Destination.Registry == nil {
			return errors.NewConfigurationError("parameter %s must have a registry destination", parameter.Name)
		}

		glog.Infof("setting registry entry %s to %v", *parameter.Destination.Registry, value)

		if err := p.registry.SetJSONUser(*parameter.Destination.Registry, value); err != nil {
			return err
		}
	}

	glog.Infof("rendering templates for binding %s", templateBindings.Name)

	for _, templateName := range templateBindings.ServiceInstance.Templates {
		template, err := getTemplate(templateName)
		if err != nil {
			return err
		}

		if err = p.renderTemplate(template); err != nil {
			return err
		}
	}

	return nil
}

// run performs asynchronous creation tasks.
func (p *ServiceInstanceCreator) run() error {
	glog.Infof("creating resources")

	for _, template := range p.templates {
		if err := p.createResource(template); err != nil {
			return err
		}
	}

	return nil
}

// Run performs asynchronous creation tasks.
func (p *ServiceInstanceCreator) Run() {
	if err := operation.Complete(p.registry, p.run()); err != nil {
		glog.Errorf("failed to delete instance")
	}
}
