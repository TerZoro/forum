package session

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func New(userID string, duration time.Duration) Session {
	return Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(duration),
		CreatedAt: time.Now().UTC(),
	}
}

func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

func (s *Session) GetID() string {
	return s.ID
}

func (s *Session) GetUserID() string {
	return s.UserID
}

func (s *Session) GetExpiresAt() time.Time {
	return s.ExpiresAt
}

func (s *Session) GetCreatedAt() time.Time {
	return s.CreatedAt
}
