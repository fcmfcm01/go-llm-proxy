package models

import (
	"time"
)

// ModelMapping represents model name mapping between providers
type ModelMapping struct {
	ID             string    `json:"id"`
	SourceProvider string    `json:"source_provider"`
	SourceModel    string    `json:"source_model"`
	TargetProvider string    `json:"target_provider"`
	TargetModel    string    `json:"target_model"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Validate validates model mapping fields
func (m *ModelMapping) Validate() error {
	if m.SourceModel == "" || len(m.SourceModel) > 100 {
		return ErrInvalidModelName
	}
	if m.TargetModel == "" || len(m.TargetModel) > 100 {
		return ErrInvalidModelName
	}
	if m.SourceProvider == "" || m.TargetProvider == "" {
		return ErrInvalidProviderID
	}
	return nil
}
