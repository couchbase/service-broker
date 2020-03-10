// package registry is the persistence layer of the service broker.
package registry

import (
	"encoding/json"
	"sync"

	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/version"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// labelBase is the root of all labels and annotations.
	labelBase = "broker.couchbase.com"

	// versionAnnotaiton recorder the broker version for upgrades.
	versionAnnotaiton = labelBase + "/version"
)

// Key is an indentifier of a value in the registry entry's KV map.
type Key string

const (
	// ServiceID is the service ID related to the instance or binding.
	ServiceID Key = "service-id"

	// PlanID is the plan ID related to the instance or binding.
	PlanID Key = "plan-id"

	// Context is the context used to create or update the instance or binding.
	Context Key = "context"

	// Parameters are the parameters used to create or update the instance or binding.
	Parameters Key = "parameters"

	// Operation records there is an asynchronous operation in progress for the instance or binding.
	// This is the analogue to an operation.Type.
	Operation Key = "operation"

	// OperationID is the unique ID for an asynchronous operation on an instance or binding.
	OperationID Key = "operation-id"

	// OperationStatus is the error string returned by an aysynchronous operation.
	OperationStatus Key = "operation-status"
)

// Entry is a KV store associated with each instance or binding.
type Entry struct {
	// secret is the Kubernetes secret used to persist information.
	secret *corev1.Secret

	// exists indicates whether the entry existed in Kubernetes when it was created.
	exists bool

	// mutex handles synchronization when reading and writing to this entry concurrently.
	// In theory the only concurrency is when a provisioner is writing status and the invoking
	// handler is reading any values to return to the user, even then this set should be
	// mutually exclusive.  However in testing, async polling and provisioners reference the
	// same underlying storage, so it needs the locks to avoid race conditions.  In real life
	// the kubernetes client should return unique memory for each handler to use.
	mutex sync.Mutex
}

// Instance creates an entry for a service instance, or retrives an existing one.
func Instance(name string) (*Entry, error) {
	resourceName := "instance-" + name
	exists := true

	// Look up an existing config map.
	secret, err := config.Clients().Kubernetes().CoreV1().Secrets(config.Namespace()).Get(resourceName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}

		exists = false
	}

	// Create a new one if we need to.
	if !exists {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: resourceName,
				Labels: map[string]string{
					"app": labelBase,
				},
				Annotations: map[string]string{
					versionAnnotaiton: version.Version,
				},
			},
			Data: map[string][]byte{},
		}
	}

	entry := &Entry{
		secret: secret,
		exists: exists,
	}

	return entry, nil
}

// Exists indicates whether the entry existed in Kubernetes when it was created.
func (e *Entry) Exists() bool {
	return e.exists
}

// Commit persists the entry transaction to Kubernetes.
func (e *Entry) Commit() error {
        e.mutex.Lock()
        defer e.mutex.Unlock()

	if e.exists {
		secret, err := config.Clients().Kubernetes().CoreV1().Secrets(config.Namespace()).Update(e.secret)
		if err != nil {
			return err
		}

		e.secret = secret

		return nil
	}

	secret, err := config.Clients().Kubernetes().CoreV1().Secrets(config.Namespace()).Create(e.secret)
	if err != nil {
		return err
	}

	e.secret = secret
	e.exists = true

	return nil
}

// Delete removes the entry from Kubernetes.
func (e *Entry) Delete() error {
	if !e.exists {
		return nil
	}

	if err := config.Clients().Kubernetes().CoreV1().Secrets(config.Namespace()).Delete(e.secret.Name, metav1.NewDeleteOptions(0)); err != nil {
		return err
	}

	return nil
}

// annotationKey converts a key into an annotation key.
func annotationKey(key Key) string {
	return labelBase + "/" + string(key)
}

// Get gets a string from the entry.
func (e *Entry) Get(key Key) (string, bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	data, ok := e.secret.Data[annotationKey(key)]
	if !ok {
		return "", false
	}

	return string(data), true
}

// GetJSON gets and decodes a JSON object from the entry.
func (e *Entry) GetJSON(key Key, value interface{}) (bool, error) {
	data, ok := e.Get(key)
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal([]byte(data), value); err != nil {
		return true, err
	}

	return true, nil
}

// Set sets an entry item.
func (e *Entry) Set(key Key, value string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.secret.Data[annotationKey(key)] = []byte(value)
}

// SetJSON encodes a JSON object and sets the entry item.
func (e *Entry) SetJSON(key Key, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	e.Set(key, string(data))

	return nil
}

// Unset removes an item from the entry item.
func (e *Entry) Unset(key Key) {
	delete(e.secret.Data, annotationKey(key))
}

// GetOwnerReference returns the owner reference to attach to all resources created
// referenced by the template binding.
func (e *Entry) GetOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Secret",
		Name:       e.secret.Name,
		UID:        e.secret.UID,
	}
}
