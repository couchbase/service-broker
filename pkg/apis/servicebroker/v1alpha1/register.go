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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	ServiceBrokerConfigKind     = "ServiceBrokerConfig"
	ServiceBrokerConfigResource = "servicebrokerconfigs"
	GroupVersion                = "v1alpha1"
	GroupName                   = "servicebroker.couchbase.com"
	Group                       = GroupName + "/" + GroupVersion
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&ServiceBrokerConfig{}, &ServiceBrokerConfigList{})
}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
