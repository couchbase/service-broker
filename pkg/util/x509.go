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
	"strings"
	"time"

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

// KeyType is a private key type.
type KeyType string

const (
	// RSA is widely supported, but the key sizes are large.
	KeyTypeRSA KeyType = "RSA"

	// KeyTypeEllipticP224 generates small keys relative to encryption strength.
	KeyTypeEllipticP224 KeyType = "EllipticP224"

	// KeyTypeEllipticP256 generates small keys relative to encryption strength.
	KeyTypeEllipticP256 KeyType = "EllipticP256"

	// KeyTypeEllipticP384 generates small keys relative to encryption strength.
	KeyTypeEllipticP384 KeyType = "EllipticP384"

	// KeyTypeEllipticP521 generates small keys relative to encryption strength.
	KeyTypeEllipticP521 KeyType = "EllipticP521"

	// KeyTypeED25519 generates small keys relative to encrption strength.
	KeyTypeED25519 KeyType = "ED25519"
)

// KeyEncodingType is a private key encoding type.
type KeyEncodingType string

const (
	// KeyEncodingPKCS1 may only be used with the RSA key type.
	KeyEncodingPKCS1 KeyEncodingType = "PKCS#1"

	// KeyEncodingPKCS8 may be used for any key type.
	KeyEncodingPKCS8 KeyEncodingType = "PKCS#8"

	// KeyEncodingSEC1 may only be used with EC key types.
	KeyEncodingSEC1 KeyEncodingType = "SEC 1"
)

// CertificateUsage defines the certificate use.
type CertificateUsage string

const (
	// CA is used for signing certificates and providing a trust anchor.
	CA CertificateUsage = "CA"

	// Server is used for server certificates.
	Server CertificateUsage = "Server"

	// Client is used for client certificates.
	Client CertificateUsage = "Client"
)

// GenerateKey creates a PEM encoded private key.  The bits parameter is required for
// RSA keys.
func GenerateKey(keyType KeyType, encoding KeyEncodingType, bits *int) ([]byte, error) {
	var key crypto.PrivateKey

	var err error

	switch keyType {
	case KeyTypeRSA:
		if bits == nil {
			return nil, errors.NewConfigurationError("RSA key length not specified")
		}

		key, err = rsa.GenerateKey(rand.Reader, *bits)
	case KeyTypeEllipticP224:
		key, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case KeyTypeEllipticP256:
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case KeyTypeEllipticP384:
		key, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case KeyTypeEllipticP521:
		key, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	case KeyTypeED25519:
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
	case KeyEncodingPKCS1:
		t = pemTypeRSAPrivateKey

		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.NewConfigurationError("invalid key for PKCS#1 encoding")
		}

		b = x509.MarshalPKCS1PrivateKey(rsaKey)
	case KeyEncodingPKCS8:
		t = pemTypePrivateKey

		b, err = x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return nil, err
		}
	case KeyEncodingSEC1:
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
func GenerateCertificate(keyPEM []byte, cn string, lifetime time.Duration, usage CertificateUsage, sans []string, caKeyPEM, caCertPEM []byte) ([]byte, error) {
	key, err := DecodePrivateKey(keyPEM)
	if err != nil {
		return nil, err
	}

	// Catch user misconfigurations.
	if _, ok := key.(ed25519.PrivateKey); ok {
		return nil, errors.NewConfigurationError("cannot use ed25519 keys for x.509 certificates")
	}

	req := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: cn,
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
	notAfter := notBefore.Add(lifetime)

	certificate := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               req.Subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		SubjectKeyId:          subjectKeyID,
	}

	for _, san := range sans {
		// A type and a name.
		requiredFields := 2

		fields := strings.Split(san, ":")
		if len(fields) != requiredFields {
			return nil, fmt.Errorf("malformed SAN %s", san)
		}

		switch fields[0] {
		case "DNS":
			certificate.DNSNames = append(certificate.DNSNames, fields[1])
		case "EMAIL":
			certificate.EmailAddresses = append(certificate.EmailAddresses, fields[1])
		default:
			return nil, fmt.Errorf("illegal SAN type %s", fields[0])
		}
	}

	switch usage {
	case CA:
		certificate.IsCA = true
		certificate.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	case Server:
		certificate.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
		certificate.ExtKeyUsage = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		}
	case Client:
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
