// Copyright 2020 Couchbase, Inc.
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

package acceptance_test

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
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

	// exampleConfigurationPrerequisites contains things that are required
	// for an acceptance tests to run correctly.  These include things like
	// cluster resources that the service broker doesn't handle.
	exampleConfigurationPrerequisites = "prerequisites.yaml"

	// exampleConfigurationSpecification contains the main configuration
	// files for an example configuration.
	exampleConfigurationSpecification = "broker.yaml"

	// exampleConfigurationServiceInstance contains the configuration service
	// instance definition.
	exampleConfigurationServiceInstance = "serviceinstance.yaml"

	// exampleDefaultServiceInstanceName is the name an example service instance
	// must be called.
	exampleDefaultServiceInstanceName = "test-instance"

	// exampleConfigurationServiceBinding contains the configuration service
	// binding definition.
	exampleConfigurationServiceBinding = "servicebinding.yaml"

	// exampleDefaultServiceBindingName is the name an example service binding
	// must be called.
	exampleDefaultServiceBindingName = "test-binding"
)

// TestExamples works through examples provided as part of the repository.
// This tests against a Kubernetes cluster to ensure the configurations
// pass validation, that the service broker can spawn a service instance
// and optionally a service binding.
func TestExamples(t *testing.T) {
	configurations, err := ioutil.ReadDir(exampleConfigurationDir)
	if err != nil {
		util.Die(t, err)
	}

	for _, configuration := range configurations {
		name := configuration.Name()

		test := func(t *testing.T) {
			// Install any prerequisites.
			prerequisitesPath := path.Join(exampleConfigurationDir, name, exampleConfigurationPrerequisites)

			if _, err := os.Stat(prerequisitesPath); err == nil {
				objects := util.MustReadYAMLObjects(t, prerequisitesPath)

				util.MustCreateResources(t, clients, "default", objects)

				for i := range objects {
					object := objects[i]

					defer util.DeleteResource(clients, "default", object)
				}

				// Hack, we need a better way of synchronizing the readiness of
				// resources such as these.  For example, a dynamic admission controller
				// installed in this phase may not have started e.g. still pulling
				// the image, when the service instance is created.  The service
				// instance then misses all its defaults and collapses in a heap.
				time.Sleep(time.Minute)
			}

			// Create a clean namespace to test in, we can clean up everything
			// by just deleting it and letting the cascade do its thing.
			namespace := util.MustCreateResource(t, clients, "", util.MustGetNamespace(t))

			defer util.DeleteResource(clients, "", namespace)

			// Install the service broker configuration for the example.
			// * Tests example passes CRD validation.
			configurationPath := path.Join(exampleConfigurationDir, name, exampleConfigurationSpecification)

			objects := util.MustReadYAMLObjects(t, configurationPath)
			serviceBrokerConfiguration := util.MustFindResource(t, objects, "servicebroker.couchbase.com/v1alpha1", "ServiceBrokerConfig", exampleDefaultResourceName)

			util.MustCreateResources(t, clients, namespace.GetName(), objects)

			// Install the service broker, we need to check that the service broker
			// flags the configuration as valid and the deployment is available.
			// As the namespace is ephemeral we need to watch out for any resources
			// that usually refer to "default" explicitly.
			// * Tests service broker comes up in Kubernetes.
			// * Tests example passses service broker validation.
			caCertificate, serverCertificate, serverKey := util.MustGenerateServiceBrokerTLS(t, namespace.GetName())

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
				util.Die(t, err)
			}

			// Override the service broker role binding namespace.
			subjects := []interface{}{
				map[string]interface{}{
					"kind":      "ServiceAccount",
					"name":      exampleDefaultResourceName,
					"namespace": namespace.GetName(),
				},
			}

			if err := unstructured.SetNestedField(serviceBrokerRoleBinding.Object, subjects, "subjects"); err != nil {
				util.Die(t, err)
			}

			util.MustCreateResources(t, clients, namespace.GetName(), objects)

			util.MustWaitFor(t, util.ResourceCondition(clients, namespace.GetName(), serviceBrokerConfiguration, string(v1.ConfigurationValid), string(v1.ConditionTrue)), time.Minute)
			util.MustWaitFor(t, util.ResourceCondition(clients, namespace.GetName(), serviceBrokerDeployment, string(appsv1.DeploymentAvailable), string(corev1.ConditionTrue)), time.Minute)

			// Register the service broker with the service catalog.
			// We replaced the service broker configuration with new TLS due to the
			// namespace change, do the same here.
			// * Tests the service catalog can talk to the service broker.
			objects = util.MustReadYAMLObjects(t, exampleClusterServiceBroker)
			clusterServiceBroker := util.MustFindResource(t, objects, "servicecatalog.k8s.io/v1beta1", "ClusterServiceBroker", exampleDefaultResourceName)

			if err := unstructured.SetNestedField(clusterServiceBroker.Object, fmt.Sprintf("https://%s.%s", exampleDefaultResourceName, namespace.GetName()), "spec", "url"); err != nil {
				util.Die(t, err)
			}

			if err := unstructured.SetNestedField(clusterServiceBroker.Object, base64.StdEncoding.EncodeToString(caCertificate), "spec", "caBundle"); err != nil {
				util.Die(t, err)
			}

			if err := unstructured.SetNestedField(clusterServiceBroker.Object, namespace.GetName(), "spec", "authInfo", "bearer", "secretRef", "namespace"); err != nil {
				util.Die(t, err)
			}

			util.MustCreateResources(t, clients, namespace.GetName(), objects)

			defer util.DeleteResource(clients, "", clusterServiceBroker)

			util.MustWaitFor(t, util.ResourceCondition(clients, namespace.GetName(), clusterServiceBroker, "Ready", "True"), time.Minute)

			// Create the service instance.
			// * Tests the configuration provisions.
			serviceInstancePath := path.Join(exampleConfigurationDir, name, exampleConfigurationServiceInstance)

			objects = util.MustReadYAMLObjects(t, serviceInstancePath)
			serviceInstance := util.MustFindResource(t, objects, "servicecatalog.k8s.io/v1beta1", "ServiceInstance", exampleDefaultServiceInstanceName)

			util.MustCreateResources(t, clients, namespace.GetName(), objects)

			util.MustWaitFor(t, util.ResourceCondition(clients, namespace.GetName(), serviceInstance, "Ready", "True"), 10*time.Minute)

			// Create the service binding if one exists.
			// * Tests the configuration provisions.
			serviceBindingPath := path.Join(exampleConfigurationDir, name, exampleConfigurationServiceBinding)
			if _, err := os.Stat(serviceBindingPath); err == nil {
				objects = util.MustReadYAMLObjects(t, serviceBindingPath)
				serviceBinding := util.MustFindResource(t, objects, "servicecatalog.k8s.io/v1beta1", "ServiceBinding", exampleDefaultServiceBindingName)

				util.MustCreateResources(t, clients, namespace.GetName(), objects)

				util.MustWaitFor(t, util.ResourceCondition(clients, namespace.GetName(), serviceBinding, "Ready", "True"), 5*time.Minute)

				// Delete the service binding.
				// * Tests the service binding is deprovisioned cleanly.
				util.DeleteResource(clients, namespace.GetName(), serviceBinding)
			}

			// Delete the service instance.
			// * Tests the service instance is deprovisioned cleanly.
			util.DeleteResource(clients, namespace.GetName(), serviceInstance)
		}

		t.Run("TestExample-"+name, test)
	}
}
