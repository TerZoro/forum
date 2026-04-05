package post

import "testing"

func TestNew_Valid(t *testing.T) {
	p, err := New("Hello World", "some content", "author-1", []string{"general"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == "" {
		t.Error("ID should not be empty")
	}
	if p.Likes != 0 || p.Dislikes != 0 {
		t.Error("new post should have zero likes and dislikes")
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Error("timestamps should be set")
	}
}

func TestNew_InvalidInputs(t *testing.T) {
	cases := []struct {
		name     string
		title    string
		content  string
		authorID string
	}{
		{"empty title", "", "content", "author-1"},
		{"whitespace title", "   ", "content", "author-1"},
		{"empty content", "title", "", "author-1"},
		{"whitespace content", "title", "   ", "author-1"},
		{"empty authorID", "title", "content", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := New(c.title, c.content, c.authorID, nil)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestNew_TrimsTitleAndContent(t *testing.T) {
	p, err := New("  trimmed  ", "  content  ", "author-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Title != "trimmed" {
		t.Errorf("title not trimmed: got %q", p.Title)
	}
	if p.Content != "content" {
		t.Errorf("content not trimmed: got %q", p.Content)
	}
}
