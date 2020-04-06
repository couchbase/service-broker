package util

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
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
	if err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Delete(config.ConfigurationName, metav1.NewDeleteOptions(0)); err != nil {
		t.Fatal(err)
	}
}

// MustCreateServiceBrokerConfig creates the service broker configuration file with a user specified one.
func MustCreateServiceBrokerConfig(t *testing.T, clients client.Clients, config *v1.ServiceBrokerConfig) {
	if _, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Create(config); err != nil {
		t.Fatal(err)
	}
}

// MustUpdateBrokerConfig updates the service broker configuration with a typesafe callback.
func MustUpdateBrokerConfig(t *testing.T, clients client.Clients, callback func(*v1.ServiceBrokerConfig)) {
	config, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Get(config.ConfigurationName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	callback(config)

	if _, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Update(config); err != nil {
		t.Fatal(err)
	}
}

// configurationValidCondition returns the validity condition of a configuration,
// or an error if it dosn't exist.
func configurationValidCondition(config *v1.ServiceBrokerConfig) (bool, error) {
	for _, condition := range config.Status.Conditions {
		if condition.Type == v1.ConfigurationValid {
			return condition.Status == v1.ConditionTrue, nil
		}
	}

	return false, fmt.Errorf("configuration valid condition not present")
}

// MustReplaceBrokerConfig updates the service broker configuration and waits
// for the broker to acquire the write lock and update the configuration to
// make it live.
func MustReplaceBrokerConfig(t *testing.T, clients client.Clients, spec *v1.ServiceBrokerConfigSpec) {
	if err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Delete(config.ConfigurationName, metav1.NewDeleteOptions(0)); err != nil {
		t.Fatal(err)
	}

	configuration := &v1.ServiceBrokerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.ConfigurationName,
		},
		Spec: *spec,
	}

	if _, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Create(configuration); err != nil {
		t.Fatal(err)
	}

	callback := func() bool {
		// Service broker will first check validity and update the resource.
		configuration, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Get(config.ConfigurationName, metav1.GetOptions{})
		if err != nil {
			return false
		}

		if ok, err := configurationValidCondition(configuration); !ok || err != nil {
			return false
		}

		// The config is live when it is set in the config package.
		config.Lock()
		defer config.Unlock()

		c := config.Config()
		if c == nil {
			return false
		}

		return reflect.DeepEqual(&c.Spec, spec)
	}

	if err := WaitFor(callback, configUpdateTimeout); err != nil {
		t.Fatal(err)
	}
}

// MustReplaceBrokerConfigWithInvalidCondition will updata the configuration and
// then ensure that the broker has registered it is invalid.
func MustReplaceBrokerConfigWithInvalidCondition(t *testing.T, clients client.Clients, spec *v1.ServiceBrokerConfigSpec) {
	if err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Delete(config.ConfigurationName, metav1.NewDeleteOptions(0)); err != nil {
		t.Fatal(err)
	}

	configuration := &v1.ServiceBrokerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.ConfigurationName,
		},
		Spec: *spec,
	}

	if _, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Create(configuration); err != nil {
		t.Fatal(err)
	}

	callback := func() bool {
		// Service broker will first check validity and update the resource.
		configuration, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Get(config.ConfigurationName, metav1.GetOptions{})
		if err != nil {
			return false
		}

		if ok, err := configurationValidCondition(configuration); ok || err != nil {
			return false
		}

		return true
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

// MustHaveRegistryEntryWithValue checks a registry entry exists.
func MustHaveRegistryEntryWithValue(t *testing.T, entry *corev1.Secret, key registry.Key, value string) {
	data, ok := entry.Data[string(key)]
	if !ok {
		t.Fatalf("registry missing key %s", key)
	}

	if string(data) != `"`+value+`"` {
		t.Fatalf("registry entry %s, expected %s", data, value)
	}
}

// byteInDictionary checks a character exists in the specified dictionary.
func byteInDictionary(c byte, dictionary string) bool {
	for index := 0; index < len(dictionary); index++ {
		if dictionary[index] == c {
			return true
		}
	}

	return false
}

// MustHaveRegistryEntryPassword checks a registry entry exists and is valid for a password.
func MustHaveRegistryEntryPassword(t *testing.T, entry *corev1.Secret, key registry.Key, length int, dictionary string) {
	data, ok := entry.Data[string(key)]
	if !ok {
		t.Fatalf("registry missing key %s", key)
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}

	if len(value) != length {
		t.Fatalf("registry entry length %d, expected %d", len(value), length)
	}

	for index := 0; index < length; index++ {
		c := value[index]
		if !byteInDictionary(c, dictionary) {
			t.Fatalf("character %v at index %d not in dictionary %s", rune(c), index, dictionary)
		}
	}
}

// haveRegistryEntriesTLS check the key/cert pair exist and are valid, returning the certificate.
func haveRegistryEntriesTLS(entry *corev1.Secret, key, cert registry.Key) (*x509.Certificate, error) {
	keyData, ok := entry.Data[string(key)]
	if !ok {
		return nil, fmt.Errorf("registry missing private key key %s", key)
	}

	certData, ok := entry.Data[string(cert)]
	if !ok {
		return nil, fmt.Errorf("registry missing certificate key %s", cert)
	}

	var keyPEM string
	if err := json.Unmarshal(keyData, &keyPEM); err != nil {
		return nil, err
	}

	var certPEM string
	if err := json.Unmarshal(certData, &certPEM); err != nil {
		return nil, err
	}

	certificate, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, err
	}

	c, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		return nil, err
	}

	return c, nil
}

// MustHaveRegistryEntriesTLS checks that the requested entries corresponding to a certificate
// and key pair exist and they are valid.
func MustHaveRegistryEntriesTLS(t *testing.T, entry *corev1.Secret, key, cert registry.Key) {
	if _, err := haveRegistryEntriesTLS(entry, key, cert); err != nil {
		t.Fatal(err)
	}
}

// MustHaveRegistryEntriesTLSAndVerify checks that the requested entries corresponding to a certificate
// and key pair exist and they are valid against a CA.
func MustHaveRegistryEntriesTLSAndVerify(t *testing.T, entry *corev1.Secret, caCert, key, cert registry.Key, usage x509.ExtKeyUsage) {
	caCertData, ok := entry.Data[string(caCert)]
	if !ok {
		t.Fatalf("registry missing ca certificate key %s", key)
	}

	var caCertPEM string
	if err := json.Unmarshal(caCertData, &caCertPEM); err != nil {
		t.Fatal(err)
	}

	certificate, err := haveRegistryEntriesTLS(entry, key, cert)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM([]byte(caCertPEM)); !ok {
		t.Fatal("unable to add CA certificate to pool")
	}

	options := x509.VerifyOptions{
		Roots: pool,
		KeyUsages: []x509.ExtKeyUsage{
			usage,
		},
	}

	if _, err := certificate.Verify(options); err != nil {
		t.Fatal(err)
	}
}

// MustNotHaveRegistryEntry checks a registry entry doesn't exist.
func MustNotHaveRegistryEntry(t *testing.T, entry *corev1.Secret, key registry.Key) {
	if _, ok := entry.Data[string(key)]; ok {
		t.Fatalf("registry has key %s", key)
	}
}
