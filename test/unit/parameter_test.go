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

package unit_test

import (
	"crypto/x509"
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/test/unit/fixtures"
	"github.com/couchbase/service-broker/test/unit/util"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// key is the name of the registry key we will create.
	key = "animal"

	// value is the value of the registry key we will create.
	value = "cat"

	// defaultValue is the default value for the registry key to use.
	defaultValue = "kitten"

	// defaultPasswordLength is used to test password generation.
	// Pick a random prime as that's unlikely to be a default ever!
	defaultPasswordLength = 23

	// defaultPasswordDictionary is the service broker default for password generation.
	defaultPasswordDictionary = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// defaultCN is the common name for a certificate.
	defaultCN = "test common name"
)

var (
	// caKeyKey is the key name used for private keys.
	caKeyKey = "ca.key"

	// caCertificateKey is the key name used for certificates.
	caCertificateKey = "ca.pem"

	// childKeyKey is the key name used for child private keys.
	childKeyKey = "child.key"

	// childCertificateKey is the key name used for child certificates.
	childCertificateKey = "child.pem"

	// defaultKeyLength is the default key length for RSA keys.  Kept small
	// because it's faster, entropy and all.  Anything smaller that 512 will
	// cause failures when generating certificates.
	defaultKeyLength = 512

	// customPasswordDictionary is a bucnh of stuff that isn't default.
	customPasswordDictionary = "!@#$%^&*()_+"
)

// TestParameters tests parameter items are correctly populated by service instance
// creation.
func TestParameters(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewParameterPipeline("/animal"))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"` + key + `":"` + value + `"}`),
	}
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.Key(key), value)
}

// TestParametersMissingPath tests parameter items are correctly populated by service instance
// creation.
func TestParametersMissingPath(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewParameterPipeline("/animal"))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustNotHaveRegistryEntry(t, entry, registry.Key(key))
}

// TestParametersMissingRequiredPath tests parameter items are correctly populated by service instance
// creation.
func TestParametersMissingRequiredPath(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewParameterPipeline("/animal").Required())
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParametersDefault tests a parameter with a default work when not specified.
func TestParametersDefault(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewParameterPipeline("/animal").WithDefault(defaultValue))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.Key(key), defaultValue)
}

// TestParametersDefaultOverride tests a parameter with a default work when specified.
func TestParametersDefaultOverride(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewParameterPipeline("/animal").WithDefault(defaultValue))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	req.Parameters = &runtime.RawExtension{
		Raw: []byte(`{"` + key + `":"` + value + `"}`),
	}
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.Key(key), value)
}

// TestParameterGenerateKeyRSAPKCS1 tests we can generate PKCS#1 formatted RSA keys.
func TestParameterGenerateKeyRSAPKCS1(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyRSAPKCS8 tests we can generate PKCS#8 formatted RSA keys.
func TestParameterGenerateKeyRSAPKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyRSAEC tests we can't generate EC formatted RSA keys.
func TestParameterGenerateKeyRSAECInvalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("RSA", "SEC 1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyRSAMissingLengthInvalid tests we handle missing RSA key length gracefully.
func TestParameterGenerateKeyRSAMissingLengthInvalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#1", nil))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP224PKCS1Invalid tests we can't generate PKCS#1 formatted EC keys.
func TestParameterGenerateKeyEllipticP224PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP224", "PKCS#1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP224PKCS8 tests we can generate PKCS#8 formatted EC keys.
func TestParameterGenerateKeyEllipticP224PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP224", "PKCS#8", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP224EC tests we can generate EC formatted EC keys.
func TestParameterGenerateKeyEllipticP224EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP224", "SEC 1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP256PKCS1Invalid tests we can't generate PKCS#1 formatted EC keys.
func TestParameterGenerateKeyEllipticP256PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP256", "PKCS#1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP256PKCS8 tests we can generate PKCS#8 formatted EC keys.
func TestParameterGenerateKeyEllipticP256PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP256", "PKCS#8", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP256EC tests we can generate EC formatted EC keys.
func TestParameterGenerateKeyEllipticP256EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP256", "SEC 1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP384PKCS1Invalid tests we can't generate PKCS#1 formatted EC keys.
func TestParameterGenerateKeyEllipticP384PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP384", "PKCS#1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP384PKCS8 tests we can generate PKCS#8 formatted EC keys.
func TestParameterGenerateKeyEllipticP384PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP384", "PKCS#8", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP384EC tests we can generate EC formatted EC keys.
func TestParameterGenerateKeyEllipticP384EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP384", "SEC 1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP512PKCS1Invalid tests we can't generate PKCS#1 formatted EC keys.
func TestParameterGenerateKeyEllipticP521PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP521", "PKCS#1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP521PKCS8 tests we can generate PKCS#8 formatted EC keys.
func TestParameterGenerateKeyEllipticP521PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP521", "PKCS#8", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP512EC tests we can generate EC formatted EC keys.
func TestParameterGenerateKeyEllipticP521EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("EllipticP521", "SEC 1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticED25519PKCS1Invalid tests we can't generate PKCS#1 formatted ED keys.
func TestParameterGenerateKeyED25519PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("ED25519", "PKCS#1", defaultKeyLength))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticED25519PKCS8 tests we can generate PKCS#8 formatted ED keys.
func TestParameterGenerateKeyED25519PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("ED25519", "PKCS#8", nil))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticED25519EC tests we can't generate EC formatted ED keys.
func TestParameterGenerateKeyED25519ECInvalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePrivateKeyPipeline("ED25519", "SEC 1", nil))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateCACertificateRSAPKCS1 tests that we can create a CA certificate with an
// RSA private key.
func TestParameterGenerateCACertificateRSAPKCS1(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#1", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLS(t, entry, registry.Key(caKeyKey), registry.Key(caCertificateKey))
}

// TestParameterGenerateCACertificateRSAPKCS8 tests that we can create a CA certificate with an
// RSA private key.
func TestParameterGenerateCACertificateRSAPKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLS(t, entry, registry.Key(caKeyKey), registry.Key(caCertificateKey))
}

// TestParameterGenerateCACertificateEllipticP224EC tests that we can create a CA certificate with an
// EC private key.
func TestParameterGenerateCACertificateEllipticP224EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("EllipticP224", "SEC 1", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLS(t, entry, registry.Key(caKeyKey), registry.Key(caCertificateKey))
}

// TestParameterGenerateCACertificateED25519PKCS8Invalid tests that creating a CA certificate with an
// ED private key is invalid.
func TestParameterGenerateCACertificateED25519PKCS8Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("ED25519", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateServerCertificateRSAPKCS1 tests that we can create a server certificate with an
// RSA private key.
func TestParameterGenerateServerCertificateRSAPKCS1(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#1", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	fixtures.AddRegistry(configuration, childKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#1", defaultKeyLength))
	fixtures.AddRegistry(configuration, childCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(childKeyKey), defaultCN, "24h", "Server", nil, fixtures.Registry(caKeyKey), fixtures.Registry(caCertificateKey)))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLSAndVerify(t, entry, registry.Key(caCertificateKey), registry.Key(childKeyKey), registry.Key(childCertificateKey), x509.ExtKeyUsageServerAuth)
}

// TestParameterGenerateServerCertificateRSAPKCS8 tests that we can create a server certificate with an
// RSA private key.
func TestParameterGenerateServerCertificateRSAPKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	fixtures.AddRegistry(configuration, childKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, childCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(childKeyKey), defaultCN, "24h", "Server", nil, fixtures.Registry(caKeyKey), fixtures.Registry(caCertificateKey)))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLSAndVerify(t, entry, registry.Key(caCertificateKey), registry.Key(childKeyKey), registry.Key(childCertificateKey), x509.ExtKeyUsageServerAuth)
}

// TestParameterGenerateServerCertificateEllipticP224EC tests that we can create a server certificate
// with an elliptic private key.
func TestParameterGenerateServerCertificateEllipticP224EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("EllipticP224", "SEC 1", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	fixtures.AddRegistry(configuration, childKeyKey, fixtures.NewGeneratePrivateKeyPipeline("EllipticP224", "SEC 1", defaultKeyLength))
	fixtures.AddRegistry(configuration, childCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(childKeyKey), defaultCN, "24h", "Server", nil, fixtures.Registry(caKeyKey), fixtures.Registry(caCertificateKey)))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLSAndVerify(t, entry, registry.Key(caCertificateKey), registry.Key(childKeyKey), registry.Key(childCertificateKey), x509.ExtKeyUsageServerAuth)
}

// TestParameterGenerateServerCertificateRSAPKCS8WithSANs tests that we can create a server certificate with an
// RSA private key.
func TestParameterGenerateServerCertificateRSAPKCS8WithSANs(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	fixtures.AddRegistry(configuration, childKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, childCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(childKeyKey), defaultCN, "24h", "Server", fixtures.NewFunction("list", "DNS:localhost", "DNS:bugs.looneytunes.com"), fixtures.Registry(caKeyKey), fixtures.Registry(caCertificateKey)))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLSAndVerify(t, entry, registry.Key(caCertificateKey), registry.Key(childKeyKey), registry.Key(childCertificateKey), x509.ExtKeyUsageServerAuth)
}

// TestParameterGenerateClientCertificateRSAPKCS1 tests that we can create a client certificate with an
// RSA private key.
func TestParameterGenerateClientCertificateRSAPKCS1(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#1", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	fixtures.AddRegistry(configuration, childKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#1", defaultKeyLength))
	fixtures.AddRegistry(configuration, childCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(childKeyKey), defaultCN, "24h", "Client", nil, fixtures.Registry(caKeyKey), fixtures.Registry(caCertificateKey)))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLSAndVerify(t, entry, registry.Key(caCertificateKey), registry.Key(childKeyKey), registry.Key(childCertificateKey), x509.ExtKeyUsageClientAuth)
}

// TestParameterGenerateClientCertificateRSAPKCS8 tests that we can create a client certificate with an
// RSA private key.
func TestParameterGenerateClientCertificateRSAPKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	fixtures.AddRegistry(configuration, childKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, childCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(childKeyKey), defaultCN, "24h", "Client", nil, fixtures.Registry(caKeyKey), fixtures.Registry(caCertificateKey)))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLSAndVerify(t, entry, registry.Key(caCertificateKey), registry.Key(childKeyKey), registry.Key(childCertificateKey), x509.ExtKeyUsageClientAuth)
}

// TestParameterGenerateClientCertificateEllipticP224EC tests that we can create a client certificate
// with an elliptic private key.
func TestParameterGenerateClientCertificateEllipticP224EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("EllipticP224", "SEC 1", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	fixtures.AddRegistry(configuration, childKeyKey, fixtures.NewGeneratePrivateKeyPipeline("EllipticP224", "SEC 1", defaultKeyLength))
	fixtures.AddRegistry(configuration, childCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(childKeyKey), defaultCN, "24h", "Client", nil, fixtures.Registry(caKeyKey), fixtures.Registry(caCertificateKey)))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLSAndVerify(t, entry, registry.Key(caCertificateKey), registry.Key(childKeyKey), registry.Key(childCertificateKey), x509.ExtKeyUsageClientAuth)
}

// TestParameterGenerateClientCertificateRSAPKCS8WithSANs tests that we can create a clientcertificate with an
// RSA private key.
func TestParameterGenerateClientCertificateRSAPKCS8WithSANs(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, caKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, caCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(caKeyKey), defaultCN, "24h", "CA", nil, nil, nil))
	fixtures.AddRegistry(configuration, childKeyKey, fixtures.NewGeneratePrivateKeyPipeline("RSA", "PKCS#8", defaultKeyLength))
	fixtures.AddRegistry(configuration, childCertificateKey, fixtures.NewGenerateCertificatePipeline(fixtures.Registry(childKeyKey), defaultCN, "24h", "Client", fixtures.NewFunction("list", "EMAIL:bugs.bunny@looneytunes.com"), fixtures.Registry(caKeyKey), fixtures.Registry(caCertificateKey)))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntriesTLSAndVerify(t, entry, registry.Key(caCertificateKey), registry.Key(childKeyKey), registry.Key(childCertificateKey), x509.ExtKeyUsageClientAuth)
}

// TestParameterGeneratePassword tests that password generation works.
func TestParameterGeneratePassword(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePasswordPipeline(defaultPasswordLength, nil))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryPassword(t, entry, key, defaultPasswordLength, defaultPasswordDictionary)
}

// TestParameterGeneratePasswordWithCustomDictionary tests that password generation works.
func TestParameterGeneratePasswordWithCustomDictionary(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	fixtures.SetRegistry(configuration, key, fixtures.NewGeneratePasswordPipeline(defaultPasswordLength, customPasswordDictionary))
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryPassword(t, entry, key, defaultPasswordLength, customPasswordDictionary)
}
