// Copyright 2021 Couchbase, Inc.
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

package registry

import (
	"encoding/json"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/version"

	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	directoryName = "couchbase-service-broker-directory"
)

// Directory is a lookup table used to locate the registry entries for a
// service instance.  The registry must live in the same namespace as the
// resources it is creating in order for garbage collection to work
// correctly.  We can only determine this when a context is provided by
// the API telling us the namespace a service instance is provisioned in to.
// To further compound the issue, the context is only provided on creation,
// so we need to cache where the registry exists, in a fixed location we
// can always access.
type Directory struct {
	// secret is the local cached version of the directory.
	// The data maps instance ID to a namespace.
	secret *corev1.Secret

	// exists describes whether the directory already exists in etcd.
	exists bool
}

// DirectoryEntry contains persistent data about a service instance.
type DirectoryEntry struct {
	// Namespace is the namespace in which the service instance,
	// and registry entries, reside.
	Namespace string `json:"namespace"`
}

// NewDirectory lookups up or creates the registry directory.
func NewDirectory(namespace string) (*Directory, error) {
	exists := true

	secret, err := config.Clients().Kubernetes().CoreV1().Secrets(namespace).Get(directoryName, metav1.GetOptions{})
	if err != nil {
		if !k8s_errors.IsNotFound(err) {
			return nil, err
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      directoryName,
				Namespace: namespace,
				Labels: map[string]string{
					"app": version.Application,
				},
				Annotations: map[string]string{
					v1.VersionAnnotaiton: version.Version,
				},
			},
		}

		exists = false
	}

	directory := &Directory{
		secret: secret,
		exists: exists,
	}

	return directory, nil
}

// commit writes out the cached directory.
func (d *Directory) commit() error {
	if d.exists {
		secret, err := config.Clients().Kubernetes().CoreV1().Secrets(d.secret.Namespace).Update(d.secret)
		if err != nil {
			return err
		}

		d.secret = secret

		return nil
	}

	secret, err := config.Clients().Kubernetes().CoreV1().Secrets(d.secret.Namespace).Create(d.secret)
	if err != nil {
		return err
	}

	d.secret = secret
	d.exists = true

	return nil
}

// Add registers a directory entry for a service instance.  This should only ever
// be set for a service instance creation.
func (d *Directory) Add(instanceID string, dirent *DirectoryEntry) error {
	if d.secret.Data == nil {
		d.secret.Data = map[string][]byte{}
	}

	data, err := json.Marshal(dirent)
	if err != nil {
		return err
	}

	d.secret.Data[instanceID] = data

	return d.commit()
}

// Lookup finds the directory entry registered for a service instance.
func (d *Directory) Lookup(instanceID string) (*DirectoryEntry, error) {
	if d.secret.Data == nil {
		return nil, errors.NewResourceNotFoundError("directory entry missing for %s", instanceID)
	}

	data, ok := d.secret.Data[instanceID]
	if !ok {
		return nil, errors.NewResourceNotFoundError("directory entry missing for %s", instanceID)
	}

	dirent := &DirectoryEntry{}
	if err := json.Unmarshal(data, dirent); err != nil {
		return nil, err
	}

	return dirent, nil
}

// Remove cleans out a service instance entry from the directory.
func (d *Directory) Remove(instanceID string) error {
	if d.secret.Data == nil {
		return errors.NewResourceNotFoundError("directory entry missing for %s", instanceID)
	}

	if _, ok := d.secret.Data[instanceID]; !ok {
		return errors.NewResourceNotFoundError("directory entry missing for %s", instanceID)
	}

	delete(d.secret.Data, instanceID)

	return d.commit()
}
