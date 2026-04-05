package comment

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        string
	Content   string
	PostID    string
	AuthorID  string
	Likes     int
	Dislikes  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func New(content, postID, authorID string) (Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Comment{}, errors.New("content empty")
	}
	if postID == "" {
		return Comment{}, errors.New("post ID empty")
	}
	if authorID == "" {
		return Comment{}, errors.New("author ID empty")
	}

	return Comment{
		ID:        uuid.New().String(),
		Content:   content,
		PostID:    postID,
		AuthorID:  authorID,
		Likes:     0,
		Dislikes:  0,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

