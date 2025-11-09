package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Entity is a common interface for all entities
type Entity interface {
	SetID(id string)
	SetCreatedAt(t time.Time)
	SetUpdatedAt(t time.Time)
}

// generateID generates a unique ID
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based ID
		return fmt.Sprintf("id_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// generateCSRFToken generates a CSRF token
func generateCSRFToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based token
		return fmt.Sprintf("csrf_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// Time helpers

// Now returns the current time
func Now() time.Time {
	return time.Now()
}

// TimeBefore checks if t1 is before t2
func TimeBefore(t1, t2 time.Time) bool {
	return t1.Before(t2)
}

// TimeAfter checks if t1 is after t2
func TimeAfter(t1, t2 time.Time) bool {
	return t1.After(t2)
}

// TimeEqual checks if t1 equals t2 (within 1 second precision)
func TimeEqual(t1, t2 time.Time) bool {
	diff := t1.Sub(t2)
	return diff < time.Second && diff > -time.Second
}

// Duration helpers

// ParseDuration parses a duration string
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// Hours converts hours to duration
func Hours(h int) time.Duration {
	return time.Hour * time.Duration(h)
}

// Minutes converts minutes to duration
func Minutes(m int) time.Duration {
	return time.Minute * time.Duration(m)
}

// Seconds converts seconds to duration
func Seconds(s int) time.Duration {
	return time.Second * time.Duration(s)
}

// FormatDuration formats a duration to a human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
}
