package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	ServiceBrokerConfigKind = "CouchbaseServiceBrokerConfig"
	ServiceBrokerConfigName = "couchbaseservicebrokerconfigs"
	GroupVersion            = "v1"
	GroupName               = "broker.couchbase.com"
	Group                   = GroupName + "/" + GroupVersion
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&CouchbaseServiceBrokerConfig{}, &CouchbaseServiceBrokerConfigList{})
}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
