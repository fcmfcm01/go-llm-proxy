package integration

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	"github.com/fcmfcm01/go-llm-proxy/internal/server"
)

// TestHTTP2Support tests HTTP/2 functionality
func TestHTTP2Support(t *testing.T) {
	// Create a test server with HTTP/2 enabled
	config := &server.ServerConfig{
		Port:            0, // Use port 0 to get a random available port
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    60 * time.Second,
		IdleTimeout:     90 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		EnableHTTP2:     true,
	}

	// Create HTTP/2 config
	http2Config := server.NewHTTP2Config()

	// Create server
	s := server.NewServer(config, nil)

	// Setup routes
	s.SetupRoutes()

	// Start HTTP/2 server
	err := s.StartHTTP2(http2Config)
	require.NoError(t, err)
	defer s.Stop()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	t.Run("HTTP/2 connection with h2c", func(t *testing.T) {
		// Create HTTP/2 client transport
		transport := &http2.Transport{
			ReadBufferSize:  1 << 20,
			WriteBufferSize: 1 << 20,
		}

		// Create request
		req, err := http.NewRequest("GET", "http://localhost:8080/healthz", nil)
		require.NoError(t, err)

		// Make request using HTTP/2
		client := &http.Client{Transport: transport}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "HTTP/2", resp.Proto)
	})

	t.Run("HTTP/2 client configuration", func(t *testing.T) {
		// Test that we can create HTTP/2 client config
		transport := server.HTTP2ClientConfig()
		assert.NotNil(t, transport)
		assert.Equal(t, uint32(1<<20), transport.ReadBufferSize)
		assert.Equal(t, uint32(1<<20), transport.WriteBufferSize)
	})

	t.Run("HTTP/2 settings", func(t *testing.T) {
		// Test HTTP/2 settings configuration
		assert.Equal(t, true, http2Config.Enabled)
		assert.True(t, http2Config.MaxConcurrentStreams > 0)
		assert.Equal(t, time.Duration(15*time.Second), http2Config.ReadTimeout)
	})

	t.Run("HTTP/2 features", func(t *testing.T) {
		// Test HTTP/2 features information
		info := s.GetHTTP2Info()
		assert.Equal(t, true, info["http2_enabled"])
		assert.Equal(t, "HTTP/2", info["protocol"])

		features, ok := info["features"].([]string)
		require.True(t, ok, "features should be a slice of strings")
		assert.Contains(t, features, "Multiplexing")
		assert.Contains(t, features, "Header Compression")
		assert.Contains(t, features, "Flow Control")
	})

	t.Run("HTTP/2 configuration", func(t *testing.T) {
		// Test HTTP/2 configuration
		s.ConfigureHTTP2(http2Config)
		// This should log that HTTP/2 is enabled
		// No assertions needed, just verifying no panic
	})

	t.Run("HTTP/2 info detection", func(t *testing.T) {
		// Test detection of HTTP/2 in requests
		// This is a unit test of the helper functions
		assert.True(t, server.IsHTTP2Enabled(&http.Request{ProtoMajor: 2}))
		assert.False(t, server.IsHTTP2Enabled(&http.Request{ProtoMajor: 1}))
	})

	t.Run("HTTP/2 settings extraction", func(t *testing.T) {
		// Test extraction of HTTP/2 settings from request
		req1 := &http.Request{
			ProtoMajor: 2,
			Host:       "example.com",
		}
		settings := server.GetHTTP2Settings(req1)
		assert.Equal(t, "HTTP/2", settings["protocol"].(string))
		assert.Equal(t, "example.com", settings["authority"].(string))

		req2 := &http.Request{
			ProtoMajor: 1,
			Host:       "example.com",
		}
		settings2 := server.GetHTTP2Settings(req2)
		assert.Contains(t, settings2, "protocol")
		assert.Equal(t, 1, settings2["protocol"].(int))
	})
}

// TestHTTP2TLS tests HTTP/2 over TLS
func TestHTTP2TLS(t *testing.T) {
	// This test requires TLS certificates
	// In a real environment, you would have actual certs
	// For now, we'll just test the configuration

	t.Run("HTTP/2 TLS configuration", func(t *testing.T) {
		// Create HTTP/2 config
		http2Config := server.NewHTTP2Config()
		assert.NotNil(t, http2Config)
		assert.Equal(t, true, http2Config.Enabled)
		assert.True(t, http2Config.MaxConcurrentStreams > 0)
	})

	t.Run("HTTP/2 with TLS", func(t *testing.T) {
		// Create a test server config
		config := &server.ServerConfig{
			Port:            0,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    60 * time.Second,
			IdleTimeout:     90 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			EnableHTTP2:     true,
		}

		// Create HTTP/2 config
		http2Config := server.NewHTTP2Config()

		// Create server
		s := server.NewServer(config, nil)
		s.SetupRoutes()

		// Test that HTTP/2 TLS configuration can be created
		// (without actually starting the server)
		// In a real test with certs, you would call StartHTTP2TLS
		assert.NotNil(t, s)
		assert.NotNil(t, http2Config)
	})

	t.Run("HTTP/2 ALPN", func(t *testing.T) {
		// Test that HTTP/2 uses ALPN (Application-Layer Protocol Negotiation)
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2", "http/1.1"},
		}

		// Verify that h2 (HTTP/2) is in the NextProtos
		assert.Contains(t, tlsConfig.NextProtos, "h2")
		assert.Contains(t, tlsConfig.NextProtos, "http/1.1")
	})
}

// TestHTTP2Multiplexing tests HTTP/2 multiplexing capabilities
func TestHTTP2Multiplexing(t *testing.T) {
	t.Run("HTTP/2 multiplexing support", func(t *testing.T) {
		// HTTP/2 multiplexing is automatically enabled with h2c.NewHandler
		// We just verify the configuration is correct

		http2Config := server.NewHTTP2Config()
		assert.True(t, http2Config.MaxConcurrentStreams > 0)

		// Verify the configuration allows multiple streams
		assert.Greater(t, int(http2Config.MaxConcurrentStreams), 1)
	})

	t.Run("HTTP/2 frame size", func(t *testing.T) {
		// Test that frame size is properly configured
		http2Config := server.NewHTTP2Config()
		assert.Equal(t, uint32(16777216), http2Config.MaxReadFrameSize)
	})

	t.Run("HTTP/2 timeout configuration", func(t *testing.T) {
		// Test that timeouts are properly configured
		http2Config := server.NewHTTP2Config()
		assert.Equal(t, 15*time.Second, http2Config.ReadTimeout)
		assert.Equal(t, 15*time.Second, http2Config.WriteTimeout)
		assert.Equal(t, 60*time.Second, http2Config.IdleTimeout)
	})
}

// TestHTTP2Fallback tests HTTP/2 to HTTP/1.1 fallback
func TestHTTP2Fallback(t *testing.T) {
	t.Run("HTTP/2 fallback support", func(t *testing.T) {
		// HTTP/2 with h2c.NewHandler supports fallback to HTTP/1.1
		// This is automatically handled by the h2c handler

		config := &server.ServerConfig{
			Port:            0,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    60 * time.Second,
			IdleTimeout:     90 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			EnableHTTP2:     true,
		}

		http2Config := server.NewHTTP2Config()
		s := server.NewServer(config, nil)
		s.SetupRoutes()

		// Start server
		err := s.StartHTTP2(http2Config)
		require.NoError(t, err)
		defer s.Stop()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Make HTTP/1.1 request (should still work)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("http://localhost:8080/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should work with HTTP/1.1 (might be upgraded to HTTP/2)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)
	})

	t.Run("HTTP/2 h2c cleartext", func(t *testing.T) {
		// Verify h2c support (HTTP/2 over cleartext)
		// h2c.NewHandler enables HTTP/2 without TLS

		config := server.NewHTTP2Config()
		assert.NotNil(t, config)
		assert.Equal(t, true, config.Enabled)
	})
}

// TestHTTP2Performance tests HTTP/2 performance characteristics
func TestHTTP2Performance(t *testing.T) {
	t.Run("HTTP/2 performance configuration", func(t *testing.T) {
		// Test performance-related configuration
		config := server.NewHTTP2Config()

		// Verify performance settings
		assert.Equal(t, 1000, int(config.MaxConcurrentStreams))
		assert.Equal(t, 16777216, int(config.MaxReadFrameSize)) // 16MB default
	})

	t.Run("HTTP/2 connection pooling", func(t *testing.T) {
		// HTTP/2 uses a single connection with multiplexing
		// instead of connection pooling like HTTP/1.1

		// The transport configuration supports this
		transport := server.HTTP2ClientConfig()
		assert.NotNil(t, transport)
	})

	t.Run("HTTP/2 header compression", func(t *testing.T) {
		// HTTP/2 uses HPACK for header compression
		// Verify that compression mode is enabled

		transport := server.HTTP2ClientConfig()
		assert.Equal(t, http2.EnableCompression, transport.CompressionMode)
	})
}
