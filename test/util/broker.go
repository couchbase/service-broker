package util

import (
	"testing"

	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MustDeleteServiceBrokerConfig deletes the service broker configuration file.
func MustDeleteServiceBrokerConfig(t *testing.T, clients client.Clients) {
	if err := clients.Broker().BrokerV1().CouchbaseServiceBrokerConfigs(Namespace).Delete("couchbase-service-broker", metav1.NewDeleteOptions(0)); err != nil {
		t.Fatal(err)
	}
}

// MustCreateServiceBrokerConfig creates the service broker configuration file with a user specified one.
func MustCreateServiceBrokerConfig(t *testing.T, clients client.Clients, config *v1.CouchbaseServiceBrokerConfig) {
	if _, err := clients.Broker().BrokerV1().CouchbaseServiceBrokerConfigs(Namespace).Create(config); err != nil {
		t.Fatal(err)
	}
}

// MustUpdateBrokerConfig updates the service broker configuration with a typesafe callback.
func MustUpdateBrokerConfig(t *testing.T, clients client.Clients, callback func(*v1.CouchbaseServiceBrokerConfig)) {
	config, err := clients.Broker().BrokerV1().CouchbaseServiceBrokerConfigs(Namespace).Get("couchbase-service-broker", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	callback(config)
	if _, err := clients.Broker().BrokerV1().CouchbaseServiceBrokerConfigs(Namespace).Update(config); err != nil {
		t.Fatal(err)
	}
}
