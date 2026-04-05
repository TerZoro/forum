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

func (p *Post) UpdateContent(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("content cannot be empty")
	}
	p.Content = content
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Post) UpdateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("title cannot be empty")
	}
	p.Title = title
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Post) AddLike() {
	p.Likes++
}

func (p *Post) RemoveLike() {
	if p.Likes > 0 {
		p.Likes--
	}
}

func (p *Post) AddDislike() {
	p.Dislikes++
}

func (p *Post) RemoveDislike() {
	if p.Dislikes > 0 {
		p.Dislikes--
	}
}
