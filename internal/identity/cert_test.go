package identity

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"testing"
	"time"

	"tomorrow-drop/internal/apperr"
)

func TestCertificateBindsIdentityKey(t *testing.T) {
	id, _ := Generate()
	now := time.Now()
	cert, err := NewCertificate(id, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("NewCertificate: %v", err)
	}
	key, err := VerifyCertificate(cert.Certificate[0], now)
	if err != nil {
		t.Fatalf("VerifyCertificate: %v", err)
	}
	if !bytes.Equal(key, id.PublicKey()) {
		t.Fatalf("certificate key differs from the identity")
	}
}

func TestVerifyRejectsExpiredCertificate(t *testing.T) {
	id, _ := Generate()
	now := time.Now()
	cert, _ := NewCertificate(id, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	_, err := VerifyCertificate(cert.Certificate[0], now)
	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("expired certificate = %v, want ErrUnauthorized", err)
	}
}

func TestVerifyRejectsCertificateNotYetValid(t *testing.T) {
	id, _ := Generate()
	now := time.Now()
	cert, _ := NewCertificate(id, now.Add(24*time.Hour), now.Add(48*time.Hour))
	_, err := VerifyCertificate(cert.Certificate[0], now)
	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("future certificate = %v, want ErrUnauthorized", err)
	}
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	id, _ := Generate()
	now := time.Now()
	cert, _ := NewCertificate(id, now.Add(-time.Hour), now.Add(time.Hour))
	tampered := append([]byte(nil), cert.Certificate[0]...)
	tampered[len(tampered)-1] ^= 0x01

	_, err := VerifyCertificate(tampered, now)
	if err == nil {
		t.Fatalf("certificate with a tampered signature was accepted")
	}
}

func TestVerifyRejectsInvalidDER(t *testing.T) {
	_, err := VerifyCertificate([]byte("not der"), time.Now())
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("invalid DER = %v, want ErrMalformed", err)
	}
}

func TestVerifyRejectsNonEd25519Algorithm(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "ecdsa"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &ecKey.PublicKey, ecKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCertificate(der, now); err == nil {
		t.Fatalf("ECDSA certificate was accepted as an Ed25519 identity")
	}
}
