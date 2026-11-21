package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	"tomorrow-drop/internal/apperr"
)

const certCommonName = "tomorrow-drop"

// NewCertificate creates a self-signed X.509 certificate with the identity's
// Ed25519 key. It is used as the credential in TLS 1.3.
func NewCertificate(id *Identity, notBefore, notAfter time.Time) (tls.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate serial number: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: certCommonName},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, id.priv.Public(), id.priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: id.priv}, nil
}

// VerifyCertificate validates a DER self-signed Ed25519 certificate and returns
// its public key. It checks the format, the algorithm, the validity period and
// the signature by the certificate's own key.
func VerifyCertificate(der []byte, now time.Time) (ed25519.PublicKey, error) {
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperr.ErrMalformed, err)
	}
	if cert.PublicKeyAlgorithm != x509.Ed25519 || cert.SignatureAlgorithm != x509.PureEd25519 {
		return nil, fmt.Errorf("unsupported certificate algorithm: %w", apperr.ErrMalformed)
	}
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, fmt.Errorf("certificate outside its validity period: %w", apperr.ErrUnauthorized)
	}
	if err := cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
		return nil, fmt.Errorf("invalid certificate signature: %w", apperr.ErrUnauthorized)
	}
	key, ok := cert.PublicKey.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not Ed25519: %w", apperr.ErrMalformed)
	}
	return key, nil
}
