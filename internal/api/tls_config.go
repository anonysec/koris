package api

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

// TLSCertificates holds loaded server certificate material.
type TLSCertificates struct {
	Cert tls.Certificate
	CA   *x509.CertPool
}

// LoadTLSCertificates loads server cert+key and optional CA from filesystem paths.
func LoadTLSCertificates(certPath, keyPath, caPath string) (*TLSCertificates, error) {
	if certPath == "" || keyPath == "" {
		return nil, nil
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if caPath != "" {
		ca, err := os.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		if !pool.AppendCertsFromPEM(ca) {
			return nil, ErrInvalidCA
		}
	}
	return &TLSCertificates{Cert: cert, CA: pool}, nil
}

// ToTLSConfig creates a *tls.Config suitable for an http.Server.
func (c *TLSCertificates) ToTLSConfig() *tls.Config {
	if c == nil {
		return &tls.Config{InsecureSkipVerify: true}
	}
	tlsCfg := &tls.Config{
		Certificates:       []tls.Certificate{c.Cert},
		ClientCAs:          c.CA,
		ClientAuth:         tls.NoClientCert,
		MinVersion:         tls.VersionTLS12,
		CurvePreferences:   []tls.CurveID{tls.CurveP256},
		PreferServerCipherSuites: true,
	}
	if c.CA != nil {
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return tlsCfg
}

var (
	ErrInvalidCA = errors.New("invalid CA certificate")
)
