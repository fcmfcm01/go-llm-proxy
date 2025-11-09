package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
)

// SessionStore defines the interface for session storage
type SessionStore interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, id string) (*models.Session, error)
	GetByUserID(ctx context.Context, userID string) ([]*models.Session, error)
	Update(ctx context.Context, session *models.Session) error
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}

// FilesystemSessionStore implements SessionStore using filesystem
type FilesystemSessionStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewFilesystemSessionStore creates a new filesystem-based session store
func NewFilesystemSessionStore(basePath string) (*FilesystemSessionStore, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	return &FilesystemSessionStore{
		basePath: basePath,
	}, nil
}

// Create creates a new session
func (s *FilesystemSessionStore) Create(ctx context.Context, session *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeSession(session)
}

// GetByID retrieves a session by ID
func (s *FilesystemSessionStore) GetByID(ctx context.Context, id string) (*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionPath := s.getSessionPath(id)
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	var session models.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// GetByUserID retrieves all sessions for a user
func (s *FilesystemSessionStore) GetByUserID(ctx context.Context, userID string) ([]*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sessions []*models.Session

	err := filepath.WalkDir(s.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var session models.Session
		if err := json.Unmarshal(data, &session); err != nil {
			return nil // Skip invalid files
		}

		if session.UserID == userID {
			sessions = append(sessions, &session)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk session directory: %w", err)
	}

	return sessions, nil
}

// Update updates an existing session
func (s *FilesystemSessionStore) Update(ctx context.Context, session *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session exists
	sessionPath := s.getSessionPath(session.ID)
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return ErrSessionNotFound
	}

	return s.writeSession(session)
}

// Delete deletes a session
func (s *FilesystemSessionStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionPath := s.getSessionPath(id)
	err := os.Remove(sessionPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteByUserID deletes all sessions for a user
func (s *FilesystemSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	sessions, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, session := range sessions {
		sessionPath := s.getSessionPath(session.ID)
		if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete session %s: %w", session.ID, err)
		}
	}

	return nil
}

// DeleteExpired deletes all expired sessions
func (s *FilesystemSessionStore) DeleteExpired(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var deletedCount int

	err := filepath.WalkDir(s.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var session models.Session
		if err := json.Unmarshal(data, &session); err != nil {
			return nil // Skip invalid files
		}

		if now.After(session.ExpiresAt) {
			if err := os.Remove(path); err == nil {
				deletedCount++
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return nil
}

// writeSession writes a session to disk
func (s *FilesystemSessionStore) writeSession(session *models.Session) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	sessionPath := s.getSessionPath(session.ID)

	// Write atomically using temp file
	tempPath := sessionPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}

	if err := os.Rename(tempPath, sessionPath); err != nil {
		os.Remove(tempPath) // Cleanup temp file
		return fmt.Errorf("failed to rename session file: %w", err)
	}

	return nil
}

// getSessionPath returns the filesystem path for a session
func (s *FilesystemSessionStore) getSessionPath(sessionID string) string {
	return filepath.Join(s.basePath, sessionID+".json")
}
