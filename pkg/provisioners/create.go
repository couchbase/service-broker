package provisioners

import (
	"encoding/json"
	"fmt"

	v1 "github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1alpha1"
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

// Creator caches various data associated with provisioning.
type Creator struct {
	resourceType ResourceType

	// templates contains the list of rendered templates.  Used as a cache
	// between the synchronous and asynchronous phases of provisioning.
	templates []*v1.CouchbaseServiceBrokerConfigTemplate
}

// NewCreator initializes all the data required for
// provisioning a service instance.
func NewCreator(resourceType ResourceType) (*Creator, error) {
	provisioner := &Creator{
		resourceType: resourceType,
	}

	return provisioner, nil
}

// renderTemplate applies any requested parameters to the template.
func (p *Creator) renderTemplate(template *v1.CouchbaseServiceBrokerConfigTemplate, entry *registry.Entry) error {
	t, err := renderTemplate(template, entry)
	if err != nil {
		return err
	}

	p.templates = append(p.templates, t)

	return nil
}

// createResource instantiates rendered template resources.
func (p *Creator) createResource(template *v1.CouchbaseServiceBrokerConfigTemplate, entry *registry.Entry) error {
	if template.Template == nil || template.Template.Raw == nil {
		glog.Infof("template has no associated object, skipping")
		return nil
	}

	// Unmarshal into instructured JSON.
	object := &unstructured.Unstructured{}
	if err := json.Unmarshal(template.Template.Raw, object); err != nil {
		glog.Infof("unmarshal of template failed: %v", err)
		return err
	}

	glog.Infof("creating resource %s/%s %s", object.GetAPIVersion(), object.GetKind(), object.GetName())

	// First we need to set up owner references so that we can garbage collect the
	// cluster easily.
	ownerReference := entry.GetOwnerReference()
	object.SetOwnerReferences([]metav1.OwnerReference{ownerReference})

	// Prepare the client code
	gvk := object.GroupVersionKind()

	mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	namespace, ok, err := entry.GetString(registry.Namespace)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("unable to lookup namespace")
	}

	client := config.Clients().Dynamic()

	// Create the object
	if _, err := client.Resource(mapping.Resource).Namespace(namespace).Create(object, metav1.CreateOptions{}); err != nil {
		// When the object already exists and it is marked as a singleton we need to
		// update the owner references to include this new serivce instance so it
		// will not be garbage collected when an existing service instance is removed.
		if k8s_errors.IsAlreadyExists(err) && template.Singleton {
			glog.Infof("singleton resource already exists, adding owner reference")

			existing, err := client.Resource(mapping.Resource).Namespace(namespace).Get(object.GetName(), metav1.GetOptions{})
			if err != nil {
				glog.Infof("unable to get existing singleton resource: %v", err)
				return err
			}

			owners, found, err := unstructured.NestedSlice(existing.Object, "metadata", "ownerReferences")
			if err != nil {
				glog.Infof("unable to get owner references for object: %v", err)
				return err
			}

			if !found {
				glog.Infof("owner references unexpectedly missing")
				return fmt.Errorf("owner references unexpectedly missing")
			}

			unstructuredOwnerReference, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ownerReference)
			if err != nil {
				glog.Infof("failed to convert owner reference to unstructured: %v", err)
				return err
			}

			owners = append(owners, unstructuredOwnerReference)
			if err := unstructured.SetNestedSlice(existing.Object, owners, "metadata", "ownerReferences"); err != nil {
				glog.Infof("unable to patch owner references for object: %v", err)
				return err
			}

			if _, err := client.Resource(mapping.Resource).Namespace(namespace).Update(existing, metav1.UpdateOptions{}); err != nil {
				glog.Infof("unable to update singleton resource owner references: %v", err)
				return err
			}

			return nil
		}

		return err
	}

	return nil
}

// Prepare does provisional synchronous tasks before provisioning.  This does
// basic template collection and rendering.
func (p *Creator) Prepare(entry *registry.Entry) error {
	serviceID, ok, err := entry.GetString(registry.ServiceID)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("unable to lookup service ID")
	}

	planID, ok, err := entry.GetString(registry.PlanID)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("unable to lookup plan ID")
	}

	glog.Infof("looking up bindings for service %s, plan %s", serviceID, planID)

	// Collate and render our templates.
	templates, err := getTemplateBinding(p.resourceType, serviceID, planID)
	if err != nil {
		return err
	}

	// Render any parameters.  As they are not associated with any template they
	// can only ever be committed to the registry.
	glog.Infof("rendering parameters for binding")

	for index := range templates.Parameters {
		parameter := &templates.Parameters[index]

		glog.Infof("rendering parameter %s", parameter.Name)

		value, err := resolveTemplateParameter(parameter, entry, true)
		if err != nil {
			return err
		}

		if value == nil {
			continue
		}

		for _, destination := range parameter.Destinations {
			if destination.Registry == nil {
				return errors.NewConfigurationError("parameter %s must have a registry destination", parameter.Name)
			}

			if err := entry.SetUser(*destination.Registry, value); err != nil {
				return err
			}
		}
	}

	glog.Infof("rendering templates for binding")

	for _, templateName := range templates.Templates {
		template, err := getTemplate(templateName)
		if err != nil {
			return err
		}

		if err = p.renderTemplate(template, entry); err != nil {
			return err
		}
	}

	return nil
}

// run performs asynchronous creation tasks.
func (p *Creator) run(entry *registry.Entry) error {
	glog.Infof("creating resources")

	for _, template := range p.templates {
		if err := p.createResource(template, entry); err != nil {
			return err
		}
	}

	return nil
}

// Run performs asynchronous creation tasks.
func (p *Creator) Run(entry *registry.Entry) {
	if err := operation.Complete(entry, p.run(entry)); err != nil {
		glog.Infof("failed to create instance: %v", err)
	}
}
