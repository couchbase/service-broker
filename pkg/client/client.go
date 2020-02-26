package client

import (
	"github.com/couchbase/service-broker/generated/clientset/versioned"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// Clients provides an abstraction layer for service broker Kubernetes client interfaces.
type Clients interface {
	// Kubernetes returns a typed client for Kubernetes resources.
	Kubernetes() kubernetes.Interface

	// Broker returns a typed client for service broker resources.
	Broker() versioned.Interface

	// Dynamic returns a dynamic client for Kubernetes resources.
	Dynamic() dynamic.Interface

	// RESTMapper returns a REST mapps for Kubernetes resources, able to translate
	// a resource type into a API endpoint.
	RESTMapper() meta.RESTMapper
}

// clientsImpl implements the default Kubernetes client interface using in-cluster configuration.
type clientsImpl struct {
	config     *rest.Config
	kubernetes kubernetes.Interface
	broker     versioned.Interface
	dynamic    dynamic.Interface
	mapper     meta.RESTMapper
}

// New returns a new set of clients for use in-cluster.
// This requires that the container has an API service token mounted.
func New() (Clients, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	kubernetes, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	broker, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamic, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	groupresources, err := restmapper.GetAPIGroupResources(kubernetes.Discovery())
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupresources)

	clients := &clientsImpl{
		config:     config,
		kubernetes: kubernetes,
		broker:     broker,
		dynamic:    dynamic,
		mapper:     mapper,
	}

	return clients, nil
}

// Kubernetes returns a typed client for Kubernetes resources.
func (c *clientsImpl) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

// Broker returns a typed client for service broker resources.
func (c *clientsImpl) Broker() versioned.Interface {
	return c.broker
}

// Dynamic returns a dynamic client for Kubernetes resources.
func (c *clientsImpl) Dynamic() dynamic.Interface {
	return c.dynamic
}

// RESTMapper returns a REST mapps for Kubernetes resources, able to translate
// a resource type into a API endpoint.
func (c *clientsImpl) RESTMapper() meta.RESTMapper {
	return c.mapper
}
