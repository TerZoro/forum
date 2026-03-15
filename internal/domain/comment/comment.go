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
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (c *Comment) UpdateContent(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("content cannot be empty")
	}
	c.Content = content
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Comment) AddLike() {
	c.Likes++
}

func (c *Comment) RemoveLike() {
	if c.Likes > 0 {
		c.Likes--
	}
}

func (c *Comment) AddDislike() {
	c.Dislikes++
}

func (c *Comment) RemoveDislike() {
	if c.Dislikes > 0 {
		c.Dislikes--
	}
}
