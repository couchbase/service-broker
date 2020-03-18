package provisioners

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" // nolint:gosec
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1alpha1"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/log"
	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/evanphx/json-patch"
	"github.com/go-openapi/jsonpointer"
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// pemTypeRSAPrivateKey is used with PKCS#1 RSA keys.
	pemTypeRSAPrivateKey = "RSA PRIVATE KEY"

	// pemTypePrivateKey is used with PKCS#8 keys.
	pemTypePrivateKey = "PRIVATE KEY"

	// pemTypeECPrivateKey is used with EC private keys.
	pemTypeECPrivateKey = "EC PRIVATE KEY"

	// pemTypeCertificate is used with all certificates.
	pemTypeCertificate = "CERTIFICATE"
)

// GetNamespace returns the namespace to provision resources in.  This is the namespace
// the broker lives in by default, however when operating as a kubernetes cluster service
// broker then this information is passed as request context.
func GetNamespace(context *runtime.RawExtension) (string, error) {
	if context != nil {
		var ctx interface{}

		if err := json.Unmarshal(context.Raw, &ctx); err != nil {
			glog.Infof("unmarshal of client context failed: %v", err)
			return "", err
		}

		pointer, err := jsonpointer.New("/namespace")
		if err != nil {
			glog.Infof("failed to parse JSON pointer: %v", err)
			return "", err
		}

		v, _, err := pointer.Get(ctx)
		if err == nil {
			namespace, ok := v.(string)
			if ok {
				return namespace, nil
			}

			glog.Infof("request context namespace not a string")

			return "", errors.NewParameterError("request context namespace not a string")
		}
	}

	return config.Namespace(), nil
}

// getServiceAndPlanNames translates from GUIDs to human readable names used in configuration.
func getServiceAndPlanNames(serviceID, planID string) (string, string, error) {
	for _, service := range config.Config().Spec.Catalog.Services {
		if service.ID == serviceID {
			for _, plan := range service.Plans {
				if plan.ID == planID {
					return service.Name, plan.Name, nil
				}
			}

			return "", "", fmt.Errorf("unable to locate plan for ID %s", planID)
		}
	}

	return "", "", fmt.Errorf("unable to locate service for ID %s", serviceID)
}

// getTemplateBindings returns the template bindings associated with a creation request's
// service and plan IDs.
func getTemplateBindings(serviceID, planID string) (*v1.CouchbaseServiceBrokerConfigBinding, error) {
	service, plan, err := getServiceAndPlanNames(serviceID, planID)
	if err != nil {
		return nil, err
	}

	for index, binding := range config.Config().Spec.Bindings {
		if binding.Service == service && binding.Plan == plan {
			return &config.Config().Spec.Bindings[index], nil
		}
	}

	return nil, fmt.Errorf("unable to locate template bindings for service plan %s/%s", service, plan)
}

// getTemplateBinding returns the binding associated with a specific resource type.
func getTemplateBinding(t ResourceType, serviceID, planID string) (*v1.CouchbaseServiceBrokerTemplateList, error) {
	bindings, err := getTemplateBindings(serviceID, planID)
	if err != nil {
		return nil, err
	}

	var templates *v1.CouchbaseServiceBrokerTemplateList

	switch t {
	case ResourceTypeServiceInstance:
		templates = bindings.ServiceInstance
	case ResourceTypeServiceBinding:
		templates = bindings.ServiceBinding
	default:
		return nil, fmt.Errorf("illegal binding type %s", string(t))
	}

	if templates == nil {
		return nil, errors.NewConfigurationError("missing bindings for type %s", string(t))
	}

	return templates, nil
}

// getTemplate returns the template corresponding to a template name.
func getTemplate(name string) (*v1.CouchbaseServiceBrokerConfigTemplate, error) {
	for index, template := range config.Config().Spec.Templates {
		if template.Name == name {
			return &config.Config().Spec.Templates[index], nil
		}
	}

	return nil, fmt.Errorf("unable to locate template for %s", name)
}

// resolveParameter attempts to find the parameter path in the provided JSON.
func resolveParameter(path string, entry *registry.Entry) (interface{}, bool, error) {
	var parameters interface{}

	ok, err := entry.Get(registry.Parameters, &parameters)
	if err != nil {
		return nil, false, err
	}

	if !ok {
		return nil, false, fmt.Errorf("unable to lookup parameters")
	}

	glog.Infof("interrogating path %s", path)

	pointer, err := jsonpointer.New(path)
	if err != nil {
		glog.Infof("failed to parse JSON pointer: %v", err)
		return nil, false, err
	}

	value, _, err := pointer.Get(parameters)
	if err != nil {
		return nil, false, nil
	}

	return value, true, nil
}

// resolveFormatParameter looks up a registry or parameter.
func resolveFormatParameter(parameter *v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter, entry *registry.Entry) (interface{}, error) {
	var value interface{}

	switch {
	case parameter.Registry != nil:
		v, ok, err := entry.GetUser(*parameter.Registry)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, nil
		}

		value = v

	case parameter.Parameter != nil:
		v, ok, err := resolveParameter(*parameter.Parameter, entry)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, nil
		}

		value = v
	default:
		return nil, fmt.Errorf("format parameter type not specified")
	}

	glog.Infof("resolved parameter value %v", value)

	return value, nil
}

// resolveFormatParameterStringList looks up a list of parameters trhrowing an error if
// any member is not a string.
func resolveFormatParameterStringList(parameters []v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter, entry *registry.Entry) ([]string, error) {
	list := []string{}

	for index := range parameters {
		value, err := resolveFormatParameter(&parameters[index], entry)
		if err != nil {
			return nil, err
		}

		if value == nil {
			continue
		}

		strValue, ok := value.(string)
		if !ok {
			return nil, errors.NewConfigurationError("string list parameter not a string %v", value)
		}

		list = append(list, strValue)
	}

	return list, nil
}

// resolveFormat attepts to render a format parameter.
func resolveFormat(format *v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormat, entry *registry.Entry) (interface{}, error) {
	parameters := make([]interface{}, len(format.Parameters))

	for index := range format.Parameters {
		value, err := resolveFormatParameter(&format.Parameters[index], entry)
		if err != nil {
			return nil, err
		}

		parameters[index] = value
	}

	return fmt.Sprintf(format.String, parameters...), nil
}

// resolveGeneratePassword generates a random string.
func resolveGeneratePassword(config *v1.CouchbaseServiceBrokerConfigTemplateParameterSourceGeneratePassword) (interface{}, error) {
	dictionary := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if config.Dictionary != nil {
		dictionary = *config.Dictionary
	}

	glog.Infof("generating string with length %d using dictionary %s", config.Length, dictionary)

	// Adjust so the length is within array bounds.
	arrayIndexOffset := 1
	dictionaryLength := len(dictionary) - arrayIndexOffset

	limit := big.NewInt(int64(dictionaryLength))
	value := ""

	for i := 0; i < config.Length; i++ {
		indexBig, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return nil, err
		}

		if !indexBig.IsInt64() {
			return nil, errors.NewConfigurationError("random index overflow")
		}

		index := int(indexBig.Int64())

		value += dictionary[index : index+1]
	}

	return value, nil
}

// resolveGenerateKey will create a PEM encoded private key.
func resolveGenerateKey(config *v1.CouchbaseServiceBrokerConfigTemplateParameterSourceGenerateKey) (interface{}, error) {
	var key crypto.PrivateKey

	var err error

	switch config.Type {
	case v1.KeyTypeRSA:
		if config.Bits == nil {
			return nil, errors.NewConfigurationError("RSA key length not specified")
		}

		key, err = rsa.GenerateKey(rand.Reader, *config.Bits)
	case v1.KeyTypeEllipticP224:
		key, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case v1.KeyTypeEllipticP256:
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case v1.KeyTypeEllipticP384:
		key, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case v1.KeyTypeEllipticP521:
		key, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	case v1.KeyTypeED25519:
		_, key, err = ed25519.GenerateKey(rand.Reader)
	default:
		return nil, errors.NewConfigurationError("invalid key type %s", config.Type)
	}

	if err != nil {
		return nil, err
	}

	var t string

	var b []byte

	switch config.Encoding {
	case v1.KeyEncodingPKCS1:
		t = pemTypeRSAPrivateKey

		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.NewConfigurationError("invalid key for PKCS#1 encoding")
		}

		b = x509.MarshalPKCS1PrivateKey(rsaKey)
	case v1.KeyEncodingPKCS8:
		t = pemTypePrivateKey

		b, err = x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return nil, err
		}
	case v1.KeyEncodingEC:
		t = pemTypeECPrivateKey

		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.NewConfigurationError("invalid key for EC encoding")
		}

		b, err = x509.MarshalECPrivateKey(ecKey)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.NewConfigurationError("invalid encoding type %s", config.Type)
	}

	block := &pem.Block{
		Type:  t,
		Bytes: b,
	}

	pemData := pem.EncodeToMemory(block)

	return string(pemData), nil
}

// generateSerial creates a unique certificate serial number as defined
// in RFC 3280.  It is upto 20 octets in length and non-negative
func generateSerial() (*big.Int, error) {
	one := 1
	shift := 128
	serialLimit := new(big.Int).Lsh(big.NewInt(int64(one)), uint(shift))

	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, err
	}

	return new(big.Int).Abs(serialNumber), nil
}

// generateSubjectKeyIdentifier creates a hash of the public key as defined in
// RFC3280 used to create certificate paths from a leaf to a CA
func generateSubjectKeyIdentifier(pub interface{}) ([]byte, error) {
	var subjectPublicKey []byte

	var err error

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		subjectPublicKey, err = asn1.Marshal(*pub)
	case *ecdsa.PublicKey:
		subjectPublicKey = elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	default:
		return nil, fmt.Errorf("invalid public key type")
	}

	if err != nil {
		return nil, err
	}

	sum := sha1.Sum(subjectPublicKey) // nolint:gosec

	return sum[:], nil
}

// decodePrivateKey reads a PEM formatted private key and parses it.
func decodePrivateKey(parameter *v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter, entry *registry.Entry) (crypto.PrivateKey, error) {
	keyPEM, err := resolveFormatParameter(parameter, entry)
	if err != nil {
		return nil, err
	}

	data, ok := keyPEM.(string)
	if !ok {
		return nil, errors.NewConfigurationError("private key is not a string")
	}

	block, rest := pem.Decode([]byte(data))
	if block == nil {
		return nil, errors.NewConfigurationError("unable to decode certificate key PEM file")
	}

	emptyArray := 0
	if rest != nil && len(rest) > emptyArray {
		return nil, errors.NewConfigurationError("unexpected content in PEM file")
	}

	var key crypto.PrivateKey

	switch block.Type {
	case pemTypeRSAPrivateKey:
		v, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		key = v
	case pemTypePrivateKey:
		v, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		key = v
	case pemTypeECPrivateKey:
		v, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		key = v
	default:
		return nil, errors.NewConfigurationError("private key format %s unsupported", block.Type)
	}

	return key, nil
}

// decodeCertificate resolves and parses a PEM formatted certificate.
func decodeCertificate(parameter *v1.CouchbaseServiceBrokerConfigTemplateParameterSourceFormatParameter, entry *registry.Entry) (*x509.Certificate, error) {
	certPEM, err := resolveFormatParameter(parameter, entry)
	if err != nil {
		return nil, err
	}

	data, ok := certPEM.(string)
	if !ok {
		return nil, errors.NewConfigurationError("private key is not a string")
	}

	block, rest := pem.Decode([]byte(data))
	if block == nil {
		return nil, errors.NewConfigurationError("unable to decode certificate key PEM file")
	}

	emptyArray := 0
	if rest != nil && len(rest) > emptyArray {
		return nil, errors.NewConfigurationError("unexpected content in PEM file")
	}

	if block.Type != pemTypeCertificate {
		return nil, errors.NewConfigurationError("certificate format %s unsupported", block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, err
}

// resolveGenerateCertificate will create a PEM encoded X.509 certificate.
func resolveGenerateCertificate(config *v1.CouchbaseServiceBrokerConfigTemplateParameterSourceGenerateCertificate, entry *registry.Entry) (interface{}, error) {
	key, err := decodePrivateKey(&config.Key, entry)
	if err != nil {
		return nil, err
	}

	req := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: config.Name.CommonName,
		},
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, req, key)
	if err != nil {
		return nil, err
	}

	req, err = x509.ParseCertificateRequest(csr)
	if err != nil {
		return nil, err
	}

	serialNumber, err := generateSerial()
	if err != nil {
		return nil, err
	}

	subjectKeyID, err := generateSubjectKeyIdentifier(req.PublicKey)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(config.Lifetime.Duration)

	certificate := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               req.Subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		SubjectKeyId:          subjectKeyID,
	}

	switch config.Usage {
	case v1.CA:
		certificate.IsCA = true
		certificate.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	case v1.Server:
		certificate.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
		certificate.ExtKeyUsage = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		}
	case v1.Client:
		certificate.KeyUsage = x509.KeyUsageDigitalSignature
		certificate.ExtKeyUsage = []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		}
	default:
		return nil, errors.NewConfigurationError("unknown usage type %v", config.Usage)
	}

	if config.AlternativeNames != nil {
		list, err := resolveFormatParameterStringList(config.AlternativeNames.DNS, entry)
		if err != nil {
			return nil, err
		}

		certificate.DNSNames = list

		list, err = resolveFormatParameterStringList(config.AlternativeNames.Email, entry)
		if err != nil {
			return nil, err
		}

		certificate.EmailAddresses = list
	}

	// Default to self signing.
	caCertificate := certificate
	caKey := key

	if config.CA != nil {
		caKey, err = decodePrivateKey(&config.CA.Key, entry)
		if err != nil {
			return nil, err
		}

		caCertificate, err = decodeCertificate(&config.CA.Certificate, entry)
		if err != nil {
			return nil, err
		}
	}

	cert, err := x509.CreateCertificate(rand.Reader, certificate, caCertificate, req.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	certPEMBlock := &pem.Block{
		Type:  pemTypeCertificate,
		Bytes: cert,
	}

	certPEM := pem.EncodeToMemory(certPEMBlock)

	return string(certPEM), nil
}

// resolveSource gets a parameter source from either metadata or a JSON path into user specified parameters.
func resolveSource(source *v1.CouchbaseServiceBrokerConfigTemplateParameterSource, entry *registry.Entry) (interface{}, error) {
	if source == nil {
		return nil, nil
	}

	// Try to resolve the parameter.
	var value interface{}

	switch {
	// Registry parameters are reference registry values.
	case source.Registry != nil:
		glog.Infof("getting registry key %s", *source.Registry)

		v, ok, err := entry.GetUser(*source.Registry)
		if err != nil {
			return nil, err
		}

		if !ok {
			glog.Infof("registry key %s not set, skipping", *source.Registry)
			break
		}

		value = v

	// Parameters reference parameters specified by the client in response to
	// the schema advertised to the client.
	case source.Parameter != nil:
		v, ok, err := resolveParameter(*source.Parameter, entry)
		if err != nil {
			return nil, err
		}

		if !ok {
			glog.Infof("client parameter %s not set, skipping", *source.Parameter)
			break
		}

		value = v

	// Format will do a string formatting with Sprintf and a variadic set
	// of parameters.
	case source.Format != nil:
		v, err := resolveFormat(source.Format, entry)
		if err != nil {
			return nil, err
		}

		value = v

	// GeneratePassword will randomly generate a password string for example.
	case source.GeneratePassword != nil:
		v, err := resolveGeneratePassword(source.GeneratePassword)
		if err != nil {
			return nil, err
		}

		value = v

	// GenerateKey will create a PEM encoded private key.
	case source.GenerateKey != nil:
		v, err := resolveGenerateKey(source.GenerateKey)
		if err != nil {
			return nil, err
		}

		value = v

	// GenerateCertificate will create a PEM encoded X.509 certificate.
	case source.GenerateCertificate != nil:
		v, err := resolveGenerateCertificate(source.GenerateCertificate, entry)
		if err != nil {
			return nil, err
		}

		value = v

	// Template will recursively render a template and return an object.
	// This allows sharing of common configuration.
	case source.Template != nil:
		t, err := getTemplate(*source.Template)
		if err != nil {
			return nil, err
		}

		template, err := renderTemplate(t, entry)
		if err != nil {
			return nil, err
		}

		var v interface{}
		if err := json.Unmarshal(template.Template.Raw, &v); err != nil {
			glog.Infof("unmarshal of template failed: %v", err)
			return nil, err
		}

		value = v
	}

	glog.Infof("resolved source value %v", value)

	return value, nil
}

// resolveTemplateParameter applies parameter lookup rules and tries to return a value.
func resolveTemplateParameter(parameter *v1.CouchbaseServiceBrokerConfigTemplateParameter, entry *registry.Entry, useDefaults bool) (interface{}, error) {
	value, err := resolveSource(parameter.Source, entry)
	if err != nil {
		return nil, err
	}

	// If no value has been found or generated then use a default if set.
	if value == nil && useDefaults && parameter.Default != nil {
		switch {
		case parameter.Default.String != nil:
			value = *parameter.Default.String
		case parameter.Default.Bool != nil:
			value = *parameter.Default.Bool
		case parameter.Default.Int != nil:
			value = *parameter.Default.Int
		case parameter.Default.Object != nil:
			var v interface{}

			if err := json.Unmarshal(parameter.Default.Object.Raw, &v); err != nil {
				glog.Infof("unmarshal of source default failed: %v", err)
				return nil, err
			}

			value = v
		default:
			return nil, errors.NewConfigurationError("undefined source default parameter")
		}

		glog.Infof("using source default %v", value)
	}

	if parameter.Required && value == nil {
		glog.Infof("source unset but is required")
		return nil, errors.NewParameterError("source for parameter %s is required", parameter.Name)
	}

	return value, nil
}

// patchObject takes a raw JSON object and applies parameters to it.
func patchObject(object []byte, parameters []v1.CouchbaseServiceBrokerConfigTemplateParameter, entry *registry.Entry, useDefaults bool) ([]byte, error) {
	// Now for the fun bit.  Work through each defined parameter and apply it to
	// the object.  This basically works like JSON patch++, automatically filling
	// in parent objects and arrays as necessary.
	for index, parameter := range parameters {
		value, err := resolveTemplateParameter(&parameters[index], entry, useDefaults)
		if err != nil {
			return nil, err
		}

		// Set each destination path using JSON patch.
		patches := []string{}

		for _, destination := range parameter.Destinations {
			switch {
			case destination.Registry != nil:
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.NewConfigurationError("parameter %s is not a string", parameter.Name)
				}

				if err := entry.SetUser(*destination.Registry, strValue); err != nil {
					return nil, errors.NewConfigurationError(err.Error())
				}

			case destination.Path != nil:
				valueJSON, err := json.Marshal(value)
				if err != nil {
					glog.Infof("marshal of value failed: %v", err)
					return nil, err
				}

				patches = append(patches, fmt.Sprintf(`{"op":"add","path":"%s","value":%s}`, *destination.Path, string(valueJSON)))
			}
		}

		minPatches := 1
		if len(patches) < minPatches {
			glog.Infof("no paths to apply parameter to")
			continue
		}

		patchSet := "[" + strings.Join(patches, ",") + "]"

		glog.Infof("applying patchset %s", patchSet)

		patch, err := jsonpatch.DecodePatch([]byte(patchSet))
		if err != nil {
			glog.Infof("decode of JSON patch failed: %v", err)
			return nil, err
		}

		if object, err = patch.Apply(object); err != nil {
			glog.Infof("apply of JSON patch failed: %v", err)
			return nil, err
		}
	}

	return object, nil
}

// renderTemplate accepts a template defined in the configuration and applies any
// request or metadata parameters to it.
func renderTemplate(template *v1.CouchbaseServiceBrokerConfigTemplate, entry *registry.Entry) (*v1.CouchbaseServiceBrokerConfigTemplate, error) {
	glog.Infof("rendering template %s", template.Name)

	if template.Template == nil || template.Template.Raw == nil {
		return nil, errors.NewConfigurationError("template %s is not defined", template.Name)
	}

	glog.V(log.LevelDebug).Infof("template source: %s", string(template.Template.Raw))

	// We will be modifying the template in place, so first clone it as the
	// config is immutable.
	t := template.DeepCopy()

	object, err := patchObject(t.Template.Raw, t.Parameters, entry, true)
	if err != nil {
		return nil, err
	}

	t.Template.Raw = object

	glog.Infof("rendered template %s", string(t.Template.Raw))

	return t, nil
}
