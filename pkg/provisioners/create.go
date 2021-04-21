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
	"encoding/json"
	"fmt"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/operation"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/golang/glog"

	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Creator caches various data associated with provisioning.
type Creator struct {
	resourceType ResourceType

	// templates contains the list of rendered templates.  Used as a cache
	// between the synchronous and asynchronous phases of provisioning.
	templates []*v1.ConfigurationTemplate
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
func (p *Creator) renderTemplate(template *v1.ConfigurationTemplate, entry *registry.Entry) error {
	t, err := renderTemplate(template, entry, nil)
	if err != nil {
		return err
	}

	p.templates = append(p.templates, t)

	return nil
}

// createResource instantiates rendered template resources.
func (p *Creator) createResource(template *v1.ConfigurationTemplate, entry *registry.Entry) error {
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

	// To support updates, knowing that Kubernetes can modify resources,
	// we must annotate the resource with the deterministic representation
	// of the resource as defined by the template rendering.
	resourceJSON, err := json.Marshal(object)
	if err != nil {
		return err
	}

	annotations, ok, err := unstructured.NestedStringMap(object.Object, "metadata", "annotations")
	if err != nil {
		return err
	}

	if !ok {
		annotations = map[string]string{}
	}

	annotations[v1.ResourceAnnotation] = string(resourceJSON)
	if err := unstructured.SetNestedStringMap(object.Object, annotations, "metadata", "annotations"); err != nil {
		return err
	}

	// First we need to set up owner references so that we can garbage collect the
	// cluster easily.  These should not be considered as part of the cached annotation
	// defined above.
	ownerReference := entry.GetOwnerReference()
	object.SetOwnerReferences([]metav1.OwnerReference{ownerReference})

	// Prepare the client code
	gvk := object.GroupVersionKind()

	mapping, err := config.Clients().RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// The namespace defaults to that configured in the object, if not
	// specified we use the namespace defined in the context (where the
	// service instance or binding is created).
	namespace := object.GetNamespace()
	if namespace == "" {
		n, ok, err := entry.GetString(registry.Namespace)
		if err != nil {
			return err
		}

		if !ok {
			return fmt.Errorf("%w: unable to lookup namespace", ErrRegistryEntryMissing)
		}

		namespace = n
	}

	glog.Infof("using namespace %s", namespace)

	// Create the object
	client := config.Clients().Dynamic()

	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		_, err = client.Resource(mapping.Resource).Create(object, metav1.CreateOptions{})
	} else {
		_, err = client.Resource(mapping.Resource).Namespace(namespace).Create(object, metav1.CreateOptions{})
	}

	if err != nil {
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
				return fmt.Errorf("%w: owner references unexpectedly missing", ErrResourceAttributeMissing)
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

			if mapping.Scope.Name() == meta.RESTScopeNameRoot {
				_, err = client.Resource(mapping.Resource).Update(existing, metav1.UpdateOptions{})
			} else {
				_, err = client.Resource(mapping.Resource).Namespace(namespace).Update(existing, metav1.UpdateOptions{})
			}

			if err != nil {
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
		return fmt.Errorf("%w: unable to lookup service ID", ErrResourceReferenceMissing)
	}

	planID, ok, err := entry.GetString(registry.PlanID)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("%w: unable to lookup plan ID", ErrResourceReferenceMissing)
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

	for _, registry := range templates.Registry {
		value, err := renderTemplateString(registry.Value, entry, nil)
		if err != nil {
			return err
		}

		if value == nil {
			continue
		}

		if err := entry.SetUser(registry.Name, value); err != nil {
			return err
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
