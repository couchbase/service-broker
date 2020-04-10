package util

import (
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/util"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// crdDir is where CRDs are installed by the docker file.
	crdDir = "/usr/local/share/couchbase-service-broker/crds"
)

// readYAMLObjects reads in a YAML file and unmarshals as unstructured objects.
func readYAMLObjects(path string) ([]*unstructured.Unstructured, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	objects := []*unstructured.Unstructured{}

	sections := strings.Split(string(data), "---")
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}

		object := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(section), object); err != nil {
			return nil, err
		}

		objects = append(objects, object)
	}

	return objects, nil
}

// MustReadYAMLObjects reads in a YAML file and unmarshals as unstructured objects.
func MustReadYAMLObjects(t *testing.T, path string) []*unstructured.Unstructured {
	objects, err := readYAMLObjects(path)
	if err != nil {
		t.Fatal(err)
	}

	return objects
}

// findResource expects to find a specifc resource in a list of objects.
func findResource(objects []*unstructured.Unstructured, groupVersion, kind, name string) (*unstructured.Unstructured, error) {
	for _, object := range objects {
		if object.GetAPIVersion() != groupVersion {
			continue
		}

		if object.GetKind() != kind {
			continue
		}

		if object.GetName() != name {
			continue
		}

		return object, nil
	}

	return nil, fmt.Errorf("unable to locate requested resource")
}

// MustFindResource expects to find a specifc resource in a list of objects.
func MustFindResource(t *testing.T, objects []*unstructured.Unstructured, groupVersion, kind, name string) *unstructured.Unstructured {
	object, err := findResource(objects, groupVersion, kind, name)
	if err != nil {
		t.Fatal(err)
	}

	return object
}

// createResources creates Kubernetes objects.
func createResources(clients client.Clients, namespace string, objects []*unstructured.Unstructured) error {
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

// MustCreateResources creates Kubernetes objects.
func MustCreateResources(t *testing.T, clients client.Clients, namespace string, objects []*unstructured.Unstructured) {
	if err := createResources(clients, namespace, objects); err != nil {
		t.Fatal(err)
	}
}

// getResource gets the most up to date version of a resource.
func getResource(clients client.Clients, namespace string, object *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	gvk := object.GroupVersionKind()

	mapping, err := clients.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		if object, err = clients.Dynamic().Resource(mapping.Resource).Get(object.GetName(), metav1.GetOptions{}); err != nil {
			return nil, err
		}

		return object, nil
	}

	if object, err = clients.Dynamic().Resource(mapping.Resource).Namespace(namespace).Get(object.GetName(), metav1.GetOptions{}); err != nil {
		return nil, err
	}

	return object, nil
}

// setupCRDs deletes any CRDs we find that belong to our API group then creates
// any that are installed in the CRD directory installed in the container, the
// make file will ensure these are up to date.
func SetupCRDs(clients client.Clients) error {
	// Just use the dynamic client here as using typed clients requires
	// a package the main service broker doesn't need to include.
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1beta1",
		Resource: "customresourcedefinitions",
	}

	crds, err := clients.Dynamic().Resource(gvr).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, crd := range crds.Items {
		group, ok, err := unstructured.NestedString(crd.Object, "spec", "group")
		if err != nil {
			return err
		}

		if !ok {
			return fmt.Errorf("crd doesn't have value for spec.group")
		}

		if group != v1.GroupName {
			continue
		}

		name := crd.GetName()

		glog.V(1).Info("Deleting CRD", name)

		if err := clients.Dynamic().Resource(gvr).Delete(name, metav1.NewDeleteOptions(0)); err != nil {
			return err
		}

		callback := func() error {
			if _, err := clients.Dynamic().Resource(gvr).Get(name, metav1.GetOptions{}); err == nil {
				return fmt.Errorf("resource still exists")
			}

			return nil
		}

		if err := util.WaitFor(callback, time.Minute); err != nil {
			return err
		}
	}

	files, err := ioutil.ReadDir(crdDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		crdPath := path.Join(crdDir, file.Name())

		crds, err := readYAMLObjects(crdPath)
		if err != nil {
			return err
		}

		for _, crd := range crds {
			glog.V(1).Info("Creating CRD", crd.GetName())

			if _, err := clients.Dynamic().Resource(gvr).Create(crd, metav1.CreateOptions{}); err != nil {
				return err
			}
		}
	}

	return nil
}

// setupNamespace creates a temporary, random namespace to use for testing in.
func setupNamespace(clients client.Clients) (string, func(), error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "acceptance-",
		},
	}

	newNamespace, err := clients.Kubernetes().CoreV1().Namespaces().Create(namespace)
	if err != nil {
		return "", nil, err
	}

	glog.V(1).Infof("Created Namespace %s", newNamespace.Name)

	cleanup := func() {
		glog.V(1).Infof("Deleting Namespace %s", newNamespace.Name)

		if err := clients.Kubernetes().CoreV1().Namespaces().Delete(newNamespace.Name, metav1.NewDeleteOptions(0)); err != nil {
			glog.Fatal(err)
		}
	}

	return newNamespace.Name, cleanup, nil
}

// MustSetupNamespace creates a temporary, random namespace to use for testing in.
func MustSetupNamespace(t *testing.T, clients client.Clients) (string, func()) {
	namespace, cleanup, err := setupNamespace(clients)
	if err != nil {
		t.Fatal(err)
	}

	return namespace, cleanup
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

// MustGenerateServiceBrokerTLS returns TLS configuration for the service broker.
func MustGenerateServiceBrokerTLS(t *testing.T, namespace string) ([]byte, []byte, []byte) {
	caCertificate, serverCertificate, serverKey, err := generateServiceBrokerTLS(namespace)
	if err != nil {
		t.Fatal(err)
	}

	return caCertificate, serverCertificate, serverKey
}

// ResourceCondition checks the resource for the specified condition.
func ResourceCondition(clients client.Clients, namespace string, object *unstructured.Unstructured, conditionType, conditionStatus string) func() error {
	return func() error {
		var err error

		object, err = getResource(clients, namespace, object)
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

			if t != conditionType {
				continue
			}

			status, ok, _ := unstructured.NestedString(conditionObject, "status")
			if !ok {
				return fmt.Errorf("object ready condition has no status")
			}

			if status != conditionStatus {
				return fmt.Errorf("object ready condition status %v", status)
			}

			return nil
		}

		return fmt.Errorf("object ready condition does not exist")
	}
}

// DeleteClusterResource removes a resource from the system.
func DeleteClusterResource(clients client.Clients, object *unstructured.Unstructured) {
	gvk := object.GroupVersionKind()

	mapping, err := clients.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		glog.V(1).Info(err)
		return
	}

	glog.V(1).Infof("Deleting %s %s", object.GetKind(), object.GetName())

	if err := clients.Dynamic().Resource(mapping.Resource).Delete(object.GetName(), metav1.NewDeleteOptions(0)); err != nil {
		glog.V(1).Info(err)
		return
	}

	// Wait for deletion.
	callback := func() error {
		if _, err := getResource(clients, "", object); err == nil {
			return fmt.Errorf("resource still exists")
		}

		return nil
	}

	if err := util.WaitFor(callback, time.Minute); err != nil {
		glog.V(1).Info(err)
	}
}
