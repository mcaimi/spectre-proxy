package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

type CA struct {
	cert       *x509.Certificate
	privateKey *rsa.PrivateKey
	certPEM    []byte
	keyPEM     []byte
}

type CAFields struct {
	CommonName         string
	Organization       string
	OrganizationalUnit string
	Country            string
	Province           string
	Locality           string
	ValidityYears      int
}

func NewCA(keySize int) (*CA, error) {
	return NewCAWithFields(keySize, CAFields{
		CommonName:    "SPECTRE Root CA",
		Organization:  "SPECTRE Proxy",
		ValidityYears: 10,
	})
}

func NewCAWithFields(keySize int, fields CAFields) (*CA, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	validityYears := fields.ValidityYears
	if validityYears <= 0 {
		validityYears = 10
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(validityYears) * 365 * 24 * time.Hour)

	subject := pkix.Name{
		CommonName: fields.CommonName,
	}
	if fields.Organization != "" {
		subject.Organization = []string{fields.Organization}
	}
	if fields.OrganizationalUnit != "" {
		subject.OrganizationalUnit = []string{fields.OrganizationalUnit}
	}
	if fields.Country != "" {
		subject.Country = []string{fields.Country}
	}
	if fields.Province != "" {
		subject.Province = []string{fields.Province}
	}
	if fields.Locality != "" {
		subject.Locality = []string{fields.Locality}
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return &CA{
		cert:       cert,
		privateKey: privateKey,
		certPEM:    certPEM,
		keyPEM:     keyPEM,
	}, nil
}

func LoadCA(certPEM, keyPEM []byte) (*CA, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode CA private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	return &CA{
		cert:       cert,
		privateKey: privateKey,
		certPEM:    certPEM,
		keyPEM:     keyPEM,
	}, nil
}

func (ca *CA) GetCertPEM() []byte {
	return ca.certPEM
}

func (ca *CA) GetKeyPEM() []byte {
	return ca.keyPEM
}

func (ca *CA) GetExpiresAt() time.Time {
	return ca.cert.NotAfter
}
