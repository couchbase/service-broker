package util

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
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/errors"
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

// GenerateKey creates a PEM encoded private key.  The bits parameter is required for
// RSA keys.
func GenerateKey(keyType v1.KeyType, encoding v1.KeyEncodingType, bits *int) ([]byte, error) {
	var key crypto.PrivateKey

	var err error

	switch keyType {
	case v1.KeyTypeRSA:
		if bits == nil {
			return nil, errors.NewConfigurationError("RSA key length not specified")
		}

		key, err = rsa.GenerateKey(rand.Reader, *bits)
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
		return nil, errors.NewConfigurationError("invalid key type %s", keyType)
	}

	if err != nil {
		return nil, err
	}

	var t string

	var b []byte

	switch encoding {
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
	case v1.KeyEncodingSEC1:
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
		return nil, errors.NewConfigurationError("invalid encoding type %s", encoding)
	}

	block := &pem.Block{
		Type:  t,
		Bytes: b,
	}

	pemData := pem.EncodeToMemory(block)

	return pemData, nil
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

// DecodePrivateKey accepts a PEM formatted private key and parses it.
func DecodePrivateKey(keyPEM []byte) (crypto.PrivateKey, error) {
	block, rest := pem.Decode(keyPEM)
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

// DecodeCertificate accepts an parses a PEM formatted certificate.
func DecodeCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, rest := pem.Decode(certPEM)
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

// GenerateCertificate generates and signs an X.509 certificate.
func GenerateCertificate(keyPEM []byte, subject pkix.Name, lifetime time.Duration, usage v1.CertificateUsage, dnsSANs, emailSANs []string, caKeyPEM, caCertPEM []byte) ([]byte, error) {
	key, err := DecodePrivateKey(keyPEM)
	if err != nil {
		return nil, err
	}

	// Catch user misconfigurations.
	if _, ok := key.(ed25519.PrivateKey); ok {
		return nil, errors.NewConfigurationError("cannot use ed25519 keys for x.509 certificates")
	}

	req := &x509.CertificateRequest{
		Subject: subject,
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
	notAfter := notBefore.Add(lifetime)

	certificate := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               req.Subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		SubjectKeyId:          subjectKeyID,
		DNSNames:              dnsSANs,
		EmailAddresses:        emailSANs,
	}

	switch usage {
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
		return nil, errors.NewConfigurationError("unknown usage type %v", usage)
	}

	// Default to self signing.
	caKey := key
	caCert := certificate

	if caKeyPEM != nil {
		caKey, err = DecodePrivateKey(caKeyPEM)
		if err != nil {
			return nil, err
		}

		caCert, err = DecodeCertificate(caCertPEM)
		if err != nil {
			return nil, err
		}
	}

	cert, err := x509.CreateCertificate(rand.Reader, certificate, caCert, req.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	certPEMBlock := &pem.Block{
		Type:  pemTypeCertificate,
		Bytes: cert,
	}

	certPEM := pem.EncodeToMemory(certPEMBlock)

	return certPEM, nil
}
