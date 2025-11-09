package models

import (
	"time"
)

// AuditLog represents audit trail for administrative actions (FR-032 requirement)
type AuditLog struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	OperationType  string    `json:"operation_type"`
	TargetResource string    `json:"target_resource"`
	IPAddress      string    `json:"ip_address"`
	Result         string    `json:"result"`
	Details        string    `json:"details,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Operation types
const (
	OpCreate = "create"
	OpUpdate = "update"
	OpDelete = "delete"
	OpLogin  = "login"
	OpLogout = "logout"
	OpToggle = "toggle"
)

// Result types
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

// Validate validates audit log fields
func (a *AuditLog) Validate() error {
	if a.UserID == "" {
		return ErrInvalidUserID
	}
	if a.Username == "" {
		return ErrInvalidUsername
	}
	if a.OperationType == "" {
		return ErrInvalidOperation
	}
	if a.Result != ResultSuccess && a.Result != ResultFailure {
		return ErrInvalidResult
	}
	return nil
}
