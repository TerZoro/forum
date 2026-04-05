package post

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID         string
	Title      string
	Content    string
	AuthorID   string
	Categories []string
	Likes      int
	Dislikes   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func New(title, content, authorID string, categories []string) (Post, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" {
		return Post{}, errors.New("title empty")
	}
	if content == "" {
		return Post{}, errors.New("content empty")
	}
	if authorID == "" {
		return Post{}, errors.New("author ID empty")
	}

	return Post{
		ID:         uuid.New().String(),
		Title:      title,
		Content:    content,
		AuthorID:   authorID,
		Categories: categories,
		Likes:      0,
		Dislikes:   0,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

