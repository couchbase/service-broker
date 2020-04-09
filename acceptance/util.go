package acceptance

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ghodss/yaml"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// readYAMLObjects reads in a YAML file and unmarshals as unstructured objects.
func readYAMLObjects(path string) ([]*unstructured.Unstructured, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	objects := []*unstructured.Unstructured{}

	sections := strings.Split(string(data), "---")
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}

		object := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(section), object); err != nil {
			return nil, err
		}

		objects = append(objects, object)
	}

	return objects, nil
}

// mustReadYAMLObjects reads in a YAML file and unmarshals as unstructured objects.
func mustReadYAMLObjects(t *testing.T, path string) []*unstructured.Unstructured {
	objects, err := readYAMLObjects(path)
	if err != nil {
		t.Fatal(err)
	}

	return objects
}
