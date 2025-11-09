package models

import (
	"time"
)

// Provider represents an LLM service provider configuration
type Provider struct {
	ID         string        `json:"id" validate:"required,alphanum,min=3,max=20"`
	Name       string        `json:"name" validate:"required,min=2,max=50"`
	APIURL     string        `json:"api_url" validate:"required,url"`
	APIKey     string        `json:"api_key"`
	Enabled    bool          `json:"enabled" default:"true"`
	Priority   int           `json:"priority" validate:"min=1,max=100"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	Timeout    time.Duration `json:"timeout" validate:"min=1s,max=60s"`
	MaxRetries int           `json:"max_retries" validate:"min=0,max=5"`
}

// Validate validates provider fields
func (p *Provider) Validate() error {
	if len(p.ID) < 3 || len(p.ID) > 20 {
		return ErrInvalidProviderID
	}
	if len(p.Name) < 2 || len(p.Name) > 50 {
		return ErrInvalidProviderName
	}
	if p.APIURL == "" {
		return ErrInvalidProviderURL
	}
	if p.Priority < 1 || p.Priority > 100 {
		return ErrInvalidProviderPriority
	}
	return nil
}
