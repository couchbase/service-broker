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

// Package registry is the persistence layer of the service broker.
package registry

import (
	"encoding/json"
	goerrors "errors"
	"fmt"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/version"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Key is an indentifier of a value in the registry entry's KV map.
type Key string

const (
	// Namespace is the namespace assigned to the instance.
	Namespace Key = "namespace"

	// InstanceID is the name of the service.
	InstanceID Key = "instance-id"

	// BindingID is the name of the binding.
	BindingID Key = "binding-id"

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

	// DashboardURL is the dashboard URL associated with a service instance.
	DashboardURL Key = "dashboard-url"

	// Credentials is the set of credentials that may be generated for a service binding.
	Credentials Key = "credentials"
)

// ErrPermsission is raised when you don't have permission to read/write a registry key.
var ErrPermsission = goerrors.New("permission error")

// keyPolicy defines managed keys and how they can be accessed by users.
type keyPolicy struct {
	// name is the name of the key.
	name Key

	// read defines whether a user can read a specific key.
	read bool

	// write defines whether a user can write a specifc key.
	write bool
}

var (
	// keyPolicies defines whether users can modify managed keys.
	keyPolicies = []keyPolicy{
		{
			name:  Namespace,
			read:  true,
			write: false,
		},
		{
			name:  InstanceID,
			read:  true,
			write: false,
		},
		{
			name:  ServiceID,
			read:  true,
			write: false,
		},
		{
			name:  PlanID,
			read:  true,
			write: false,
		},
		{
			name:  Context,
			read:  false,
			write: false,
		},
		{
			name:  Parameters,
			read:  false,
			write: false,
		},
		{
			name:  Operation,
			read:  false,
			write: false,
		},
		{
			name:  OperationID,
			read:  false,
			write: false,
		},
		{
			name:  OperationStatus,
			read:  false,
			write: false,
		},
		{
			name:  DashboardURL,
			read:  true,
			write: true,
		},
		{
			name:  Credentials,
			read:  true,
			write: true,
		},
	}
)

// findKeyPolicy looks up a defined key policy.
func findKeyPolicy(name string) *keyPolicy {
	for index := range keyPolicies {
		if keyPolicies[index].name == Key(name) {
			return &keyPolicies[index]
		}
	}

	return nil
}

// isKeyWritable checks to see whether a key can be read.
func isKeyReadable(name string) bool {
	policy := findKeyPolicy(name)
	if policy == nil {
		return true
	}

	return policy.read
}

// isKeyWritable checks to see whether a key can be written.
func isKeyWritable(name string) bool {
	policy := findKeyPolicy(name)
	if policy == nil {
		return true
	}

	return policy.write
}

// Type defines the registry type.
type Type string

const (
	// ServiceInstance is used for service instance registries.
	ServiceInstance Type = "service-instance"

	// ServiceBinding is used for service instance registries.
	ServiceBinding Type = "service-binding"
)

// Entry is a KV store associated with each instance or binding.
type Entry struct {
	// secret is the Kubernetes secret used to persist information.
	secret *corev1.Secret

	// exists indicates whether the entry existed in Kubernetes when it was created.
	exists bool

	// readOnly indicates whether this instance is read only.
	// Once set it cannot be unset.  Read only instances cannot be deleted or
	// updated.
	readOnly bool
}

// Name returns the name of the registry secret.
func Name(t Type, name string) string {
	return "registry-" + string(t) + "-" + name
}

// New creates a registry entry, or retrives an existing one.
func New(t Type, namespace, name string, readOnly bool) (*Entry, error) {
	resourceName := Name(t, name)
	exists := true

	// Look up an existing config map.
	secret, err := config.Clients().Kubernetes().CoreV1().Secrets(namespace).Get(resourceName, metav1.GetOptions{})
	if err != nil {
		if !k8s_errors.IsNotFound(err) {
			return nil, err
		}

		exists = false
	}

	// Create a new one if we need to.
	if !exists {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
				Labels: map[string]string{
					"app": version.Application,
				},
				Annotations: map[string]string{
					v1.VersionAnnotaiton: version.Version,
				},
			},
		}
	}

	entry := &Entry{
		secret:   secret,
		exists:   exists,
		readOnly: readOnly,
	}

	return entry, nil
}

// Clone duplicates a registry entry, the clone is read only to allow concurrency
// while the master copy retains its read/write status.
func (e *Entry) Clone() *Entry {
	return &Entry{
		secret:   e.secret.DeepCopy(),
		exists:   e.exists,
		readOnly: true,
	}
}

// Inherit is used when creating a service binding registry entry.  It gets a copy
// of all data cached in the service instance.
func (e *Entry) Inherit(o *Entry) {
	if o.secret.Data == nil {
		return
	}

	if e.secret.Data == nil {
		e.secret.Data = map[string][]byte{}
	}

	for k, v := range o.secret.Data {
		e.secret.Data[k] = v
	}
}

// Exists indicates whether the entry existed in Kubernetes when it was created.
func (e *Entry) Exists() bool {
	return e.exists
}

// Commit persists the entry transaction to Kubernetes.
func (e *Entry) Commit() error {
	if e.readOnly {
		return fmt.Errorf("%w: registry entry is read only", ErrPermsission)
	}

	if e.exists {
		secret, err := config.Clients().Kubernetes().CoreV1().Secrets(e.secret.Namespace).Update(e.secret)
		if err != nil {
			return err
		}

		e.secret = secret

		return nil
	}

	secret, err := config.Clients().Kubernetes().CoreV1().Secrets(e.secret.Namespace).Create(e.secret)
	if err != nil {
		return err
	}

	e.secret = secret
	e.exists = true

	return nil
}

// Delete removes the entry from Kubernetes.
func (e *Entry) Delete() error {
	if e.readOnly {
		return fmt.Errorf("%w: registry entry is read only", ErrPermsission)
	}

	if !e.exists {
		return nil
	}

	if err := config.Clients().Kubernetes().CoreV1().Secrets(e.secret.Namespace).Delete(e.secret.Name, metav1.NewDeleteOptions(0)); err != nil {
		return err
	}

	return nil
}

// Get gets a string from the entry.
func (e *Entry) GetString(key Key) (string, bool, error) {
	var value string

	ok, err := e.Get(key, &value)
	if !ok || err != nil {
		return "", ok, err
	}

	return value, true, nil
}

// Get gets an entry item.
func (e *Entry) Get(key Key, value interface{}) (bool, error) {
	if e.secret.Data == nil {
		return false, nil
	}

	data, ok := e.secret.Data[string(key)]
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(data, value); err != nil {
		return true, err
	}

	return true, nil
}

// Set sets an entry item.
func (e *Entry) Set(key Key, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if e.secret.Data == nil {
		e.secret.Data = map[string][]byte{}
	}

	e.secret.Data[string(key)] = data

	return nil
}

// GetUser gets and decodes a JSON object from the registry.
func (e *Entry) GetUser(key string) (interface{}, bool, error) {
	if !isKeyReadable(key) {
		return "", false, errors.NewConfigurationError("registry key %s cannot be read", key)
	}

	var value interface{}

	ok, err := e.Get(Key(key), &value)
	if !ok || err != nil {
		return nil, ok, err
	}

	return value, true, nil
}

// SetUser encodes a JSON object and sets the entry item.
func (e *Entry) SetUser(key string, value interface{}) error {
	glog.Infof("setting registry entry %s to %s", key, value)

	if !isKeyWritable(key) {
		return errors.NewConfigurationError("registry key %s cannot be written", key)
	}

	return e.Set(Key(key), value)
}

// Unset removes an item from the entry item.
func (e *Entry) Unset(key Key) {
	delete(e.secret.Data, string(key))
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
