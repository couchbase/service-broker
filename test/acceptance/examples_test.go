package acceptance

import (
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"testing"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/util"

	"github.com/golang/glog"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

// createResources creates Kubernetes objects.
func createResources(namespace string, objects []*unstructured.Unstructured) error {
	for _, object := range objects {
		gvk := object.GroupVersionKind()

		mapping, err := clients.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		glog.V(1).Infof("Creating %s %s", object.GetKind(), object.GetName())

		if mapping.Scope.Name() == meta.RESTScopeNameRoot {
			if _, err := clients.Dynamic().Resource(mapping.Resource).Create(object, metav1.CreateOptions{}); err != nil {
				return err
			}

			continue
		}

		if _, err := clients.Dynamic().Resource(mapping.Resource).Namespace(namespace).Create(object, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// mustCreateResources creates Kubernetes objects.
func mustCreateResources(t *testing.T, namespace string, objects []*unstructured.Unstructured) {
	if err := createResources(namespace, objects); err != nil {
		t.Fatal(err)
	}
}

// generateServiceBrokerTLS returns TLS configuration for the service broker.
func generateServiceBrokerTLS(namespace string) ([]byte, []byte, []byte, error) {
	bits := 2048

	caKey, err := util.GenerateKey(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &bits)
	if err != nil {
		return nil, nil, nil, err
	}

	subject := pkix.Name{
		CommonName: "Service Broker CA",
	}

	caCertificate, err := util.GenerateCertificate(caKey, subject, time.Hour, v1.CA, nil, nil, nil, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	serverKey, err := util.GenerateKey(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &bits)
	if err != nil {
		return nil, nil, nil, err
	}

	subject = pkix.Name{
		CommonName: "Service Broker",
	}

	dnsSANs := []string{
		"couchbase-service-broker",
		fmt.Sprintf("couchbase-service-broker.%s", namespace),
		fmt.Sprintf("couchbase-service-broker.%s.svc", namespace),
	}

	serverCertificate, err := util.GenerateCertificate(serverKey, subject, time.Hour, v1.Server, dnsSANs, nil, caKey, caCertificate)
	if err != nil {
		return nil, nil, nil, err
	}

	return caCertificate, serverCertificate, serverKey, nil
}

// mustGenerateServiceBrokerTLS returns TLS configuration for the service broker.
func mustGenerateServiceBrokerTLS(t *testing.T, namespace string) ([]byte, []byte, []byte) {
	caCertificate, serverCertificate, serverKey, err := generateServiceBrokerTLS(namespace)
	if err != nil {
		t.Fatal(err)
	}

	return caCertificate, serverCertificate, serverKey
}

// configurationValid returns a verification function that reports whether a service broker
// configuration is valid as per the status condition.
func configurationValid(namespace string) func() error {
	return func() error {
		configuration, err := clients.Broker().ServicebrokerV1alpha1().ServiceBrokerConfigs(namespace).Get(config.ConfigurationNameDefault, metav1.GetOptions{})
		if err != nil {
			return err
		}

		for _, condition := range configuration.Status.Conditions {
			if condition.Type != v1.ConfigurationValid {
				continue
			}

			if condition.Status == v1.ConditionTrue {
				return nil
			}

			return fmt.Errorf("configuration validation condition %v", condition.Status)
		}

		return fmt.Errorf("configuration validation condition does not exist")
	}
}

// deploymentAvailable returns a verification function that reports whether the service
// broker deployment is available as per its status conditions.
func deploymentAvailable(namespace string) func() error {
	return func() error {
		deployment, err := clients.Kubernetes().AppsV1().Deployments(namespace).Get(exampleBrokerDeploymentName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		for _, condition := range deployment.Status.Conditions {
			if condition.Type != appsv1.DeploymentAvailable {
				continue
			}

			if condition.Status == corev1.ConditionTrue {
				return nil
			}

			return fmt.Errorf("deployment available condition %v", condition.Status)
		}

		return fmt.Errorf("deployment available condition does not exist")
	}
}

// clusterServiceBrokerReady is a verification function that reports whether the
// cluster service broker is ready as per its status conditions.
func clusterServiceBrokerReady() error {
	gvr := schema.GroupVersionResource{
		Group:    "servicecatalog.k8s.io",
		Version:  "v1beta1",
		Resource: "clusterservicebrokers",
	}

	object, err := clients.Dynamic().Resource(gvr).Get("couchbase-service-broker", metav1.GetOptions{})
	if err != nil {
		return err
	}

	conditions, ok, _ := unstructured.NestedSlice(object.Object, "status", "conditions")
	if !ok {
		return fmt.Errorf("object has no status conditions")
	}

	for _, condition := range conditions {
		conditionObject, ok := condition.(map[string]interface{})
		if !ok {
			return fmt.Errorf("object condition malformed")
		}

		t, ok, _ := unstructured.NestedString(conditionObject, "type")
		if !ok {
			return fmt.Errorf("object condition has no type")
		}

		if t != "Ready" {
			continue
		}

		status, ok, _ := unstructured.NestedString(conditionObject, "status")
		if !ok {
			return fmt.Errorf("object ready condition has no status")
		}

		if status != "True" {
			return fmt.Errorf("object ready condition status %v", status)
		}

		return nil
	}

	return fmt.Errorf("object ready condition does not exist")
}

// cleanupClusterServiceBroker removes a cluster service broker from the system.
func cleanupClusterServiceBroker() {
	gvr := schema.GroupVersionResource{
		Group:    "servicecatalog.k8s.io",
		Version:  "v1beta1",
		Resource: "clusterservicebrokers",
	}

	glog.V(1).Infof("Deleting ClusterServiceBroker couchbase-service-broker")

	if err := clients.Dynamic().Resource(gvr).Delete("couchbase-service-broker", metav1.NewDeleteOptions(0)); err != nil {
		glog.V(1).Info(err)
		return
	}

	callback := func() error {
		if _, err := clients.Dynamic().Resource(gvr).Get("couchbase-service-broker", metav1.GetOptions{}); err == nil {
			return fmt.Errorf("resource still exists")
		}

		return nil
	}

	if err := util.WaitFor(callback, time.Minute); err != nil {
		glog.V(1).Info(err)
	}
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
			namespace, cleanupNamespace := mustSetupNamespace(t, clients)
			defer cleanupNamespace()

			// Install the service broker configuration for the example.
			// * Tests example passes CRD validation.
			configurationPath := path.Join(exampleConfigurationDir, name, exampleConfigurationSpecification)

			objects := mustReadYAMLObjects(t, configurationPath)

			mustCreateResources(t, namespace, objects)

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

			mustCreateResources(t, namespace, objects)

			util.MustWaitFor(t, configurationValid(namespace), time.Minute)
			util.MustWaitFor(t, deploymentAvailable(namespace), time.Minute)

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

			mustCreateResources(t, namespace, objects)

			defer cleanupClusterServiceBroker()

			util.MustWaitFor(t, clusterServiceBrokerReady, time.Minute)
		}

		t.Run("TestExample-"+name, test)
	}
}
