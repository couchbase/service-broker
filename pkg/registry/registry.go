// package registry is the persistence layer of the service broker.
//
// It is modelled on Kubernetes ConfigMaps.  These are the canonical source of truth
// about a service instance.  They also act as the root element of ownership, so
// deleting a registry entry will deprovision the service instance and any service
// bindings using Kubernetes' built in garbage collection.

package registry

import (
	"fmt"

	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/version"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// versionAnnotaiton recorder the broker version for upgrades.
	versionAnnotaiton = "broker.couchbase.com/version"

	// instanceIDAnnotation is used to store the raw instance ID a registry
	// entry corresponds to.
	instanceIDAnnotation = "broker.couchbase.com/instanceid"

	// bindingIDAnnotaiton is used to store the raw binding ID a registry
	// entry corresponds to.
	bindingIDAnnotaiton = "broker.couchbase.com/bindingid"

	// ServiceInstanceRequestKey is used to store the JSON encoded request object in
	// the servics instance's registry entry.
	ServiceInstanceRequestKey = "serviceInstanceRequest"

	// ServiceOfferingKey is used to store the service offering in a service instance's
	// registry entry.
	ServiceOfferingKey = "serviceID"

	// ServicePlanKey is used to store the service plan in a service instance's
	// registry entry.
	ServicePlanKey = "planID"

	// CACertificateKey is used to store the base64 encoded CA certificate in the
	// service instances's registry entry.
	CACertificateKey = "caCert"

	// RegistryKeyDashboardURL is used to store the service instance
	// dachboard location.
	RegistryKeyDashboardURL = "dashboardURL"
)

// RegistryEntryName defines a type for a name for type safety reasons.
type RegistryEntryName struct {
	instanceID string
	bindingID  string
}

// name returns the unique ConfigMap name for the registry entry type.
func (r RegistryEntryName) name() (string, error) {
	if r.bindingID != "" {
		return "service-binding-" + r.bindingID, nil
	}
	if r.instanceID != "" {
		return "service-instance-" + r.instanceID, nil
	}
	return "", fmt.Errorf("illegal registry entry name: %v", r)
}

// annotations returns the annotations to apply to registry entry ConfigMaps.
func (r RegistryEntryName) annotations() map[string]string {
	annotations := map[string]string{
		versionAnnotaiton:    version.Version,
		instanceIDAnnotation: r.instanceID,
	}
	if r.bindingID != "" {
		annotations[bindingIDAnnotaiton] = r.bindingID
	}
	return annotations
}

// ServiceInstanceRegistryName returns a unique name for the service broker
// registry ConfigMaps.  We have to prefix these with something different than
// the instanceID as the operator will create a config map with the same name
// for persistent storage.
func ServiceInstanceRegistryName(instanceID string) RegistryEntryName {
	return RegistryEntryName{
		instanceID: instanceID,
	}
}

// ServiceBindingRegistryName returns a unique name for a service broker
// registry ConfigMap.
func ServiceBindingRegistryName(instanceID, bindingID string) RegistryEntryName {
	return RegistryEntryName{
		instanceID: instanceID,
		bindingID:  bindingID,
	}
}

// RegistryEntry is the main type used to lookup service instances and persist
// data.
type RegistryEntry struct {
	// client is a cached client to allow creation, update and deletion
	// of config maps
	client kubernetes.Interface

	// configMap is the cached configuration map underlying the service
	// instance registry entry.
	configMap *corev1.ConfigMap
}

// GetOwnerReference returns the owner reference to attach to all resources created
// referenced by the template binding.
func (r *RegistryEntry) GetOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       r.configMap.Name,
		UID:        r.configMap.UID,
	}
}

// GetValue returns a value from the registry's KV store.
func (r *RegistryEntry) Get(key string) (string, error) {
	value, ok := r.configMap.Data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in registry for instance %s", key, r.configMap.Annotations[instanceIDAnnotation])
	}
	return value, nil
}

// SetValue sets a value in the registry's KV store and persists the data.
func (r *RegistryEntry) Set(key, value string) error {
	configMap := r.configMap.DeepCopy()
	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}
	configMap.Data[key] = value
	configMap, err := r.client.CoreV1().ConfigMaps(configMap.Namespace).Update(configMap)
	if err != nil {
		return err
	}
	r.configMap = configMap
	return nil
}

// Registry is a global object used to manage individual registry entries.
type Registry struct {
	// naemspace defines the namespace where the registry entries live,
	// which is the same as where the broker is running.
	namespace string
}

// New creates a new registry.
func New(namespace string) *Registry {
	return &Registry{
		namespace: namespace,
	}
}

// New creates a new registry entry for the instance ID.
func (r *Registry) New(name RegistryEntryName) (*RegistryEntry, error) {
	registryName, err := name.name()
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        registryName,
			Annotations: name.annotations(),
		},
	}

	client := config.Clients().Kubernetes()
	configMap, err = config.Clients().Kubernetes().CoreV1().ConfigMaps(r.namespace).Create(configMap)
	if err != nil {
		return nil, err
	}

	entry := &RegistryEntry{
		client:    client,
		configMap: configMap,
	}

	return entry, nil
}

// Get gets an existing registry entry for the instance ID.
func (r *Registry) Get(name RegistryEntryName) (*RegistryEntry, error) {
	registryName, err := name.name()
	if err != nil {
		return nil, err
	}

	client := config.Clients().Kubernetes()
	configMap, err := client.CoreV1().ConfigMaps(r.namespace).Get(registryName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	entry := &RegistryEntry{
		client:    client,
		configMap: configMap,
	}

	return entry, nil
}

// Delete deletes an existing registry entry for the instance ID.
func (r *Registry) Delete(name RegistryEntryName) error {
	registryName, err := name.name()
	if err != nil {
		return err
	}

	client := config.Clients().Kubernetes()
	return client.CoreV1().ConfigMaps(r.namespace).Delete(registryName, metav1.NewDeleteOptions(0))
}
