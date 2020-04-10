package acceptance

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"testing"
	"time"

	"github.com/couchbase/service-broker/pkg/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// token is a bearer token for API authentication.
	token = "allMyCatsDoIsMeowRandomlyAndBegForFood"

	// exampleDir contains any common example files.
	exampleDir = "/usr/local/share/couchbase-service-broker/examples"

	// exampleBrokerConfiguration is the common service broker configuration file.
	exampleBrokerConfiguration = exampleDir + "/broker.yaml"

	// exampleClusterServiceBroker is the common service broker registration file.
	exampleClusterServiceBroker = exampleDir + "/clusterservicebroker.yaml"

	// exampleBrokerDeploymentName is the common service broker deployment name.
	exampleBrokerDeploymentName = "couchbase-service-broker"

	// exampleConfigurationDir contains any application specific examples.
	exampleConfigurationDir = exampleDir + "/" + "configurations"

	// exampleConfigurationSpecification contains the main configuration
	// files for an example configuration.
	exampleConfigurationSpecification = "broker.yaml"

	// exampleConfigurationServiceInstance contains the configuration service
	// instance definition.
	//exampleConfigurationServiceInstance = "serviceinstance.yaml"

	// exampleConfigurationServiceBinding contains the configuration service
	// binding definition.
	//exampleConfigurationServiceBinding = "servicebinding.yaml"
)

// TestExamples works through examples provided as part of the repository.
// This tests against a Kubernetes cluster to ensure the configurations
// pass validation, that the service broker can spawn a service instance
// and optionally a service binding.
func TestExamples(t *testing.T) {
	configurations, err := ioutil.ReadDir(exampleConfigurationDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, configuration := range configurations {
		name := configuration.Name()

		test := func(t *testing.T) {
			// Create a clean namespace to test in, we can clean up everything
			// by just deleting it and letting the cascade do its thing.
			namespace, cleanupNamespace := mustSetupNamespace(t, clients)
			defer cleanupNamespace()

			// Install the service broker configuration for the example.
			// * Tests example passes CRD validation.
			configurationPath := path.Join(exampleConfigurationDir, name, exampleConfigurationSpecification)

			objects := mustReadYAMLObjects(t, configurationPath)

			mustCreateResources(t, clients, namespace, objects)

			// Install the service broker, we need to check that the service broker
			// flags the configuration as valid and the deployment is available.
			// As the namespace is ephemeral we need to watch out for any resources
			// that usually refer to "default" explicitly.
			// * Tests service broker comes up in Kubernetes
			// * Tests example passses service broker validation
			caCertificate, serverCertificate, serverKey := mustGenerateServiceBrokerTLS(t, namespace)

			objects = mustReadYAMLObjects(t, exampleBrokerConfiguration)

			for _, object := range objects {
				// Override the service broker TLS secret data.
				if object.GetKind() == "Secret" {
					data := map[string]interface{}{
						"token":           base64.StdEncoding.EncodeToString([]byte(token)),
						"tls-certificate": base64.StdEncoding.EncodeToString(serverCertificate),
						"tls-private-key": base64.StdEncoding.EncodeToString(serverKey),
					}

					if err := unstructured.SetNestedField(object.Object, data, "data"); err != nil {
						t.Fatal(err)
					}
				}

				// Override the service broker role binding namespace.
				if object.GetKind() == "RoleBinding" {
					subjects := []interface{}{
						map[string]interface{}{
							"kind":      "ServiceAccount",
							"name":      "couchbase-service-broker",
							"namespace": namespace,
						},
					}

					if err := unstructured.SetNestedField(object.Object, subjects, "subjects"); err != nil {
						t.Fatal(err)
					}
				}
			}

			mustCreateResources(t, clients, namespace, objects)

			util.MustWaitFor(t, configurationValid(clients, namespace), time.Minute)
			util.MustWaitFor(t, deploymentAvailable(clients, namespace, exampleBrokerDeploymentName), time.Minute)

			// Register the service broker with the service catalog.
			// We replaced the service broker configuration with new TLS due to the
			// namespace change, do the same here.
			objects = mustReadYAMLObjects(t, exampleClusterServiceBroker)

			for _, object := range objects {
				if object.GetKind() == "ClusterServiceBroker" {
					if err := unstructured.SetNestedField(object.Object, fmt.Sprintf("https://couchbase-service-broker.%s", namespace), "spec", "url"); err != nil {
						t.Fatal(err)
					}

					if err := unstructured.SetNestedField(object.Object, base64.StdEncoding.EncodeToString(caCertificate), "spec", "caBundle"); err != nil {
						t.Fatal(err)
					}

					if err := unstructured.SetNestedField(object.Object, namespace, "spec", "authInfo", "bearer", "secretRef", "namespace"); err != nil {
						t.Fatal(err)
					}
				}
			}

			mustCreateResources(t, clients, namespace, objects)

			defer cleanupClusterServiceBroker(clients)

			util.MustWaitFor(t, clusterServiceBrokerReady(clients), time.Minute)
		}

		t.Run("TestExample-"+name, test)
	}
}
