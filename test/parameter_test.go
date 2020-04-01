package test

import (
	"crypto/x509"
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/test/fixtures"
	"github.com/couchbase/service-broker/test/util"

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
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.ParametersToRegistry("/animal", key, false)
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
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.ParametersToRegistry("/animal", key, false)
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
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.ParametersToRegistry("/animal", key, true)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, "/v2/service_instances/pinkiepie?accepts_incomplete=true", http.StatusBadRequest, req, api.ErrorParameterError)
}

// TestParametersDefault tests a parameter with a default work when not specified.
func TestParametersDefault(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.ParametersToRegistryWithDefault("/animal", key, defaultValue, false)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryWithValue(t, entry, registry.Key(key), defaultValue)
}

// TestParameterGenerateKeyRSAPKCS1 tests we can generate PKCS#1 formatted RSA keys.
func TestParameterGenerateKeyRSAPKCS1(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS1, &defaultKeyLength, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyRSAPKCS8 tests we can generate PKCS#8 formatted RSA keys.
func TestParameterGenerateKeyRSAPKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyRSAEC tests we can't generate EC formatted RSA keys.
func TestParameterGenerateKeyRSAECInvalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingSEC1, &defaultKeyLength, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyRSAMissingLengthInvalid tests we handle missing RSA key length gracefully.
func TestParameterGenerateKeyRSAMissingLengthInvalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP224PKCS1Invalid tests we can't generate PKCS#1 formatted EC keys.
func TestParameterGenerateKeyEllipticP224PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP224, v1.KeyEncodingPKCS1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP224PKCS8 tests we can generate PKCS#8 formatted EC keys.
func TestParameterGenerateKeyEllipticP224PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP224, v1.KeyEncodingPKCS8, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP224EC tests we can generate EC formatted EC keys.
func TestParameterGenerateKeyEllipticP224EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP224, v1.KeyEncodingSEC1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP256PKCS1Invalid tests we can't generate PKCS#1 formatted EC keys.
func TestParameterGenerateKeyEllipticP256PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP256, v1.KeyEncodingPKCS1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP256PKCS8 tests we can generate PKCS#8 formatted EC keys.
func TestParameterGenerateKeyEllipticP256PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP256, v1.KeyEncodingPKCS8, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP256EC tests we can generate EC formatted EC keys.
func TestParameterGenerateKeyEllipticP256EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP256, v1.KeyEncodingSEC1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP384PKCS1Invalid tests we can't generate PKCS#1 formatted EC keys.
func TestParameterGenerateKeyEllipticP384PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP384, v1.KeyEncodingPKCS1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP384PKCS8 tests we can generate PKCS#8 formatted EC keys.
func TestParameterGenerateKeyEllipticP384PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP384, v1.KeyEncodingPKCS8, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP384EC tests we can generate EC formatted EC keys.
func TestParameterGenerateKeyEllipticP384EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP384, v1.KeyEncodingSEC1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP512PKCS1Invalid tests we can't generate PKCS#1 formatted EC keys.
func TestParameterGenerateKeyEllipticP521PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP521, v1.KeyEncodingPKCS1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticP521PKCS8 tests we can generate PKCS#8 formatted EC keys.
func TestParameterGenerateKeyEllipticP521PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP521, v1.KeyEncodingPKCS8, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticP512EC tests we can generate EC formatted EC keys.
func TestParameterGenerateKeyEllipticP521EC(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP521, v1.KeyEncodingSEC1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticED25519PKCS1Invalid tests we can't generate PKCS#1 formatted ED keys.
func TestParameterGenerateKeyED25519PKCS1Invalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeED25519, v1.KeyEncodingPKCS1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateKeyEllipticED25519PKCS8 tests we can generate PKCS#8 formatted ED keys.
func TestParameterGenerateKeyED25519PKCS8(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeED25519, v1.KeyEncodingPKCS8, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)
}

// TestParameterGenerateKeyEllipticED25519EC tests we can't generate EC formatted ED keys.
func TestParameterGenerateKeyED25519ECInvalid(t *testing.T) {
	defer mustReset(t)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.KeyParameterToRegistry(v1.KeyTypeED25519, v1.KeyEncodingSEC1, nil, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateCACertificateRSAPKCS1 tests that we can create a CA certificate with an
// RSA private key.
func TestParameterGenerateCACertificateRSAPKCS1(t *testing.T) {
	defer mustReset(t)

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS1, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP224, v1.KeyEncodingSEC1, nil, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeED25519, v1.KeyEncodingPKCS8, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustPutAndError(t, util.ServiceInstanceURI(fixtures.ServiceInstanceName, util.CreateServiceInstanceQuery()), http.StatusBadRequest, req, api.ErrorConfigurationError)
}

// TestParameterGenerateServerCertificateRSAPKCS1 tests that we can create a server certificate with an
// RSA private key.
func TestParameterGenerateServerCertificateRSAPKCS1(t *testing.T) {
	defer mustReset(t)

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS1, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)
	parameters = append(parameters, fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS1, &defaultKeyLength, childKeyKey)...)
	parameters = append(parameters, fixtures.SignedCertificateParameterToRegistry(&childKeyKey, defaultCN, v1.Server, &caKeyKey, &caCertificateKey, childCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)
	parameters = append(parameters, fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, childKeyKey)...)
	parameters = append(parameters, fixtures.SignedCertificateParameterToRegistry(&childKeyKey, defaultCN, v1.Server, &caKeyKey, &caCertificateKey, childCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP224, v1.KeyEncodingSEC1, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)
	parameters = append(parameters, fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP224, v1.KeyEncodingSEC1, &defaultKeyLength, childKeyKey)...)
	parameters = append(parameters, fixtures.SignedCertificateParameterToRegistry(&childKeyKey, defaultCN, v1.Server, &caKeyKey, &caCertificateKey, childCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	sans := []string{
		"localhost",
		"bugs.looneytunes.com",
	}

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)
	parameters = append(parameters, fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, childKeyKey)...)
	parameters = append(parameters, fixtures.SignedCertificateParameterToRegistryWithDNSSANs(&childKeyKey, defaultCN, v1.Server, sans, &caKeyKey, &caCertificateKey, childCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS1, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)
	parameters = append(parameters, fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS1, &defaultKeyLength, childKeyKey)...)
	parameters = append(parameters, fixtures.SignedCertificateParameterToRegistry(&childKeyKey, defaultCN, v1.Client, &caKeyKey, &caCertificateKey, childCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)
	parameters = append(parameters, fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, childKeyKey)...)
	parameters = append(parameters, fixtures.SignedCertificateParameterToRegistry(&childKeyKey, defaultCN, v1.Client, &caKeyKey, &caCertificateKey, childCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP224, v1.KeyEncodingSEC1, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)
	parameters = append(parameters, fixtures.KeyParameterToRegistry(v1.KeyTypeEllipticP224, v1.KeyEncodingSEC1, &defaultKeyLength, childKeyKey)...)
	parameters = append(parameters, fixtures.SignedCertificateParameterToRegistry(&childKeyKey, defaultCN, v1.Client, &caKeyKey, &caCertificateKey, childCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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

	sans := []string{
		"bugs.bunny@looneytunes.com",
	}

	parameters := fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, caKeyKey)
	parameters = append(parameters, fixtures.CertificateParameterToRegistry(&caKeyKey, defaultCN, v1.CA, caCertificateKey)...)
	parameters = append(parameters, fixtures.KeyParameterToRegistry(v1.KeyTypeRSA, v1.KeyEncodingPKCS8, &defaultKeyLength, childKeyKey)...)
	parameters = append(parameters, fixtures.SignedCertificateParameterToRegistryWithEmailSANs(&childKeyKey, defaultCN, v1.Client, sans, &caKeyKey, &caCertificateKey, childCertificateKey)...)

	configuration := fixtures.BasicConfiguration()
	configuration.Bindings[0].ServiceInstance.Parameters = parameters
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
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.PasswordParameterToRegistry(defaultPasswordLength, nil, key)
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
	configuration.Bindings[0].ServiceInstance.Parameters = fixtures.PasswordParameterToRegistry(defaultPasswordLength, &customPasswordDictionary, key)
	util.MustReplaceBrokerConfig(t, clients, configuration)

	req := fixtures.BasicServiceInstanceCreateRequest()
	util.MustCreateServiceInstanceSuccessfully(t, fixtures.ServiceInstanceName, req)

	entry := util.MustGetRegistryEntry(t, clients, registry.ServiceInstance, fixtures.ServiceInstanceName)
	util.MustHaveRegistryEntryPassword(t, entry, key, defaultPasswordLength, customPasswordDictionary)
}
