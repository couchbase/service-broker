package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/couchbase/service-broker/pkg/util"

	"github.com/ghodss/yaml"
)

var (
	crds []interface{}
)

// buffer post processes raw output strings and buffers them
func buffer(crd interface{}) {
	crds = append(crds, crd)
}

// dump formats the CRDs as YAML and echos to standard out
func dump() error {
	var yamls []string

	for _, crd := range crds {
		data, err := yaml.Marshal(crd)
		if err != nil {
			return err
		}
		// Hack: the status attribute is formatted, but the API rejects this so
		// we need a way to rid ourselves of it.
		parts := strings.Split(string(data), "\nstatus:\n")
		yamls = append(yamls, parts[0])
	}

	fmt.Println(strings.Join(yamls, "\n---\n"))
	return nil
}

func main() {
	buffer(util.GetCouchbaseServiceBrokerConfigCRD())
	if err := dump(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
