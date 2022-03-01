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

package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/pkg/util"

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
	if err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Delete(context.TODO(), config.ConfigurationName, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
}

// MustCreateServiceBrokerConfig creates the service broker configuration file with a user specified one.
func MustCreateServiceBrokerConfig(t *testing.T, clients client.Clients, config *v1.ServiceBrokerConfig) {
	if _, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Create(context.TODO(), config, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
}

// MustUpdateBrokerConfig updates the service broker configuration with a typesafe callback.
func MustUpdateBrokerConfig(t *testing.T, clients client.Clients, callback func(*v1.ServiceBrokerConfig)) {
	config, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Get(context.TODO(), config.ConfigurationName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	callback(config)

	if _, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Update(context.TODO(), config, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
}

// configurationValidCondition returns the validity condition of a configuration,
// or an error if it dosn't exist.
func configurationValidCondition(config *v1.ServiceBrokerConfig, status v1.ConditionStatus) error {
	for _, condition := range config.Status.Conditions {
		if condition.Type != v1.ConfigurationValid {
			continue
		}

		if condition.Status != status {
			return fmt.Errorf("configuration valid condition %v", condition.Status)
		}

		return nil
	}

	return fmt.Errorf("configuration valid condition not present")
}

// MustReplaceBrokerConfig updates the service broker configuration and waits
// for the broker to acquire the write lock and update the configuration to
// make it live.
func MustReplaceBrokerConfig(t *testing.T, clients client.Clients, spec *v1.ServiceBrokerConfigSpec) {
	if err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Delete(context.TODO(), config.ConfigurationName, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}

	configuration := &v1.ServiceBrokerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.ConfigurationName,
		},
		Spec: *spec,
	}

	if _, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Create(context.TODO(), configuration, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	callback := func() error {
		// Service broker will first check validity and update the resource.
		configuration, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Get(context.TODO(), config.ConfigurationName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if err := configurationValidCondition(configuration, v1.ConditionTrue); err != nil {
			return err
		}

		// The config is live when it is set in the config package.
		config.Lock()
		defer config.Unlock()

		c := config.Config()
		if c == nil {
			return fmt.Errorf("no config available")
		}

		if !reflect.DeepEqual(&c.Spec, spec) {
			return fmt.Errorf("specification do not match")
		}

		return nil
	}

	if err := util.WaitFor(callback, configUpdateTimeout); err != nil {
		t.Fatal(err)
	}

	// Every catalog used in testing should be validated at the API level to
	// ensure all permutations are valid.
	mustValidateCatalog(t)
}

// MustReplaceBrokerConfigWithInvalidCondition will updata the configuration and
// then ensure that the broker has registered it is invalid.
func MustReplaceBrokerConfigWithInvalidCondition(t *testing.T, clients client.Clients, spec *v1.ServiceBrokerConfigSpec) {
	if err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Delete(context.TODO(), config.ConfigurationName, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}

	configuration := &v1.ServiceBrokerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.ConfigurationName,
		},
		Spec: *spec,
	}

	if _, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Create(context.TODO(), configuration, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	callback := func() error {
		// Service broker will first check validity and update the resource.
		configuration, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(Namespace).Get(context.TODO(), config.ConfigurationName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if err := configurationValidCondition(configuration, v1.ConditionFalse); err != nil {
			return err
		}

		return nil
	}

	if err := util.WaitFor(callback, configUpdateTimeout); err != nil {
		t.Fatal(err)
	}
}

// MustGetRegistryEntry returns the registry entry for a service instance.
func MustGetRegistryEntry(t *testing.T, clients client.Clients, rt registry.Type, name string) *corev1.Secret {
	entry, err := clients.Kubernetes().CoreV1().Secrets(Namespace).Get(context.TODO(), registry.Name(rt, name), metav1.GetOptions{})
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

// keyInSlice looks for a key in the provided slice.
func keyInSlice(key string, slice []string) bool {
	for _, item := range slice {
		if item == key {
			return true
		}
	}

	return false
}

// mustValidateObject validates an API object.  The required attributes must exist, the optional
// ones may exist, anything not in these slices is illegal and a bug.
func mustValidateObject(t *testing.T, object map[string]interface{}, required, optional []string) {
	for _, requiredAttribute := range required {
		if _, ok := object[requiredAttribute]; !ok {
			t.Fatalf("required attribute %s not in object", requiredAttribute)
		}
	}

	for attribute := range object {
		if !keyInSlice(attribute, required) && !keyInSlice(attribute, optional) {
			t.Fatalf("attribute %s unexpectedly in object", attribute)
		}
	}
}

// mustValidateCatalog ensures the catalog has the correct attributes.
// This should be done with schema validation, provided it rejects rogue attributes.
func mustValidateCatalog(t *testing.T) {
	var object interface{}

	if err := Get("/v2/catalog", http.StatusOK, &object); err != nil {
		t.Fatal(err)
	}

	catalog, ok := object.(map[string]interface{})
	if !ok {
		t.Fatalf("object not correctly formatted")
	}

	required := []string{
		"services",
	}

	mustValidateObject(t, catalog, required, nil)

	services, ok := catalog["services"].([]interface{})
	if !ok {
		t.Fatalf("object not correctly formatted")
	}

	for _, object := range services {
		service, ok := object.(map[string]interface{})
		if !ok {
			t.Fatalf("object not correctly formatted")
		}

		required := []string{
			"name",
			"id",
			"description",
			"bindable",
			"plans",
		}

		optional := []string{
			"tags",
			"requires",
			"metadata",
			"dashboard_client",
			"plan_updatable",
		}

		mustValidateObject(t, service, required, optional)

		if object, ok := service["dashboard_client"]; ok {
			dashboardClient, ok := object.(map[string]interface{})
			if !ok {
				t.Fatalf("object not correctly formatted")
			}

			optional = []string{
				"id",
				"secret",
				"redirect_uri",
			}

			mustValidateObject(t, dashboardClient, nil, optional)
		}

		plans, ok := service["plans"].([]interface{})
		if !ok {
			t.Fatalf("object not correctly formatted")
		}

		for _, object := range plans {
			plan, ok := object.(map[string]interface{})
			if !ok {
				t.Fatalf("object not correctly formatted")
			}

			required = []string{
				"id",
				"name",
				"description",
			}

			optional = []string{
				"metadata",
				"free",
				"bindable",
				"schemas",
			}

			mustValidateObject(t, plan, required, optional)

			if object, ok := plan["schemas"]; ok {
				schemas, ok := object.(map[string]interface{})
				if !ok {
					t.Fatalf("object not correctly formatted")
				}

				optional = []string{
					"service_instance",
					"service_binding",
				}

				mustValidateObject(t, schemas, nil, optional)

				if object, ok := schemas["service_instance"]; ok {
					serviceInstanceSchema, ok := object.(map[string]interface{})
					if !ok {
						t.Fatalf("object not correctly formatted")
					}

					optional = []string{
						"create",
						"update",
					}

					mustValidateObject(t, serviceInstanceSchema, nil, optional)
				}

				if object, ok := schemas["service_binding"]; ok {
					serviceBindingSchema, ok := object.(map[string]interface{})
					if !ok {
						t.Fatalf("object not correctly formatted")
					}

					optional = []string{
						"create",
					}

					mustValidateObject(t, serviceBindingSchema, nil, optional)
				}
			}
		}
	}
}
