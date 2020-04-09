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
