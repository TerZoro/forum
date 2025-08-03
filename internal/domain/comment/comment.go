package comment

import "time"

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
