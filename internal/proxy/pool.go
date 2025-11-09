package proxy

import (
	"net/http"
	"sync"
	"time"
)

// ConnectionPool manages HTTP connections for a provider
type ConnectionPool struct {
	transport *http.Transport
	client    *http.Client
	maxIdle   int
	idleTime  time.Duration
	mu        sync.RWMutex
	stats     PoolStats
}

// PoolStats holds connection pool statistics
type PoolStats struct {
	ActiveConnections   int64         `json:"active_connections"`
	IdleConnections     int64         `json:"idle_connections"`
	TotalConnections    int64         `json:"total_connections"`
	RequestsServed      int64         `json:"requests_served"`
	RequestsFailed      int64         `json:"requests_failed"`
	AverageResponseTime time.Duration `json:"avg_response_time"`
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxIdle int, idleTime time.Duration) *ConnectionPool {
	pool := &ConnectionPool{
		maxIdle:  maxIdle,
		idleTime: idleTime,
		stats: PoolStats{
			ActiveConnections:   0,
			IdleConnections:     0,
			TotalConnections:    0,
			RequestsServed:      0,
			RequestsFailed:      0,
			AverageResponseTime: 0,
		},
	}

	// Create transport with connection pooling
	pool.transport = &http.Transport{
		MaxIdleConns:        maxIdle,
		IdleConnTimeout:     idleTime,
		MaxIdleConnsPerHost: maxIdle / 2,
		DisableCompression:  false,
	}

	// Create HTTP client with the transport
	pool.client = &http.Client{
		Transport: pool.transport,
		Timeout:   30 * time.Second,
	}

	return pool
}

// GetClient returns the HTTP client from the pool
func (p *ConnectionPool) GetClient() *http.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.client
}

// Do performs an HTTP request using the connection pool
func (p *ConnectionPool) Do(req *http.Request) (*http.Response, error) {
	start := time.Now()

	p.mu.Lock()
	p.stats.ActiveConnections++
	p.mu.Unlock()

	resp, err := p.client.Do(req)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.stats.ActiveConnections--
	p.stats.RequestsServed++

	if err != nil {
		p.stats.RequestsFailed++
	} else if resp != nil {
		// Update idle connections count
		// Note: This is a simplified calculation
		p.stats.IdleConnections = int64(p.maxIdle / 2)
		p.stats.TotalConnections = p.stats.ActiveConnections + p.stats.IdleConnections
	}

	// Update average response time
	p.updateAverageResponseTime(time.Since(start))

	return resp, err
}

// updateAverageResponseTime calculates running average
func (p *ConnectionPool) updateAverageResponseTime(duration time.Duration) {
	if p.stats.RequestsServed == 1 {
		p.stats.AverageResponseTime = duration
	} else {
		// Running average
		totalTime := p.stats.AverageResponseTime*time.Duration(p.stats.RequestsServed-1) + duration
		p.stats.AverageResponseTime = totalTime / time.Duration(p.stats.RequestsServed)
	}
}

// GetStats returns connection pool statistics
func (p *ConnectionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// Close closes all idle connections
func (p *ConnectionPool) Close() {
	p.transport.CloseIdleConnections()
}

// ProviderConnectionManager manages connection pools for multiple providers
type ProviderConnectionManager struct {
	pools    map[string]*ConnectionPool
	mu       sync.RWMutex
	defaults ConnectionPoolConfig
}

// ConnectionPoolConfig holds connection pool configuration
type ConnectionPoolConfig struct {
	MaxIdleConnections int
	IdleTimeout        time.Duration
}

// DefaultConnectionPoolConfig returns default connection pool configuration
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxIdleConnections: 100,
		IdleTimeout:        90 * time.Second,
	}
}

// NewProviderConnectionManager creates a new connection manager
func NewProviderConnectionManager(defaults ConnectionPoolConfig) *ProviderConnectionManager {
	if defaults.MaxIdleConnections == 0 {
		defaults = DefaultConnectionPoolConfig()
	}

	return &ProviderConnectionManager{
		pools:    make(map[string]*ConnectionPool),
		defaults: defaults,
	}
}

// GetPool returns or creates a connection pool for a provider
func (m *ProviderConnectionManager) GetPool(providerID string) *ConnectionPool {
	m.mu.RLock()
	pool, exists := m.pools[providerID]
	m.mu.RUnlock()

	if exists {
		return pool
	}

	// Create new pool
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := m.pools[providerID]; exists {
		return pool
	}

	pool = NewConnectionPool(m.defaults.MaxIdleConnections, m.defaults.IdleTimeout)
	m.pools[providerID] = pool

	return pool
}

// RemovePool removes a connection pool for a provider
func (m *ProviderConnectionManager) RemovePool(providerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pool, exists := m.pools[providerID]; exists {
		pool.Close()
		delete(m.pools, providerID)
	}
}

// GetAllStats returns statistics for all provider pools
func (m *ProviderConnectionManager) GetAllStats() map[string]PoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]PoolStats)
	for providerID, pool := range m.pools {
		stats[providerID] = pool.GetStats()
	}

	return stats
}

// CloseAll closes all connection pools
func (m *ProviderConnectionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, pool := range m.pools {
		pool.Close()
	}
	m.pools = make(map[string]*ConnectionPool)
}
