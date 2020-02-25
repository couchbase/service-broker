package config

import (
	"fmt"
	"time"

	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/client"
	informerv1 "github.com/couchbase/service-broker/pkg/generated/informers/externalversions/broker.couchbase.com/v1"

	"k8s.io/client-go/tools/cache"

	"github.com/golang/glog"
)

type configuration struct {
	// clients is the set of clients this instance of the broker uses, by default
	// this will use in-cluster Kubernetes, however may be replaced by fake clients
	// by a test framework.
	clients client.Clients

	// config is the user supplied configuration custom resource.
	config *v1.CouchbaseServiceBrokerConfig

	// token is the API access token.
	token string

	// namespace is the default namespace the broker is running in.
	namespace string

	// ready tells the server that the broker is correctly configured
	// and ready to handle operations.
	ready bool
}

// c is the global configuration struct.
var c *configuration

// createHandler add the service broker configuration when the underlying
// resource is created.
// TODO: This is not atomic.
func createHandler(obj interface{}) {
	brokerConfiguration, ok := obj.(*v1.CouchbaseServiceBrokerConfig)
	if !ok {
		glog.Error("unexpected object type in config add")
		return
	}
	if brokerConfiguration.Name != "couchbase-service-broker" {
		glog.V(1).Info("unexpected object name in config delete:", brokerConfiguration.Name)
		return
	}
	glog.Info("service broker configuration created, service ready")
	c.config = brokerConfiguration
	c.ready = true
}

// updateHandler modifies the service broker configuration when the underlying
// resource updates.
// TODO: This is not atomic.
func updateHandler(oldObj, newObj interface{}) {
	brokerConfiguration, ok := newObj.(*v1.CouchbaseServiceBrokerConfig)
	if !ok {
		glog.Error("unexpected object type in config update")
		return
	}
	if brokerConfiguration.Name != "couchbase-service-broker" {
		glog.V(1).Info("unexpected object name in config update:", brokerConfiguration.Name)
		return
	}
	glog.Info("service broker configuration updated")
	c.config = brokerConfiguration
	c.ready = true
}

// deleteHandler deletes the service broker configuration when the underlying
// resource is deleted.
// TODO: This is not atomic.
func deleteHandler(obj interface{}) {
	brokerConfiguration, ok := obj.(*v1.CouchbaseServiceBrokerConfig)
	if !ok {
		glog.Error("unexpected object type in config delete")
		return
	}
	if brokerConfiguration.Name != "couchbase-service-broker" {
		glog.V(1).Info("unexpected object name in config delete:", brokerConfiguration.Name)
		return
	}
	glog.Info("service broker configuration deleted, service unready")
	c.ready = false
	c.config = nil
}

// Configure initializes global configuration and must be called before starting
// the API service.
func Configure(clients client.Clients, namespace, token string) error {
	glog.Info("configuring service broker")

	// Create the global configuration structure.
	c = &configuration{
		clients:   clients,
		token:     token,
		namespace: namespace,
	}

	handlers := &cache.ResourceEventHandlerFuncs{
		AddFunc:    createHandler,
		UpdateFunc: updateHandler,
		DeleteFunc: deleteHandler,
	}

	informer := informerv1.NewCouchbaseServiceBrokerConfigInformer(clients.Broker(), namespace, time.Minute, nil)
	informer.AddEventHandler(handlers)

	stop := make(chan struct{})
	go informer.Run(stop)
	if !cache.WaitForCacheSync(stop, informer.HasSynced) {
		return fmt.Errorf("service broker config shared informer failed to syncronize")
	}

	return nil
}

// Clients returns a set of Kubernetes clients.
func Clients() client.Clients {
	return c.clients
}

// Config returns the user specified custom resource.
func Config() *v1.CouchbaseServiceBrokerConfig {
	return c.config
}

// Token returns the API bearer token.
func Token() string {
	return c.token
}

// Namespace returns the broker namespace.
func Namespace() string {
	return c.namespace
}

// Ready returns whether the config is valid and the service ready to accept requests.
func Ready() bool {
	return c.ready
}
