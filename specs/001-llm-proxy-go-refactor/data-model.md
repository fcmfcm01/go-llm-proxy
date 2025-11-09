# Data Model: LLM代理服务器

**Date**: 2025-11-08
**Feature**: specs/001-llm-proxy-go-refactor/spec.md

## Overview

This document defines the data model for the LLM proxy server.

## Core Entities

### 1. Provider

Represents an LLM service provider configuration.

```go
type Provider struct {
    ID          string    `json:"id" validate:"required,alphanum,min=3,max=20"`
    Name        string    `json:"name" validate:"required,min=2,max=50"`
    APIURL      string    `json:"api_url" validate:"required,url"`
    Enabled     bool      `json:"enabled" default:"true"`
    Priority    int       `json:"priority" validate:"min=1,max=100"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    Timeout     time.Duration `json:"timeout" validate:"min=1s,max=60s"`
    MaxRetries  int           `json:"max_retries" validate:"min=0,max=5"`
}
```

### 2. ModelMapping

Represents model name mapping between providers.

```go
type ModelMapping struct {
    ID              string    `json:"id"`
    SourceProvider  string    `json:"source_provider"`
    SourceModel     string    `json:"source_model"`
    TargetProvider  string    `json:"target_provider"`
    TargetModel     string    `json:"target_model"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

### 3. User

Represents system users.

```go
type User struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    PasswordHash string    `json:"password_hash"`
    Role         UserRole  `json:"role"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}
```

### 4. Session

Represents user authentication sessions.

```go
type Session struct {
    ID          string    `json:"id"`
    UserID      string    `json:"user_id"`
    Username    string    `json:"username"`
    Role        UserRole  `json:"role"`
    CreatedAt   time.Time `json:"created_at"`
    ExpiresAt   time.Time `json:"expires_at"`
    LastActive  time.Time `json:"last_active"`
    IPAddress   string    `json:"ip_address"`
    UserAgent   string    `json:"user_agent"`
    CSRFToken   string    `json:"csrf_token"`
}
```

## Relationships

```
User 1----* Session
Provider 1----* ModelMapping
```

All entities include proper validation rules and state transitions.

### 5. AuditLog

Represents audit trail for administrative actions (FR-032 requirement).

```go
type AuditLog struct {
    ID            string    `json:"id"`
    Timestamp     time.Time `json:"timestamp"`
    UserID        string    `json:"user_id"`
    Username      string    `json:"username"`
    OperationType string    `json:"operation_type"`
    TargetResource string   `json:"target_resource"`
    IPAddress     string    `json:"ip_address"`
    Result        string    `json:"result"`
    Details       string    `json:"details,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
}
```

## Relationships

```
User 1----* Session
Provider 1----* ModelMapping
AuditLog *----1 User
```

## Validation Rules

### Provider
- ID: 3-20 characters, alphanumeric only
- Name: 2-50 characters
- APIURL: Must be valid URL
- Priority: 1-100 (lower number = higher priority)

### ModelMapping
- SourceModel/TargetModel: Required, 1-100 characters
- Provider IDs: Must exist in Provider collection

### User
- Username: 3-30 characters, alphanumeric and underscore only
- PasswordHash: bcrypt/Argon2 hash (60 characters)
- Role: enum (admin, user)

### Session
- ID: 32+ character random string
- ExpiresAt: Must be in the future
- 24-hour default timeout (FR-013)

### AuditLog
- OperationType: enum (create, update, delete, login, logout, toggle)
- Result: enum (success, failure)
- Timestamps: RFC3339 format
- All fields required except Details
- Ordered chronologically by Timestamp
