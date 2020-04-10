package acceptance

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"testing"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/test/acceptance/util"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	// exampleDefaultResourceName is the common service broker resource name.
	exampleDefaultResourceName = "couchbase-service-broker"

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
			namespace, cleanupNamespace := util.MustSetupNamespace(t, clients)
			defer cleanupNamespace()

			// Install the service broker configuration for the example.
			// * Tests example passes CRD validation.
			configurationPath := path.Join(exampleConfigurationDir, name, exampleConfigurationSpecification)

			objects := util.MustReadYAMLObjects(t, configurationPath)
			serviceBrokerConfiguration := util.MustFindResource(t, objects, "broker.couchbase.com/v1alpha1", "ServiceBrokerConfig", exampleDefaultResourceName)

			util.MustCreateResources(t, clients, namespace, objects)

			// Install the service broker, we need to check that the service broker
			// flags the configuration as valid and the deployment is available.
			// As the namespace is ephemeral we need to watch out for any resources
			// that usually refer to "default" explicitly.
			// * Tests service broker comes up in Kubernetes
			// * Tests example passses service broker validation
			caCertificate, serverCertificate, serverKey := util.MustGenerateServiceBrokerTLS(t, namespace)

			objects = util.MustReadYAMLObjects(t, exampleBrokerConfiguration)
			serviceBrokerSecret := util.MustFindResource(t, objects, "v1", "Secret", exampleDefaultResourceName)
			serviceBrokerRoleBinding := util.MustFindResource(t, objects, "rbac.authorization.k8s.io/v1", "RoleBinding", exampleDefaultResourceName)
			serviceBrokerDeployment := util.MustFindResource(t, objects, "apps/v1", "Deployment", exampleDefaultResourceName)

			// Override the service broker TLS secret data.
			data := map[string]interface{}{
				"token":           base64.StdEncoding.EncodeToString([]byte(token)),
				"tls-certificate": base64.StdEncoding.EncodeToString(serverCertificate),
				"tls-private-key": base64.StdEncoding.EncodeToString(serverKey),
			}

			if err := unstructured.SetNestedField(serviceBrokerSecret.Object, data, "data"); err != nil {
				t.Fatal(err)
			}

			// Override the service broker role binding namespace.
			subjects := []interface{}{
				map[string]interface{}{
					"kind":      "ServiceAccount",
					"name":      exampleDefaultResourceName,
					"namespace": namespace,
				},
			}

			if err := unstructured.SetNestedField(serviceBrokerRoleBinding.Object, subjects, "subjects"); err != nil {
				t.Fatal(err)
			}

			util.MustCreateResources(t, clients, namespace, objects)

			util.MustWaitFor(t, util.ResourceCondition(clients, namespace, serviceBrokerConfiguration, string(v1.ConfigurationValid), string(v1.ConditionTrue)), time.Minute)
			util.MustWaitFor(t, util.ResourceCondition(clients, namespace, serviceBrokerDeployment, string(appsv1.DeploymentAvailable), string(corev1.ConditionTrue)), time.Minute)

			// Register the service broker with the service catalog.
			// We replaced the service broker configuration with new TLS due to the
			// namespace change, do the same here.
			objects = util.MustReadYAMLObjects(t, exampleClusterServiceBroker)
			clusterServiceBroker := util.MustFindResource(t, objects, "servicecatalog.k8s.io/v1beta1", "ClusterServiceBroker", exampleDefaultResourceName)

			if err := unstructured.SetNestedField(clusterServiceBroker.Object, fmt.Sprintf("https://%s.%s", exampleDefaultResourceName, namespace), "spec", "url"); err != nil {
				t.Fatal(err)
			}

			if err := unstructured.SetNestedField(clusterServiceBroker.Object, base64.StdEncoding.EncodeToString(caCertificate), "spec", "caBundle"); err != nil {
				t.Fatal(err)
			}

			if err := unstructured.SetNestedField(clusterServiceBroker.Object, namespace, "spec", "authInfo", "bearer", "secretRef", "namespace"); err != nil {
				t.Fatal(err)
			}

			util.MustCreateResources(t, clients, namespace, objects)

			defer util.DeleteClusterResource(clients, clusterServiceBroker)

			util.MustWaitFor(t, util.ResourceCondition(clients, namespace, clusterServiceBroker, "Ready", "True"), time.Minute)
		}

		t.Run("TestExample-"+name, test)
	}
}
