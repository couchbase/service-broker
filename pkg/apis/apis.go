package apis

import (
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

func AddToScheme(s *runtime.Scheme) error {
	schemeBuilders := runtime.SchemeBuilder{
		v1.AddToScheme,
	}

	return schemeBuilders.AddToScheme(s)
}
