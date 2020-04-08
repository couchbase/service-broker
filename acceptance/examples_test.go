package acceptance

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// exampleDir contains any common example files.
	exampleDir = "/usr/local/share/couchbase-service-broker/examples"

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

// createResources creates Kubernetes objects.
func createResources(namespace string, objects []*unstructured.Unstructured) error {
	for _, object := range objects {
		gvk := object.GroupVersionKind()

		mapping, err := clients.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		glog.V(1).Infof("Creating %s %s", object.GetKind(), object.GetName())

		if _, err := clients.Dynamic().Resource(mapping.Resource).Namespace(namespace).Create(object, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

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
			namespace, cleanup, err := setupNamespace(clients)
			if err != nil {
				t.Fatal(err)
			}

			defer cleanup()

			// Install the service broker configuration for the example.
			// * Tests example passes CRD validation.
			configurationPath := path.Join(exampleConfigurationDir, name, exampleConfigurationSpecification)

			objects, err := readYAMLObjects(configurationPath)
			if err != nil {
				t.Fatal(err)
			}

			if err := createResources(namespace, objects); err != nil {
				t.Fatal(err)
			}

			// Install the service broker, we need to check that the service broker
			// flags the configuration as valid and the deployment is available.
			// As the namespace is ephemeral we need to watch out for the ssrvice broker
			// secret and replace its certificate/key pair with something containing
			// the correct DNS SAN for the service.
			// * Tests example passses service broker validation
			glog.V(1).Info("placeholder")
		}

		t.Run("TestExample-"+name, test)
	}
}
