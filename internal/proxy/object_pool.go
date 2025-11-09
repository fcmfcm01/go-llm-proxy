package proxy

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// BufferPool provides object pooling for byte buffers
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() *bytes.Buffer {
	buf := bp.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	bp.pool.Put(buf)
}

// HTTPRequestPool provides object pooling for HTTP requests
type HTTPRequestPool struct {
	pool sync.Pool
}

// NewHTTPRequestPool creates a new HTTP request pool
func NewHTTPRequestPool() *HTTPRequestPool {
	return &HTTPRequestPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &RequestWrapper{
					Buffer: NewBufferPool(),
				}
			},
		},
	}
}

// Get retrieves a request wrapper from the pool
func (rp *HTTPRequestPool) Get() *RequestWrapper {
	return rp.pool.Get().(*RequestWrapper)
}

// Put returns a request wrapper to the pool
func (rp *HTTPRequestPool) Put(req *RequestWrapper) {
	if req == nil {
		return
	}
	req.Reset()
	rp.pool.Put(req)
}

// RequestWrapper wraps an HTTP request with pool management
type RequestWrapper struct {
	Buffer *BufferPool
	Data   []byte
	ResetAt time.Time
}

// Reset resets the request wrapper for reuse
func (rw *RequestWrapper) Reset() {
	rw.Data = rw.Data[:0]
	rw.ResetAt = time.Now()
}

// ResponsePool provides object pooling for HTTP responses
type ResponsePool struct {
	pool sync.Pool
}

// NewResponsePool creates a new response pool
func NewResponsePool() *ResponsePool {
	return &ResponsePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &ResponseWrapper{
					Buffer: make([]byte, 0, 4096),
				}
			},
		},
	}
}

// Get retrieves a response wrapper from the pool
func (rp *ResponsePool) Get() *ResponseWrapper {
	return rp.pool.Get().(*ResponseWrapper)
}

// Put returns a response wrapper to the pool
func (rp *ResponsePool) Put(resp *ResponseWrapper) {
	if resp == nil {
		return
	}
	resp.Buffer = resp.Buffer[:0]
	rp.pool.Put(resp)
}

// ResponseWrapper wraps an HTTP response with pool management
type ResponseWrapper struct {
	Buffer      []byte
	StatusCode  int
	Headers     map[string]string
	ReceivedAt  time.Time
}

// Writer is a pool-aware io.Writer
type Writer struct {
	pool sync.Pool
}

// NewWriter creates a new pooled writer
func NewWriter() *Writer {
	return &Writer{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

// Get retrieves a buffer from the pool for writing
func (w *Writer) Get() *bytes.Buffer {
	buf := w.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool after writing
func (w *Writer) Put(buf *bytes.Buffer) io.Writer {
	if buf == nil {
		return nil
	}
	w.pool.Put(buf)
	return buf
}

// MultiPool manages multiple object pools
type MultiPool struct {
	bufferPool    *BufferPool
	requestPool   *HTTPRequestPool
	responsePool  *ResponsePool
	writerPool    *Writer
}

// NewMultiPool creates a new multi-pool instance
func NewMultiPool() *MultiPool {
	return &MultiPool{
		bufferPool:   NewBufferPool(),
		requestPool:  NewHTTPRequestPool(),
		responsePool: NewResponsePool(),
		writerPool:   NewWriter(),
	}
}

// GetBuffer retrieves a buffer from the pool
func (mp *MultiPool) GetBuffer() *bytes.Buffer {
	return mp.bufferPool.Get()
}

// PutBuffer returns a buffer to the pool
func (mp *MultiPool) PutBuffer(buf *bytes.Buffer) {
	mp.bufferPool.Put(buf)
}

// GetRequest retrieves a request wrapper from the pool
func (mp *MultiPool) GetRequest() *RequestWrapper {
	return mp.requestPool.Get()
}

// PutRequest returns a request wrapper to the pool
func (mp *MultiPool) PutRequest(req *RequestWrapper) {
	mp.requestPool.Put(req)
}

// GetResponse retrieves a response wrapper from the pool
func (mp *MultiPool) GetResponse() *ResponseWrapper {
	return mp.responsePool.Get()
}

// PutResponse returns a response wrapper to the pool
func (mp *MultiPool) PutResponse(resp *ResponseWrapper) {
	mp.responsePool.Put(resp)
}

// GetWriter retrieves a writer from the pool
func (mp *MultiPool) GetWriter() *bytes.Buffer {
	return mp.writerPool.Get()
}

// PutWriter returns a writer to the pool
func (mp *MultiPool) PutWriter(buf *bytes.Buffer) {
	mp.writerPool.Put(buf)
}

// Global pools for common use
var (
	GlobalBufferPool  = NewBufferPool()
	GlobalRequestPool = NewHTTPRequestPool()
	GlobalResponsePool = NewResponsePool()
	GlobalMultiPool   = NewMultiPool()
)

// WithPool provides a convenient way to use a buffer from the pool
func WithBuffer(fn func(*bytes.Buffer) error) error {
	buf := GlobalBufferPool.Get()
	defer GlobalBufferPool.Put(buf)
	return fn(buf)
}

// ReadCloserPool provides pooling for ReadClosers
type ReadCloserPool struct {
	pool sync.Pool
}

// NewReadCloserPool creates a new ReadCloser pool
func NewReadCloserPool() *ReadCloserPool {
	return &ReadCloserPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &ReadCloserWrapper{
					Data: make([]byte, 0, 4096),
				}
			},
		},
	}
}

// Get retrieves a ReadCloser wrapper from the pool
func (rcp *ReadCloserPool) Get() *ReadCloserWrapper {
	return rcp.pool.Get().(*ReadCloserWrapper)
}

// Put returns a ReadCloser wrapper to the pool
func (rcp *ReadCloserPool) Put(rc *ReadCloserWrapper) {
	if rc == nil {
		return
	}
	rc.Data = rc.Data[:0]
	// Return wrapper to pool
	rcp.pool.Put(rc)
}

// ReadCloserWrapper wraps an io.ReadCloser with pool management
type ReadCloserWrapper struct {
	Data    []byte
	ReadAt  time.Time
}

// NewPooledReadCloser creates a pooled ReadCloser from data
func NewPooledReadCloser(data []byte) io.ReadCloser {
	return &PooledReadCloser{
		Data: data,
		pos:  0,
	}
}

// PooledReadCloser is a pool-aware io.ReadCloser
type PooledReadCloser struct {
	Data []byte
	pos  int
}

// Read reads data from the pooled buffer
func (prc *PooledReadCloser) Read(p []byte) (n int, err error) {
	if prc.pos >= len(prc.Data) {
		return 0, io.EOF
	}
	n = copy(p, prc.Data[prc.pos:])
	prc.pos += n
	return n, nil
}

// Close closes the ReadCloser and returns it to the pool
func (prc *PooledReadCloser) Close() error {
	// Reset position
	prc.pos = 0
	// Return to pool
	GlobalResponsePool.Put(&ResponseWrapper{
		Buffer: prc.Data,
	})
	return nil
}
