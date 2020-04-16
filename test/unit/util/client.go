// Copyright 2020 Couchbase, Inc.
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

package util

import (
	"fmt"
	"testing"

	"github.com/couchbase/service-broker/generated/clientset/servicebroker"
	servicebrokerfake "github.com/couchbase/service-broker/generated/clientset/servicebroker/fake"
	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/client"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicclient "k8s.io/client-go/dynamic"
	dynamicclientfake "k8s.io/client-go/dynamic/fake"
	kubernetesclient "k8s.io/client-go/kubernetes"
	kubernetesclientfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/restmapper"
)

var (
	// resources is a list of API resources that the broker knows about.
	// The broker will use the discovery client to map from a dynamic object's
	// api version and kind, into a group, version and resource for use with
	// the dynamic client.  Any resources that are rendered should be registered
	// here.
	// NOTE: Ideally there would be a way to extract this information from the
	// scheme, but there doesn't seem to be.
	resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Namespaced: true, Group: "", Version: "v1", Kind: "Pod"},
			},
		},
	}

	// DefaultBrokerConfig is a minimal service broker config to allow initialization.
	DefaultBrokerConfig = &v1.ServiceBrokerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "couchbase-service-broker",
			Namespace: Namespace,
		},
	}

	// defaultBrokerObjects is a global list of objects that the fake client will
	// contain.
	defaultBrokerObjects = []runtime.Object{
		DefaultBrokerConfig,
	}
)

// clientsImpl implements the Kubernetes clients interface for testing purposes.
// This uses fake clients as a drop-in replacement within the code service broker
// code.  This allows us full control over what resources are already populated in
// Kubernetes and visibility of what gets generated.  In doing so we have full
// control over behaviour driven development.
type clientsImpl struct {
	kubernetes kubernetesclient.Interface
	broker     servicebroker.Interface
	dynamic    dynamicclient.Interface
	mapper     meta.RESTMapper
}

// NewClients creates a new set of fake clients for use by testing.
func NewClients() (client.Clients, error) {
	// Create all the clients, seeding with default objects.
	kubernetes := kubernetesclientfake.NewSimpleClientset()
	broker := servicebrokerfake.NewSimpleClientset(defaultBrokerObjects...)
	dynamic := dynamicclientfake.NewSimpleDynamicClient(scheme.Scheme)

	// Initialize the discovery API.
	kubernetes.Fake.Resources = resources

	// Initialize the REST mapper once the discover interface is populated.
	groupresources, err := restmapper.GetAPIGroupResources(kubernetes.Discovery())
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupresources)

	clients := &clientsImpl{
		kubernetes: kubernetes,
		broker:     broker,
		dynamic:    dynamic,
		mapper:     mapper,
	}

	return clients, nil
}

// ResetClients resets clients back to a pristine state.  The kubernetes and dynamic
// clients have their own object caches, therefore objects created with the kubernetes
// client cannot be cleaned up using the dynamic client.  This also has implications
// for the service broker as a whole.  Clients are reset by just overwriting them,
// therefore at any one time only the top level clients interface is valid.  Objects
// contained within must not be cached e.g. references to the kubernetes client for
// example.
func ResetClients(clients client.Clients) error {
	c, ok := clients.(*clientsImpl)
	if !ok {
		return fmt.Errorf("wrong client type")
	}

	// Create all the clients, seeding with default objects.
	kubernetes := kubernetesclientfake.NewSimpleClientset()
	dynamic := dynamicclientfake.NewSimpleDynamicClient(scheme.Scheme)

	// Initialize the discovery API.
	kubernetes.Fake.Resources = resources

	// Initialize the REST mapper once the discover interface is populated.
	groupresources, err := restmapper.GetAPIGroupResources(kubernetes.Discovery())
	if err != nil {
		return err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupresources)

	c.kubernetes = kubernetes
	c.dynamic = dynamic
	c.mapper = mapper

	return nil
}

// ResetDynamicClient deletes any objects created by template rendering.
// This simulates garbage collection when a registry item is deleted.
func MustResetDynamicClient(t *testing.T, clients client.Clients) {
	c, ok := clients.(*clientsImpl)
	if !ok {
		t.Fatal("wrong client type")
	}

	dynamic := dynamicclientfake.NewSimpleDynamicClient(scheme.Scheme)

	c.dynamic = dynamic
}

// Kubernetes returns a typed client for Kubernetes resources.
func (c *clientsImpl) Kubernetes() kubernetesclient.Interface {
	return c.kubernetes
}

// Broker returns a typed client for service broker resources.
func (c *clientsImpl) Broker() servicebroker.Interface {
	return c.broker
}

// Dynamic returns a dynamic client for Kubernetes resources.
func (c *clientsImpl) Dynamic() dynamicclient.Interface {
	return c.dynamic
}

// RESTMapper returns a REST mapps for Kubernetes resources, able to translate
// a resource type into a API endpoint.
func (c *clientsImpl) RESTMapper() meta.RESTMapper {
	return c.mapper
}
