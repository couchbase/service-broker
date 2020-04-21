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

package fixtures

import (
	"fmt"
)

// argument encodes an argument as per the go text/template library.
// Only scalar types are supported.
// https://golang.org/pkg/text/template/#hdr-Arguments
func argument(value interface{}) string {
	switch t := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, t)
	case int:
		return fmt.Sprintf(`%d`, t)
	case bool:
		if t {
			return `true`
		}

		return `false`
	case nil:
		return `nil`
	}

	return ``
}

// Pipeline represents a pipeline of of go template functions.
// https://golang.org/pkg/text/template/#hdr-Pipelines
type Pipeline string

// NewPipeline creates a new pipeline from a function.  While a pipeline can
// start with an argument, all our data accessors are through functions so
// this is the common path.
func NewPipeline(fn Function) Pipeline {
	return Pipeline(fn)
}

// NewRegistryPipeline creates a pipeline initialized with a registry lookup
// function.
func NewRegistryPipeline(arg interface{}) Pipeline {
	return NewPipeline(Registry(arg))
}

// NewParameterPipeline creates a pipeline initialized with a parameter lookup
// function.
func NewParameterPipeline(arg interface{}) Pipeline {
	return NewPipeline(Parameter(arg))
}

// NewGeneratePasswordPipeline creates a pipeline initialized with a generate
// password function.
func NewGeneratePasswordPipeline(length, dictionary interface{}) Pipeline {
	return NewPipeline(GeneratePassword(length, dictionary))
}

// NewGeneratePrivateKeyPipeline creates a pipeline initialized with a generate
// private key function.
func NewGeneratePrivateKeyPipeline(typ, encoding, bits interface{}) Pipeline {
	return NewPipeline(GeneratePrivateKey(typ, encoding, bits))
}

// NewGenerateCertificatePipeline creates a pipeline initialized with a generate
// certificate function.
func NewGenerateCertificatePipeline(key, cn, lifetime, usage, sans, caKey, caCert interface{}) Pipeline {
	return NewPipeline(GenerateCertificate(key, cn, lifetime, usage, sans, caKey, caCert))
}

// With appends a function to a pipeline.
func (p Pipeline) With(fn Function) Pipeline {
	if p == "" {
		return Pipeline(fn)
	}

	return Pipeline(string(p) + " | " + string(fn))
}

// WithDefault appends a defaulting function to a pipeline.
func (p Pipeline) WithDefault(arg interface{}) Pipeline {
	return p.With(Default(arg))
}

// Required appends a function to a pipeline that sets a default if the
// input is nil.
func (p Pipeline) Required() Pipeline {
	return p.With(Required())
}

// Function represents a named function that accepts an arbitrary number
// of arguments.
// https://golang.org/pkg/text/template/#hdr-Functions
type Function string

// NewFunction creates a new named function.
func NewFunction(fn string, args ...interface{}) Function {
	expression := fn

	for _, arg := range args {
		switch t := arg.(type) {
		case Function:
			expression = fmt.Sprintf("%s (%s)", expression, string(t))
		case Pipeline:
			expression = fmt.Sprintf("%s (%s)", expression, string(t))
		case string, int, bool, nil:
			expression = fmt.Sprintf("%s %s", expression, argument(t))
		}
	}

	return Function(expression)
}

// Registry returns a function that looks up a registry entry.
func Registry(arg interface{}) Function {
	return NewFunction("registry", arg)
}

// Parameter returns a function that looks up a parameter path.
func Parameter(arg interface{}) Function {
	return NewFunction("parameter", arg)
}

// GeneratePassword returns a function that generates a random password string.
func GeneratePassword(length, dictionary interface{}) Function {
	return NewFunction("generatePassword", length, dictionary)
}

// GeneratePrivateKey returns a function that generates a private key.
func GeneratePrivateKey(typ, encoding, bits interface{}) Function {
	return NewFunction("generatePrivateKey", typ, encoding, bits)
}

// GenerateCertificate returns a function that generates a certificate.
func GenerateCertificate(key, cn, lifetime, usage, sans, caKey, caCert interface{}) Function {
	return NewFunction("generateCertificate", key, cn, lifetime, usage, sans, caKey, caCert)
}

// Default generates a function that returns a default if the input it nil.
func Default(arg interface{}) Function {
	return NewFunction("default", arg)
}

// Required returns a function that raises an error if the input is nil.
func Required() Function {
	return NewFunction(`required`)
}
