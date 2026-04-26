// Package ssl handles SSL/TLS certificate management.
// See AI.md PART 15 for SSL/TLS and Let's Encrypt specifications.
package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// SSLConfig holds SSL/TLS configuration
type SSLConfig struct {
	// Enable SSL/TLS
	Enabled bool
	// Certificate file path
	CertFile string
	// Private key file path
	KeyFile string
	// Let's Encrypt enabled
	LetsEncrypt bool
	// Domain for Let's Encrypt
	Domain string
	// SSL directory
	SSLDir string
}

// LoadOrGenerateTLS loads existing certificates or generates self-signed ones
func LoadOrGenerateTLS(cfg SSLConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// Try loading existing certificates
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err == nil {
			return &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}, nil
		}
	}

	// Check Let's Encrypt directory
	if cfg.LetsEncrypt && cfg.Domain != "" {
		certPath := filepath.Join(cfg.SSLDir, "letsencrypt", cfg.Domain, "fullchain.pem")
		keyPath := filepath.Join(cfg.SSLDir, "letsencrypt", cfg.Domain, "privkey.pem")

		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err == nil {
			return &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}, nil
		}
	}

	// Check local cert directory
	localCertPath := filepath.Join(cfg.SSLDir, "local", "cert.pem")
	localKeyPath := filepath.Join(cfg.SSLDir, "local", "key.pem")

	cert, err := tls.LoadX509KeyPair(localCertPath, localKeyPath)
	if err == nil {
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}, nil
	}

	// Generate self-signed certificate
	if err := generateSelfSignedCert(localCertPath, localKeyPath, cfg.Domain); err != nil {
		return nil, fmt.Errorf("generating self-signed cert: %w", err)
	}

	cert, err = tls.LoadX509KeyPair(localCertPath, localKeyPath)
	if err != nil {
		return nil, fmt.Errorf("loading generated cert: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// generateSelfSignedCert creates a self-signed certificate
func generateSelfSignedCert(certPath, keyPath, domain string) error {
	// Create directories
	if err := os.MkdirAll(filepath.Dir(certPath), 0700); err != nil {
		return err
	}

	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating private key: %w", err)
	}

	// Create certificate template
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ytdl"},
			CommonName:   domain,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if domain == "" || domain == "localhost" {
		template.DNSNames = []string{"localhost"}
	} else {
		template.DNSNames = []string{domain, "localhost"}
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}

	// Write certificate
	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certFile.Close()
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Write private key
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return nil
}

// RedirectHTTPToHTTPS returns an HTTP handler that redirects all requests to HTTPS
func RedirectHTTPToHTTPS(httpsPort int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := fmt.Sprintf("https://%s:%d%s", r.Host, httpsPort, r.RequestURI)
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}
