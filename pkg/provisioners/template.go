// Copyright 2020 Couchbase, Inc.
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

package provisioners

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"text/template"
	"text/template/parse"
	"time"

	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/log"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/pkg/util"

	"github.com/go-openapi/jsonpointer"
	"github.com/golang/glog"
)

// templateFunctionRegistry looks up a registry value.
// Raises an error if the user does not have permissions to read the registry
// key or we encountered an unexpected internal error.  May return a nil value
// if the key does not exist.
func templateFunctionRegistry(entry *registry.Entry) func(string) (interface{}, error) {
	return func(key string) (interface{}, error) {
		glog.V(log.LevelDebug).Infof("registry: key '%s'", key)

		value, ok, err := entry.GetUser(key)
		if err != nil {
			return nil, errors.NewConfigurationError("registry read error: %v", err)
		}

		if !ok {
			return nil, nil
		}

		glog.V(log.LevelDebug).Infof("registry: value '%v'", value)

		return value, nil
	}
}

// templateFunctionParameter looks up a parameter.
// Raises an error if we encountered an unexpected internal error.  May return
// a nil value if the path does not exist.
func templateFunctionParameter(entry *registry.Entry) func(string) (interface{}, error) {
	return func(path string) (interface{}, error) {
		glog.V(log.LevelDebug).Infof("parameter: path '%s'", path)

		var parameters interface{}

		ok, err := entry.Get(registry.Parameters, &parameters)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, fmt.Errorf("unable to lookup parameters")
		}

		pointer, err := jsonpointer.New(path)
		if err != nil {
			return nil, errors.NewConfigurationError("json pointer malformed: %v", err)
		}

		value, _, err := pointer.Get(parameters)
		if err != nil {
			return nil, nil
		}

		glog.V(log.LevelDebug).Infof("parameter: value '%v'", value)

		return value, nil
	}
}

// templateFunctionSnippet recursively renders a template snippet.
// Returns an error if the template does not exist or the rendering of
// the template fialed.
func templateFunctionSnippet(entry *registry.Entry) func(name string) (interface{}, error) {
	return func(name string) (interface{}, error) {
		glog.V(log.LevelDebug).Infof("template: name '%s'", name)

		template, err := getTemplate(name)
		if err != nil {
			return nil, err
		}

		template, err = renderTemplate(template, entry)
		if err != nil {
			return nil, err
		}

		var value interface{}

		if err := json.Unmarshal(template.Template.Raw, &value); err != nil {
			return nil, errors.NewConfigurationError("template not JSON formatted: %v", err)
		}

		glog.V(log.LevelDebug).Infof("template: value '%v'", value)

		return value, nil
	}
}

// templateFunctionList makes a slice out of a variadic set of inputs.
func templateFunctionList(elements ...interface{}) []interface{} {
	return elements
}

// templateFunctionGeneratePassword generates a password.
func templateFunctionGeneratePassword(length int, dictionary interface{}) (string, error) {
	d := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	if dictionary != nil {
		typed, ok := dictionary.(string)
		if !ok {
			return "", errors.NewConfigurationError("password dictionary not a string")
		}

		d = typed
	}

	glog.V(log.LevelDebug).Infof("generatingPassword: length %d, dictionary '%s'", length, d)

	// Adjust so the length is within array bounds.
	arrayIndexOffset := 1
	dictionaryLength := len(d) - arrayIndexOffset

	limit := big.NewInt(int64(dictionaryLength))
	value := ""

	for i := 0; i < length; i++ {
		indexBig, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}

		if !indexBig.IsInt64() {
			return "", errors.NewConfigurationError("random index overflow")
		}

		index := int(indexBig.Int64())

		value += d[index : index+1]
	}

	glog.V(log.LevelDebug).Infof("generatePassword: value '%v'", value)

	return value, nil
}

// templateFunctionGeneratePrivatekey generates a private key.
func templateFunctionGeneratePrivatekey(typ, encoding string, bits interface{}) (string, error) {
	glog.V(log.LevelDebug).Infof("generatingPrivateKey: type '%s', encoding '%s', bits %v", typ, encoding, bits)

	var b *int

	if bits != nil {
		value, ok := bits.(int)
		if !ok {
			return "", errors.NewConfigurationError("bits is not an integer")
		}

		b = &value
	}

	key, err := util.GenerateKey(util.KeyType(typ), util.KeyEncodingType(encoding), b)
	if err != nil {
		return "", err
	}

	value := string(key)

	glog.V(log.LevelDebug).Infof("generatePrivateKey: value '%v'", value)

	return value, nil
}

// templateFunctionGenerateCertificate generates a certiifcate.
func templateFunctionGenerateCertificate(key, cn, lifetime, usage string, sans []interface{}, caKey, caCert interface{}) (string, error) {
	glog.V(log.LevelDebug).Infof("generateCertificate: key '%s', cn '%s', lifetime '%s', usage '%s', sans %v, ca key '%s', ca cert '%s'", key, cn, lifetime, usage, sans, caKey, caCert)

	duration, err := time.ParseDuration(lifetime)
	if err != nil {
		return "", err
	}

	var caKeyTyped []byte

	if caKey != nil {
		t, ok := caKey.(string)
		if !ok {
			return "", errors.NewConfigurationError("CA key not a string")
		}

		caKeyTyped = []byte(t)
	}

	var caCertTyped []byte

	if caCert != nil {
		t, ok := caCert.(string)
		if !ok {
			return "", errors.NewConfigurationError("CA certificate not a string")
		}

		caCertTyped = []byte(t)
	}

	sansTyped := make([]string, len(sans))

	for index, san := range sans {
		t, ok := san.(string)
		if !ok {
			return "", errors.NewConfigurationError("SAN %v not a strings", san)
		}

		sansTyped[index] = t
	}

	cert, err := util.GenerateCertificate([]byte(key), cn, duration, util.CertificateUsage(usage), sansTyped, caKeyTyped, caCertTyped)
	if err != nil {
		return "", err
	}

	value := string(cert)

	glog.V(log.LevelDebug).Infof("generateCertificate: value '%v'", value)

	return value, nil
}

// templateFunctionRequired returns an error if the input is nil.
func templateFunctionRequired(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, errors.NewConfigurationError("required value is nil")
	}

	return value, nil
}

// templateFunctionGenerateDefault sets a default if its input is nil.
func templateFunctionGenerateDefault(def, value interface{}) interface{} {
	glog.V(log.LevelDebug).Infof("default: default '%v',  value '%v'", def, value)

	if value == nil {
		value = def
	}

	glog.V(log.LevelDebug).Infof("default: value '%v'", value)

	return def
}

// templateFunctionGenerateJSON marshals template output into a JSON string.  As template
// processing assumes the output is a string, we have to encode to JSON to preserve structure
// as a string.
func templateFunctionGenerateJSON(object interface{}) (string, error) {
	glog.V(log.LevelDebug).Infof("json: object '%v'", object)

	raw, err := json.Marshal(object)
	if err != nil {
		return "", err
	}

	value := string(raw)

	glog.V(log.LevelDebug).Infof("json: value '%v'", value)

	return value, nil
}

const (
	// templatePrefix denotes the start of a Go template.
	templatePrefix = "{{"

	// templateSuffix denotes the end of a Go template.
	templateSuffix = "}}"
)

// jsonify is a command to be appended to actions in the template
// parse tree.
var jsonify = &parse.CommandNode{
	NodeType: parse.NodeCommand,
	Args: []parse.Node{
		parse.NewIdentifier("json"),
	},
}

// transformActionsToJSON walks the parse tree and finds any actions that
// would usually generate text output.  These are appened with a JSON function
// that turns the abstract type into a JSON string, preserving type for later
// decoding and patching into the resource structure.
func transformActionsToJSON(n parse.Node) {
	if n == nil {
		return
	}

	switch node := n.(type) {
	case *parse.ActionNode:
		if node.Pipe != nil && len(node.Pipe.Decl) == 0 {
			node.Pipe.Cmds = append(node.Pipe.Cmds, jsonify)
		}
	case *parse.BranchNode:
		transformActionsToJSON(node.List)
		transformActionsToJSON(node.ElseList)
	case *parse.CommandNode:
		for _, arg := range node.Args {
			transformActionsToJSON(arg)
		}
	case *parse.IfNode:
		transformActionsToJSON(node.BranchNode.List)
		transformActionsToJSON(node.BranchNode.ElseList)
	case *parse.ListNode:
		for _, item := range node.Nodes {
			transformActionsToJSON(item)
		}
	case *parse.PipeNode:
		for _, cmd := range node.Cmds {
			transformActionsToJSON(cmd)
		}
	}
}

// renderTemplateString takes a string and returns either the literal value if it's
// not a template or the object returned after template rendering.
func renderTemplateString(str string, entry *registry.Entry) (interface{}, error) {
	// Template expansion must occur in a string, and it must be all one template.
	if !strings.HasPrefix(str, templatePrefix) {
		return str, nil
	}

	if !strings.HasSuffix(str, templateSuffix) {
		return nil, errors.NewConfigurationError("dynamic attribute '%s' malformed", str)
	}

	glog.V(log.LevelDebug).Infof("resolving dynamic attribute %s", str)

	funcs := map[string]interface{}{
		"registry":            templateFunctionRegistry(entry),
		"parameter":           templateFunctionParameter(entry),
		"snippet":             templateFunctionSnippet(entry),
		"list":                templateFunctionList,
		"generatePassword":    templateFunctionGeneratePassword,
		"generatePrivateKey":  templateFunctionGeneratePrivatekey,
		"generateCertificate": templateFunctionGenerateCertificate,
		"required":            templateFunctionRequired,
		"default":             templateFunctionGenerateDefault,
		"json":                templateFunctionGenerateJSON,
	}

	tmpl, err := template.New("inline template").Funcs(funcs).Parse(str)
	if err != nil {
		return nil, err
	}

	// Implictly add in a JSON transformation to preserve type and structure.
	transformActionsToJSON(tmpl.Root)

	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, nil); err != nil {
		return nil, errors.NewConfigurationError("dynamic attribute resolution failed: %v", err)
	}

	var value interface{}

	if err := json.Unmarshal(buf.Bytes(), &value); err != nil {
		return nil, err
	}

	return value, nil
}

// recurseRenderTemplate takes a template and recursively walks the data structure.
// Templates are passed around as JSON, therefore structured objects so have the
// benefit of having free error checking.  Strings are special in that they may be
// go templates that can resolve to an arbitrary JSON structure to replace the
// template string.
func recurseRenderTemplate(object interface{}, entry *registry.Entry) (interface{}, error) {
	// For maps and lists, recursovely render each value and replace with what is returned.
	// Strings are special and may undergo templating.
	switch t := object.(type) {
	case map[string]interface{}:
		for k, v := range t {
			value, err := recurseRenderTemplate(v, entry)
			if err != nil {
				return nil, err
			}

			if value == nil {
				delete(t, k)
				break
			}

			t[k] = value
		}
	case []interface{}:
		for i, v := range t {
			value, err := recurseRenderTemplate(v, entry)
			if err != nil {
				return nil, err
			}

			t[i] = value
		}
	case string:
		value, err := renderTemplateString(t, entry)
		if err != nil {
			return nil, err
		}

		return value, nil
	}

	return object, nil
}
