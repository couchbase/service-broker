package provisioners

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// serviceInstanceCreator caches various data associated with provisioning.
type serviceInstanceCreator struct {
	// registry is the instance registry.
	registry *registry.Registry

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
func NewServiceInstanceCreator(registry *registry.Registry, instanceID string, request *api.CreateServiceInstanceRequest) (*serviceInstanceCreator, error) {
	namespace, err := getNamespace(request.Context)
	if err != nil {
		return nil, err
	}

	provisioner := &serviceInstanceCreator{
		registry:   registry,
		instanceID: instanceID,
		request:    request,
		namespace:  namespace,
	}
	return provisioner, nil
}

// provisionClusterTLS creates the PKI infrastructure to auto configure TLS.
func (p *serviceInstanceCreator) provisionClusterTLS(object *unstructured.Unstructured) error {
	glog.Info("auto configuring TLS")

	return nil
}

// renderTemplate applies any requested parameters to the template.
func (p *serviceInstanceCreator) renderTemplate(template *v1.CouchbaseServiceBrokerConfigTemplate) error {
	t, err := renderTemplate(template, p.instanceID, p.request.Parameters)
	if err != nil {
		return err
	}
	p.templates = append(p.templates, t)
	return nil
}

// createResource instantiates rendered template resources.
func (p *serviceInstanceCreator) createResource(template *v1.CouchbaseServiceBrokerConfigTemplate) error {
	// Unmarshal into instructured JSON.
	object := &unstructured.Unstructured{}
	if err := json.Unmarshal(template.Template.Raw, object); err != nil {
		glog.Errorf("unmarshal of template failed: %v", err)
		return err
	}

	glog.Infof("creating resource %s/%s %s", object.GetAPIVersion(), object.GetKind(), object.GetName())

	// First we need to set up owner references so that we can garbage collect the
	// cluster easily.
	registryEntry, err := p.registry.Get(registry.ServiceInstanceRegistryName(p.instanceID))
	if err != nil {
		glog.Errorf("failed to get service instance registry entry: %v", err)
		return err
	}
	ownerReference := registryEntry.GetOwnerReference()
	object.SetOwnerReferences([]metav1.OwnerReference{ownerReference})

	// Prepare the client code
	gvk := object.GroupVersionKind()
	mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}
	client := config.Clients().Dynamic()

	// Do any implicit resource patching that the service broker offers.
	if object.GetKind() == "CouchbaseCluster" {
		glog.Info("detected CouchbaseCluster resource, performing auto config tasks")

		if err := p.provisionClusterTLS(object); err != nil {
			return err
		}
	}

	// Create the object
	if _, err := client.Resource(mapping.Resource).Namespace(p.namespace).Create(object, metav1.CreateOptions{}); err != nil {
		// When the object already exists and it is marked as a singleton we need to
		// update the owner references to include this new serivce instance so it
		// will not be garbage collected when an existing service instance is removed.
		if errors.IsAlreadyExists(err) && template.Singleton {
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

// gatherMetadata extracts any derived metadata from the supplied set of templates.
func (p *serviceInstanceCreator) gatherMetadata() error {
	registryEntry, err := p.registry.Get(registry.ServiceInstanceRegistryName(p.instanceID))
	if err != nil {
		return err
	}

	for _, template := range p.templates {
		// Unmarshal into instructured JSON.
		object := &unstructured.Unstructured{}
		if err := json.Unmarshal(template.Template.Raw, object); err != nil {
			glog.Errorf("unmarshal of template failed: %v", err)
			return err
		}

		switch object.GetKind() {
		case "CouchbaseCluster":
			// Couchbase clusters set the dashboard URI.  This is communicated back to
			// the client when a service instance is initially provisioned.
			dashboard := fmt.Sprintf("https://%s.%s.svc:18091", object.GetName(), p.namespace)
			if domain, found, _ := unstructured.NestedString(object.Object, "spec", "networking", "dns", "domain"); found {
				dashboard = fmt.Sprintf("https://console.%s:18091", domain)
			}
			if err := registryEntry.Set(registry.RegistryKeyDashboardURL, dashboard); err != nil {
				return err
			}
		}
	}

	return nil
}

// prepareServiceInstance does provisional synchronous tasks before provisioning.  This does
// basic template collection and rendering.
func (p *serviceInstanceCreator) PrepareServiceInstance() error {
	// Collate and render our templates.
	glog.Infof("looking up bindings for service %s, plan %s", p.request.ServiceID, p.request.PlanID)
	templateBindings, err := getTemplateBindings(p.request.ServiceID, p.request.PlanID)
	if err != nil {
		return err
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

	if err := p.gatherMetadata(); err != nil {
		return err
	}

	return nil
}

// Run performs asynchronous creation tasks.
func (p *serviceInstanceCreator) Run() error {
	glog.Infof("creating resources")
	for _, template := range p.templates {
		if err := p.createResource(template); err != nil {
			return err
		}
	}

	return nil
}
