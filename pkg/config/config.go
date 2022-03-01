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

package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	informerv1 "github.com/couchbase/service-broker/generated/informers/externalversions/servicebroker/v1alpha1"
	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/log"
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	// ConfigurationNameDefault is the default configuration name.
	ConfigurationNameDefault = "couchbase-service-broker"
)

var (
	// ConfigurationName is the configuration resource name.
	// This has a default for the benfit of testing, it is overidden
	// by flags for the main binary.
	ConfigurationName = ConfigurationNameDefault

	// ErrCacheSync is raised when a shared informer failed to synchronize.
	ErrCacheSync = errors.New("cache synchronization error")
)

type configuration struct {
	// clients is the set of clients this instance of the broker uses, by default
	// this will use in-cluster Kubernetes, however may be replaced by fake clients
	// by a test framework.
	clients client.Clients

	// config is the user supplied configuration custom resource.
	config *v1.ServiceBrokerConfig

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

	if err := updateStatus(brokerConfiguration); err != nil {
		glog.Info("service broker configuration invalid, see resource status for details")
		glog.V(1).Info(err)

		c.lock.Lock()
		defer c.lock.Unlock()

		c.config = nil

		return
	}

	glog.Info("service broker configuration created, service ready")

	if glog.V(1) {
		object, err := json.Marshal(brokerConfiguration)
		if err == nil {
			glog.V(1).Info(string(object))
		}
	}

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

	if err := updateStatus(brokerConfiguration); err != nil {
		glog.Info("service broker configuration invalid, see resource status for details")
		glog.V(1).Info(err)

		c.lock.Lock()
		defer c.lock.Unlock()

		c.config = nil

		return
	}

	glog.Info("service broker configuration updated")

	if glog.V(1) {
		object, err := json.Marshal(brokerConfiguration)
		if err == nil {
			glog.V(1).Info(string(object))
		}
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.config = brokerConfiguration
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
func Configure(clients client.Clients, namespace string) error {
	glog.Info("configuring service broker")

	// Create the global configuration structure.
	c = &configuration{
		clients: clients,
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
		return fmt.Errorf("%w: service broker config shared informer failed to syncronize", ErrCacheSync)
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

// updateStatus runs any analysis on the confiuration, makes and commits any modifications.
// In particular this allows the status to say you have made a configuration error.
// A returned error means don't accept the configuration, set to nil so the service broker
// reports unready and doesn't serve any API requests.
func updateStatus(config *v1.ServiceBrokerConfig) error {
	var rerr error

	// Assume the configuration is valid, then modify if an error
	// has occurred, finally retain the transition time if an existing
	// condition exists and it has the same status.
	validCondition := v1.ServiceBrokerConfigCondition{
		Type:   v1.ConfigurationValid,
		Status: v1.ConditionTrue,
		LastTransitionTime: metav1.Time{
			Time: time.Now(),
		},
		Reason: "ValidationSucceeded",
	}

	if err := validate(config); err != nil {
		validCondition.Status = v1.ConditionFalse
		validCondition.Reason = "ValidationFailed"
		validCondition.Message = err.Error()

		rerr = err
	}

	for _, condition := range config.Status.Conditions {
		if condition.Type == v1.ConfigurationValid {
			if condition.Status == validCondition.Status {
				validCondition.LastTransitionTime = condition.LastTransitionTime
			}

			break
		}
	}

	// Update the status if it has been modified.
	status := v1.ServiceBrokerConfigStatus{
		Conditions: []v1.ServiceBrokerConfigCondition{
			validCondition,
		},
	}

	if reflect.DeepEqual(config.Status, status) {
		return rerr
	}

	newConfig := config.DeepCopy()
	newConfig.Status = status

	if _, err := c.clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(newConfig.Namespace).Update(context.TODO(), newConfig, metav1.UpdateOptions{}); err != nil {
		glog.Infof("failed to update service broker configuration status: %v", err)
		return rerr
	}

	return rerr
}
