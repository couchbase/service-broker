package util

import (
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/client"
	brokerclient "github.com/couchbase/service-broker/pkg/generated/clientset/versioned"
	brokerclientfake "github.com/couchbase/service-broker/pkg/generated/clientset/versioned/fake"

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
	// DefaultBrokerConfig is a minimal service broker config to allow initialization.
	DefaultBrokerConfig = &v1.CouchbaseServiceBrokerConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.Group,
			Kind:       v1.ServiceBrokerConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "couchbase-service-broker",
			Namespace: Namespace,
		},
	}

	// defaultBrokerObjects is a global list of objects that the fake client will
	// contain.
	// TODO: These should be like fixtures.
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
	broker     brokerclient.Interface
	dynamic    dynamicclient.Interface
	mapper     meta.RESTMapper
}

// NewClients creates a new set of fake clients for use by testing.
func NewClients() (client.Clients, error) {
	kubernetes := kubernetesclientfake.NewSimpleClientset()
	broker := brokerclientfake.NewSimpleClientset(defaultBrokerObjects...)
	dynamic := dynamicclientfake.NewSimpleDynamicClient(scheme.Scheme)

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

// Kubernetes returns a typed client for Kubernetes resources.
func (c *clientsImpl) Kubernetes() kubernetesclient.Interface {
	return c.kubernetes
}

// Broker returns a typed client for service broker resources.
func (c *clientsImpl) Broker() brokerclient.Interface {
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
