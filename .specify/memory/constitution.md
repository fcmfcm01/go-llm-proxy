# Go LLM Proxy Constitution

## Core Principles

### I. Idiomatic Go Development (NON-NEGOTIABLE)

All code MUST follow Go community best practices and official guidelines. Go code must be idiomatic, readable, and maintainable using standard library where possible. Every package must include comprehensive documentation explaining purpose, usage, and behavior. Code style MUST follow `gofmt` and `goimports` standards. Error handling MUST be explicit using Go's error type—no ignored errors. Concurrency patterns MUST use goroutines and channels appropriately, with proper synchronization. Context package MUST be used for cancellation, deadlines, and request-scoped values across all API boundaries. Reflection MUST be avoided unless absolutely necessary; type safety is paramount.

**Rationale**: Idiomatic Go ensures code is maintainable, performant, and understandable by the Go community. Standard practices reduce cognitive load and prevent subtle bugs.

### II. Test-Driven Development (NON-NEGOTIABLE)

Test-First Development is mandatory: Tests written → User approved → Tests fail → Then implement. Red-Green-Refactor cycle strictly enforced. Unit test coverage MUST be ≥ 85% for all packages, with critical paths (proxy handler, converter, config manager) requiring ≥ 95% coverage. Every public function and method MUST have corresponding tests. Table-driven tests MUST be used for multiple input scenarios. Benchmarks MUST be written for performance-critical code (proxy forwarding, format conversion). All tests MUST run in CI pipeline without network dependencies (use mocks). TDD cycle: (1) Write failing test with exact scenario, (2) Write minimal code to pass, (3) Refactor while keeping tests green.

**Rationale**: TDD ensures correctness, prevents regressions, and creates comprehensive test suites. High coverage is critical for a proxy service where failures impact all clients.

### III. Comprehensive Testing Strategy

Four-layer testing pyramid required: (1) Unit tests for all pure functions and methods, (2) Integration tests for API endpoints with real HTTP calls, (3) Contract tests for provider API compatibility, (4) End-to-end tests for full user workflows. Contract tests MUST verify Anthropic↔OpenAI format conversion accuracy with test vectors. Integration tests MUST test actual proxy behavior with mock provider backends. Performance tests MUST validate p95 latency < 10ms for conversion, < 100ms for proxy forwarding. Load tests MUST verify service handles 1000+ concurrent connections. All test suites MUST run in < 30 seconds locally, < 5 minutes in CI. Failing tests block deployments.

**Rationale**: Multi-layer testing ensures reliability across the entire stack. Proxy services require comprehensive validation to prevent cascading failures.

### IV. User Experience Consistency

All API endpoints MUST follow RESTful conventions with consistent JSON response formats. All responses MUST include success flag, data payload, and meaningful error messages. All errors MUST include HTTP status codes, error codes, and actionable messages. Web UI MUST be consistent with Bootstrap 5 styling and responsive design. Loading states and error states MUST be user-friendly with clear guidance. API versioning MUST be semantic (v1, v2) with backward compatibility for at least one major version. Documentation MUST be auto-generated from code and kept synchronized. All user-facing text MUST be clear, actionable, and consistent in tone.

**Rationale**: Consistent UX reduces user confusion, improves adoption, and simplifies support. Clear error messages accelerate debugging and reduce support burden.

### V. Performance & Resource Efficiency (NON-NEGOTIABLE)

Service MUST maintain memory footprint < 50MB under normal load (vs Python's 100MB+). Response time p95 MUST be < 10ms for format conversion, < 100ms for proxy forwarding. Service MUST handle 1000+ concurrent connections using goroutines efficiently. HTTP client connection pools MUST be configured (MaxIdleConns: 100, MaxIdleConnsPerHost: 10). Object pools MUST be used for frequently allocated structures (sync.Pool for request/response objects). JSON processing MUST use standard encoding/json initially, switch to json-iterator only if profiling shows bottleneck. Goroutine pools SHOULD be considered for CPU-intensive tasks to limit concurrency. Zero-copy patterns MUST be used for large payloads (bytes.Buffer). Resource monitoring via Prometheus metrics is mandatory.

**Rationale**: High performance is critical for proxy services. The migration from Python to Go is driven by performance requirements—these MUST be met and validated.

### VI. Observability & Monitoring

Structured logging is mandatory using logrus with JSON formatter. All logs MUST include timestamps (RFC3339), levels, and contextual fields. Request logging MUST include method, path, status code, duration, and trace ID. All logs MUST go to stdout for container environments. Prometheus metrics MUST be exported: http_requests_total, http_request_duration_seconds, proxy_requests_total, conversion_latency_seconds. Health check endpoints MUST be provided: /healthz (basic), /healthz/detailed (includes dependency checks). Metrics collection MUST be non-blocking (< 1ms overhead). Distributed tracing SHOULD be implemented for complex workflows. Log levels MUST be configurable (debug, info, warn, error).

**Rationale**: Production proxy services require comprehensive observability for debugging, performance tuning, and SLA monitoring.

### VII. Configuration & Deployment

Configuration MUST support YAML/TOML/JSON formats via spf13/viper. All config values MUST support environment variable overrides. Hot-reload capability MUST be implemented for runtime configuration changes. Docker deployment MUST use multi-stage builds to minimize image size. Container memory limits MUST be set (recommend 128MB). Health checks MUST be configured in docker-compose. Configuration validation MUST occur at startup with clear error messages. Secrets MUST never be committed to version control. Configuration files MUST have documented examples. Default values MUST be production-safe (security-focused).

**Rationale**: Flexible configuration enables deployment across environments. Safe defaults and validation prevent misconfigurations in production.

### VIII. Security & Authentication

Authentication MUST be implemented for all admin endpoints using session-based auth. Session timeout MUST be configurable (default 24h). Passwords MUST be hashed using bcrypt or Argon2. HTTPS enforcement in production (redirect HTTP to HTTPS). CORS headers MUST be explicitly configured. Input validation MUST use go-playground/validator for all user inputs. SQL injection prevention through parameterized queries. Rate limiting MUST be implemented (QPS limits per IP). Security headers MUST be set (X-Content-Type-Options, X-Frame-Options, etc.). Dependencies MUST be scanned for vulnerabilities. No credentials in logs or error messages.

**Rationale**: Proxy services handle sensitive API keys and user data. Security MUST be implemented by default to prevent breaches.

## Performance Standards

**Service Level Objectives**:
- p95 latency for format conversion: < 10ms
- p95 latency for proxy forwarding: < 100ms
- p99 latency for overall requests: < 500ms
- Memory usage under normal load: < 50MB
- Memory usage under peak load: < 100MB
- Concurrent connection capacity: 1000+
- CPU utilization at 80% load: < 2 cores
- Container startup time: < 3 seconds
- Health check response time: < 5ms
- Zero-downtime deployments: Required

**Measurement**: All metrics MUST be measured using Prometheus and validated via load testing. Performance regressions MUST fail CI pipeline.

**Load Testing Requirements**: Before each release, run load test with 1000 concurrent users for 10 minutes. Success criteria: < 1% error rate, p95 latency within SLO, memory stable (no leaks).

## Security Requirements

**Authentication & Authorization**:
- Session-based authentication for admin interface
- Configurable session timeout (default 24h)
- Password policy enforcement
- No default credentials in production

**Network Security**:
- HTTPS enforcement in production
- CORS configuration for web UI
- Rate limiting: 100 QPS per IP for admin, 1000 QPS for proxy
- Request size limits: 10MB max

**Data Protection**:
- Provider API keys encrypted at rest
- No credentials in logs
- Input validation on all endpoints
- Secure headers (HSTS, X-Frame-Options, etc.)

**Dependency Security**:
- All dependencies scanned with govulncheck
- Critical vulnerabilities patched within 24h
- High severity within 7 days
- Minimal dependency footprint

## Development Workflow

**Code Review Process**:
- All code MUST be reviewed before merge
- Reviewer MUST verify constitution compliance
- Minimum 1 approval for non-maintainers, 2 for architectural changes
- CI checks MUST pass (tests, lint, security scan)
- Performance impact MUST be assessed for changes

**Quality Gates**:
- Unit test coverage ≥ 85% (≥ 95% for critical paths)
- All tests green in CI
- Lint checks pass (golangci-lint)
- No known high/critical security vulnerabilities
- Performance benchmarks maintained or improved
- Documentation updated for API changes

**Version Control**:
- Semantic versioning (MAJOR.MINOR.PATCH)
- Conventional commits (feat:, fix:, docs:, etc.)
- Release tags for all versions
- Changelog maintained

**CI/CD Pipeline**:
- Automated tests on every PR
- Security scanning on dependencies
- Performance regression testing
- Docker image building and scanning
- Automated deployment to staging (optional)

## Governance

**Constitution Authority**: This constitution supersedes all other development practices and documentation. All contributors MUST comply with these principles.

**Amendment Process**:
1. Propose change with detailed rationale
2. Open RFC issue for discussion
3. Review impact on existing code and processes
4. Obtain approval from project maintainers
5. Update constitution version following semantic versioning
6. Create migration plan for existing code if needed
7. Communicate changes to all contributors

**Versioning Policy**:
- MAJOR: Backward-incompatible changes (principle removal or redefinition)
- MINOR: New principles or materially expanded guidance
- PATCH: Clarifications, wording changes, non-semantic refinements

**Compliance Verification**: All PRs MUST verify constitution compliance during review. Project maintainers MAY request changes to ensure compliance. Critical violations MAY block merges.

**Guidance Reference**: Use this constitution in conjunction with project documentation (Go重构方案.md) for development guidance. When conflicts arise, constitution takes precedence.

**Review Cadence**: Constitution MUST be reviewed quarterly for relevance and effectiveness. Emergency amendments MAY be made for critical security or performance issues.

**Version**: 1.0.0 | **Ratified**: 2025-11-08 | **Last Amended**: 2025-11-08
