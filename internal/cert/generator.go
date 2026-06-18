package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

type Generator struct {
	ca       *CA
	keySize  int
	validity time.Duration
}

func NewGenerator(ca *CA, keySize int, validity time.Duration) *Generator {
	return &Generator{
		ca:       ca,
		keySize:  keySize,
		validity: validity,
	}
}

func (g *Generator) GenerateCertificate(hostname string) (*tls.Certificate, []byte, []byte, time.Time, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, g.keySize)
	if err != nil {
		return nil, nil, nil, time.Time{}, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, nil, time.Time{}, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(g.validity)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"SPECTRE Proxy"},
			CommonName:   hostname,
		},
		DNSNames:    []string{hostname},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, g.ca.cert, &privateKey.PublicKey, g.ca.privateKey)
	if err != nil {
		return nil, nil, nil, time.Time{}, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, nil, time.Time{}, fmt.Errorf("failed to load key pair: %w", err)
	}

	return &tlsCert, certPEM, keyPEM, notAfter, nil
}
