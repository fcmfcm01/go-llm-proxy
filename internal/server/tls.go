package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
)

// TLSConfig holds TLS configuration
type TLSConfig struct {
	CertPath                 string
	KeyPath                  string
	MinVersion               string
	MaxVersion               string
	PreferServerCipherSuites bool
	CurvePreferences         []tls.CurveID
}

// NewTLSConfig creates a new TLS configuration
func NewTLSConfig(certPath, keyPath, minVersion, maxVersion string) *TLSConfig {
	return &TLSConfig{
		CertPath:                 certPath,
		KeyPath:                  keyPath,
		MinVersion:               minVersion,
		MaxVersion:               maxVersion,
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519, // Curve25519
			tls.CurveP256,   // P-256
			tls.CurveP384,   // P-384
		},
	}
}

// GetTLSConfig returns the TLS configuration for the server
func (c *TLSConfig) GetTLSConfig() *tls.Config {
	// Parse version
	minVersion, ok := parseTLSVersion(c.MinVersion)
	if !ok {
		minVersion = tls.VersionTLS13
	}

	maxVersion, ok := parseTLSVersion(c.MaxVersion)
	if !ok {
		maxVersion = tls.VersionTLS13
	}

	config := &tls.Config{
		MinVersion:               minVersion,
		MaxVersion:               maxVersion,
		PreferServerCipherSuites: c.PreferServerCipherSuites,
		CurvePreferences:         c.CurvePreferences,
		// Cipher suites for TLS 1.2
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		// Client certificate authentication can be added here
		// ClientAuth: tls.RequireAndVerifyClientCert,
	}

	return config
}

// StartTLS starts the server with TLS
func (s *Server) StartTLS(config *TLSConfig) error {
	// Validate certificates
	if err := s.validateCertificates(config); err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	s.SetupRoutes()

	tlsConfig := config.GetTLSConfig()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
		TLSConfig:    tlsConfig,
	}

	go func() {
		s.logger.Infof("Starting HTTPS server on port %d with TLS", s.config.Port)
		s.logger.Infof("Using TLS version: %s", config.MinVersion)
		if err := s.httpServer.ListenAndServeTLS(config.CertPath, config.KeyPath); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Failed to start TLS server: %v", err)
		}
	}()

	s.logger.Info("TLS Server started successfully")
	return nil
}

// validateCertificates validates that the certificate files exist and are valid
func (s *Server) validateCertificates(config *TLSConfig) error {
	// Check certificate file
	if _, err := os.Stat(config.CertPath); os.IsNotExist(err) {
		return fmt.Errorf("certificate file not found: %s", config.CertPath)
	}

	// Check key file
	if _, err := os.Stat(config.KeyPath); os.IsNotExist(err) {
		return fmt.Errorf("private key file not found: %s", config.KeyPath)
	}

	// Load certificate to validate
	_, err := tls.LoadX509KeyPair(config.CertPath, config.KeyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate or key: %w", err)
	}

	s.logger.Info("Certificate validation successful")
	return nil
}

// parseTLSVersion parses TLS version string to tls.Version constant
func parseTLSVersion(version string) (uint16, bool) {
	switch version {
	case "1.2":
		return tls.VersionTLS12, true
	case "1.3":
		return tls.VersionTLS13, true
	default:
		return 0, false
	}
}

// CreateSelfSignedCert creates a self-signed certificate for development
func CreateSelfSignedCert(certPath, keyPath string) error {
	// This is a placeholder for self-signed certificate generation
	// In production, you would use Let's Encrypt or other CA
	// For development, you can use tools like mkcert

	return fmt.Errorf("self-signed certificate generation not implemented. Please generate certificates using: mkcert -install")
}
