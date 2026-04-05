package comment

import "testing"

func TestNew_Valid(t *testing.T) {
	c, err := New("nice post", "post-1", "author-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID == "" {
		t.Error("ID should not be empty")
	}
	if c.Likes != 0 || c.Dislikes != 0 {
		t.Error("new comment should have zero likes and dislikes")
	}
	if c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() {
		t.Error("timestamps should be set")
	}
}

func TestNew_InvalidInputs(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		postID   string
		authorID string
	}{
		{"empty content", "", "post-1", "author-1"},
		{"whitespace content", "   ", "post-1", "author-1"},
		{"empty postID", "content", "", "author-1"},
		{"empty authorID", "content", "post-1", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := New(c.content, c.postID, c.authorID)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestNew_TrimsContent(t *testing.T) {
	c, err := New("  hello  ", "post-1", "author-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Content != "hello" {
		t.Errorf("content not trimmed: got %q", c.Content)
	}
}
