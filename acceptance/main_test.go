package acceptance

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/couchbase/service-broker/pkg/apis"
	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/test/util"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	// crdDir is where CRDs are installed by the docker file.
	crdDir = "/usr/local/share/couchbase-service-broker/crds"
)

var (
	// clients is the global cache of clients.
	clients client.Clients
)

// setupCRDs deletes any CRDs we find that belong to our API group then creates
// any that are installed in the CRD directory installed in the container, the
// make file will ensure these are up to date.
func setupCRDs(clients client.Clients) error {
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

		callback := func() bool {
			if _, err := clients.Dynamic().Resource(gvr).Get(name, metav1.GetOptions{}); err == nil {
				return false
			}

			return true
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

// mustSetupNamespace creates a temporary, random namespace to use for testing in.
func mustSetupNamespace(t *testing.T, clients client.Clients) (string, func()) {
	namespace, cleanup, err := setupNamespace(clients)
	if err != nil {
		t.Fatal(err)
	}

	return namespace, cleanup
}

// TestMain performs any cluster initialization.
func TestMain(m *testing.M) {
	// For the benefit of glog.
	flag.Parse()

	// Add any custom resource types to the global scheme.
	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	var err error

	// Create any clients we need.
	clients, err = client.New()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	// Delete and recreate any CRDs so we have the most up to date
	// versions installed.
	if err := setupCRDs(clients); err != nil {
		glog.Fatal(err)
		os.Exit(0)
	}

	// Run the tests.
	os.Exit(m.Run())
}
