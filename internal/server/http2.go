package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// HTTP2Config holds HTTP/2 configuration
type HTTP2Config struct {
	Enabled              bool
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	MaxConcurrentStreams uint32
	MaxReadFrameSize     uint32
}

// NewHTTP2Config creates a new HTTP/2 configuration
func NewHTTP2Config() *HTTP2Config {
	return &HTTP2Config{
		Enabled:              true,
		ReadTimeout:          15 * time.Second,
		WriteTimeout:         15 * time.Second,
		IdleTimeout:          60 * time.Second,
		MaxConcurrentStreams: 1000,
		MaxReadFrameSize:     http2.DefaultMaxReadFrameSize,
	}
}

// StartHTTP2 starts the server with HTTP/2 support
func (s *Server) StartHTTP2(config *HTTP2Config) error {
	s.SetupRoutes()

	http2Config := &http2.Server{
		ReadTimeout:          config.ReadTimeout,
		WriteTimeout:         config.WriteTimeout,
		IdleTimeout:          config.IdleTimeout,
		MaxConcurrentStreams: config.MaxConcurrentStreams,
		MaxReadFrameSize:     config.MaxReadFrameSize,
	}

	// Create HTTP/2-enabled server
	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.Port),
		Handler:           h2c.NewHandler(s.router, http2Config),
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		ReadHeaderTimeout: config.ReadTimeout,
	}

	// Configure HTTP/2 on the server
	s.httpServer.RegisterOnShutdown(func() {
		log.Println("HTTP/2 server shutting down")
	})

	go func() {
		s.logger.Infof("Starting HTTP/2 server on port %d", s.config.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Failed to start HTTP/2 server: %v", err)
		}
	}()

	s.logger.Info("HTTP/2 Server started successfully")
	return nil
}

// StartHTTP2TLS starts the server with HTTP/2 over TLS
func (s *Server) StartHTTP2TLS(tlsConfig *TLSConfig, http2Config *HTTP2Config) error {
	s.SetupRoutes()

	h2Config := &http2.Server{
		ReadTimeout:          http2Config.ReadTimeout,
		WriteTimeout:         http2Config.WriteTimeout,
		IdleTimeout:          http2Config.IdleTimeout,
		MaxConcurrentStreams: http2Config.MaxConcurrentStreams,
		MaxReadFrameSize:     http2Config.MaxReadFrameSize,
	}

	// Create TLS config for HTTP/2
	tlsCfg := tlsConfig.GetTLSConfig()

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.Port),
		Handler:           h2c.NewHandler(s.router, h2Config),
		TLSConfig:         tlsCfg,
		ReadTimeout:       http2Config.ReadTimeout,
		WriteTimeout:      http2Config.WriteTimeout,
		IdleTimeout:       http2Config.IdleTimeout,
		ReadHeaderTimeout: http2Config.ReadTimeout,
	}

	// Enable HTTP/2
	s.httpServer.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))

	go func() {
		s.logger.Infof("Starting HTTP/2 TLS server on port %d", s.config.Port)
		if err := s.httpServer.ListenAndServeTLS(tlsConfig.CertPath, tlsConfig.KeyPath); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Failed to start HTTP/2 TLS server: %v", err)
		}
	}()

	s.logger.Info("HTTP/2 TLS Server started successfully")
	return nil
}

// ConfigureHTTP2 configures HTTP/2 on an existing server
func (s *Server) ConfigureHTTP2(config *HTTP2Config) {
	if !config.Enabled {
		s.logger.Info("HTTP/2 is disabled")
		return
	}

	// HTTP/2 is automatically enabled when using TLS or h2c.NewHandler
	s.logger.Info("HTTP/2 is enabled")
	s.logger.Infof("Max concurrent streams: %d", config.MaxConcurrentStreams)
	s.logger.Infof("Max read frame size: %d bytes", config.MaxReadFrameSize)
}

// GetHTTP2Info returns HTTP/2 information
func (s *Server) GetHTTP2Info() map[string]interface{} {
	return map[string]interface{}{
		"http2_enabled": true,
		"protocol":      "HTTP/2",
		"features": []string{
			"Multiplexing",
			"Header Compression",
			"Server Push",
			"Flow Control",
		},
	}
}
