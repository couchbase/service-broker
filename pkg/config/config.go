package config

import (
	"fmt"
	"sync"
	"time"

	informerv1 "github.com/couchbase/service-broker/generated/informers/externalversions/servicebroker/v1alpha1"
	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/log"
	"github.com/golang/glog"

	"k8s.io/client-go/tools/cache"
)

const (
	// ConfigurationName is the configuration resource name.
	ConfigurationName = "couchbase-service-broker"
)

type configuration struct {
	// clients is the set of clients this instance of the broker uses, by default
	// this will use in-cluster Kubernetes, however may be replaced by fake clients
	// by a test framework.
	clients client.Clients

	// config is the user supplied configuration custom resource.
	config *v1.ServiceBrokerConfig

	// token is the API access token.
	token string

	// namespace is the default namespace the broker is running in.
	namespace string

	// lock is used to remove races around the use of the context.
	// The context can be read by many, but can only be written
	// by one when there are no readers.
	// Updates must appear atomic so handlers should hold the read
	// lock while processing a request.
	lock sync.RWMutex
}

// c is the global configuration struct.
var c *configuration

// createHandler add the service broker configuration when the underlying
// resource is created.
func createHandler(obj interface{}) {
	brokerConfiguration, ok := obj.(*v1.ServiceBrokerConfig)
	if !ok {
		glog.Error("unexpected object type in config add")
		return
	}

	if brokerConfiguration.Name != ConfigurationName {
		glog.V(log.LevelDebug).Info("unexpected object name in config delete:", brokerConfiguration.Name)
		return
	}

	glog.Info("service broker configuration created, service ready")

	c.lock.Lock()
	c.config = brokerConfiguration
	c.lock.Unlock()
}

// updateHandler modifies the service broker configuration when the underlying
// resource updates.
func updateHandler(oldObj, newObj interface{}) {
	brokerConfiguration, ok := newObj.(*v1.ServiceBrokerConfig)
	if !ok {
		glog.Error("unexpected object type in config update")
		return
	}

	if brokerConfiguration.Name != ConfigurationName {
		glog.V(log.LevelDebug).Info("unexpected object name in config update:", brokerConfiguration.Name)
		return
	}

	glog.Info("service broker configuration updated")

	c.lock.Lock()
	c.config = brokerConfiguration
	c.lock.Unlock()
}

// deleteHandler deletes the service broker configuration when the underlying
// resource is deleted.
func deleteHandler(obj interface{}) {
	brokerConfiguration, ok := obj.(*v1.ServiceBrokerConfig)
	if !ok {
		glog.Error("unexpected object type in config delete")
		return
	}

	if brokerConfiguration.Name != ConfigurationName {
		glog.V(log.LevelDebug).Info("unexpected object name in config delete:", brokerConfiguration.Name)
		return
	}

	glog.Info("service broker configuration deleted, service unready")

	c.lock.Lock()
	c.config = nil
	c.lock.Unlock()
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

	informer := informerv1.NewServiceBrokerConfigInformer(clients.Broker(), namespace, time.Minute, nil)
	informer.AddEventHandler(handlers)

	stop := make(chan struct{})

	go informer.Run(stop)

	if !cache.WaitForCacheSync(stop, informer.HasSynced) {
		return fmt.Errorf("service broker config shared informer failed to syncronize")
	}

	return nil
}

// Lock puts a read lock on the configuration during the lifetime
// of a request.
func Lock() {
	c.lock.RLock()
}

// Unlock releases the read lock on the configuration after a
// request has completed.
func Unlock() {
	c.lock.RUnlock()
}

// Clients returns a set of Kubernetes clients.
func Clients() client.Clients {
	return c.clients
}

// Config returns the user specified custom resource.
func Config() *v1.ServiceBrokerConfig {
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
