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

package client

import (
	"github.com/couchbase/service-broker/generated/clientset/servicebroker"

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
	Broker() servicebroker.Interface

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
	broker     servicebroker.Interface
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

	broker, err := servicebroker.NewForConfig(config)
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
func (c *clientsImpl) Broker() servicebroker.Interface {
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
