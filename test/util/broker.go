package util

import (
	"reflect"
	"testing"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1alpha1"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/registry"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// configUpdateTimeout is how long to wait before declaring a configuration
	// update as failed.
	configUpdateTimeout = 10 * time.Second
)

// MustDeleteServiceBrokerConfig deletes the service broker configuration file.
func MustDeleteServiceBrokerConfig(t *testing.T, clients client.Clients) {
	if err := clients.Broker().BrokerV1alpha1().ServiceBrokerConfigs(Namespace).Delete(config.ConfigurationName, metav1.NewDeleteOptions(0)); err != nil {
		t.Fatal(err)
	}
}

// MustCreateServiceBrokerConfig creates the service broker configuration file with a user specified one.
func MustCreateServiceBrokerConfig(t *testing.T, clients client.Clients, config *v1.ServiceBrokerConfig) {
	if _, err := clients.Broker().BrokerV1alpha1().ServiceBrokerConfigs(Namespace).Create(config); err != nil {
		t.Fatal(err)
	}
}

// MustUpdateBrokerConfig updates the service broker configuration with a typesafe callback.
func MustUpdateBrokerConfig(t *testing.T, clients client.Clients, callback func(*v1.ServiceBrokerConfig)) {
	config, err := clients.Broker().BrokerV1alpha1().ServiceBrokerConfigs(Namespace).Get(config.ConfigurationName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	callback(config)

	if _, err := clients.Broker().BrokerV1alpha1().ServiceBrokerConfigs(Namespace).Update(config); err != nil {
		t.Fatal(err)
	}
}

// MustReplaceBrokerConfig updates the service broker configuration and waits
// for the broker to acquire the write lock and update the configuration to
// make it live.
func MustReplaceBrokerConfig(t *testing.T, clients client.Clients, spec *v1.ServiceBrokerConfigSpec) {
	configuration, err := clients.Broker().BrokerV1alpha1().ServiceBrokerConfigs(Namespace).Get(config.ConfigurationName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	configuration.Spec = *spec

	if _, err := clients.Broker().BrokerV1alpha1().ServiceBrokerConfigs(Namespace).Update(configuration); err != nil {
		t.Fatal(err)
	}

	callback := func() bool {
		config.Lock()
		defer config.Unlock()

		return reflect.DeepEqual(&config.Config().Spec, spec)
	}

	if err := WaitFor(callback, configUpdateTimeout); err != nil {
		t.Fatal(err)
	}
}

// MustGetRegistryEntry returns the registry entry for a service instance.
func MustGetRegistryEntry(t *testing.T, clients client.Clients, rt registry.Type, name string) *corev1.Secret {
	entry, err := clients.Kubernetes().CoreV1().Secrets(Namespace).Get(registry.Name(rt, name), metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	return entry
}

// MustHaveRegistryEntry checks a registry entry exists.
func MustHaveRegistryEntry(t *testing.T, entry *corev1.Secret, key registry.Key, value string) {
	data, ok := entry.Data[string(key)]
	if !ok {
		t.Fatalf("registry missing key %s", key)
	}

	if string(data) != `"`+value+`"` {
		t.Fatalf("registry entry %s, expected %s", data, value)
	}
}

// MustNotHaveRegistryEntry checks a registry entry doesn't exist.
func MustNotHaveRegistryEntry(t *testing.T, entry *corev1.Secret, key registry.Key) {
	if _, ok := entry.Data[string(key)]; ok {
		t.Fatalf("registry has key %s", key)
	}
}
