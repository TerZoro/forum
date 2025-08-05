package sqlite

import (
	"context"
	"database/sql"
	"forum/internal/domain/account"
	"forum/internal/domain/comment"
	"forum/internal/domain/post"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) (*Repository, error) {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS accounts (
                id TEXT PRIMARY KEY,
                email TEXT NOT NULL UNIQUE,
                username TEXT NOT NULL UNIQUE,
                password TEXT NOT NULL,
                created_at DATETIME NOT NULL
        );`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS posts (
                id TEXT PRIMARY KEY,
                title TEXT NOT NULL,
                content TEXT NOT NULL,
                author_id TEXT NOT NULL,
                likes INTEGER DEFAULT 0,
                dislikes INTEGER DEFAULT 0,
                created_at DATETIME NOT NULL,
                updated_at DATETIME NOT NULL,
                FOREIGN KEY (author_id) REFERENCES accounts (id)
        );`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS comments (
                id TEXT PRIMARY KEY,
                content TEXT NOT NULL,
                post_id TEXT NOT NULL,
                author_id TEXT NOT NULL,
                likes INTEGER DEFAULT 0,
                dislikes INTEGER DEFAULT 0,
                created_at DATETIME NOT NULL,
                updated_at DATETIME NOT NULL,
                FOREIGN KEY (post_id) REFERENCES posts (id),
                FOREIGN KEY (author_id) REFERENCES accounts (id)
        );`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS post_categories (
                post_id TEXT NOT NULL,
                category TEXT NOT NULL,
                PRIMARY KEY (post_id, category),
                FOREIGN KEY (post_id) REFERENCES posts (id)
        );`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS post_likes (
                post_id TEXT NOT NULL,
                user_id TEXT NOT NULL,
                is_like BOOLEAN NOT NULL,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                PRIMARY KEY (post_id, user_id),
                FOREIGN KEY (post_id) REFERENCES posts (id),
                FOREIGN KEY (user_id) REFERENCES accounts (id)
        );`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS comment_likes (
                comment_id TEXT NOT NULL,
                user_id TEXT NOT NULL,
                is_like BOOLEAN NOT NULL,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                PRIMARY KEY (comment_id, user_id),
                FOREIGN KEY (comment_id) REFERENCES comments (id),
                FOREIGN KEY (user_id) REFERENCES accounts (id)
        );`)
	if err != nil {
		return nil, err
	}

	return &Repository{db: db}, nil
}

// Account methods
func (r *Repository) SignUp(ctx context.Context, a account.Account) (string, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO accounts (id, email, username, password, created_at)
                 VALUES (?, ?, ?, ?, ?)`,
		a.ID, a.Email, a.Username, a.Password, a.CreateAt)
	return a.ID, err
}

func (r *Repository) GetAccountByEmail(ctx context.Context, email string) (account.Account, error) {
	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at FROM accounts WHERE email = ?`,
		email).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt)
	return a, err
}

func (r *Repository) GetAccountByID(ctx context.Context, id string) (account.Account, error) {
	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at FROM accounts WHERE id = ?`,
		id).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt)
	return a, err
}

func (r *Repository) GetAccountByUsername(ctx context.Context, username string) (account.Account, error) {
	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at FROM accounts WHERE username = ?`,
		username).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt)
	return a, err
}

// Post methods
func (r *Repository) CreatePost(ctx context.Context, p post.Post) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO posts (id, title, content, author_id, likes, dislikes, created_at, updated_at)
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Title, p.Content, p.AuthorID, p.Likes, p.Dislikes, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return err
	}

	for _, category := range p.Categories {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO post_categories (post_id, category) VALUES (?, ?)`,
			p.ID, category)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetPosts(ctx context.Context) ([]post.Post, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at 
		 FROM posts p ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []post.Post
	for rows.Next() {
		var p post.Post
		err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.AuthorID, &p.Likes, &p.Dislikes, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Get categories for this post
		categories, err := r.getPostCategories(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		p.Categories = categories

		posts = append(posts, p)
	}

	return posts, nil
}

func (r *Repository) GetPostByID(ctx context.Context, postID string) (post.Post, error) {
	var p post.Post
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, content, author_id, likes, dislikes, created_at, updated_at 
		 FROM posts WHERE id = ?`, postID).Scan(&p.ID, &p.Title, &p.Content, &p.AuthorID, &p.Likes, &p.Dislikes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return post.Post{}, err
	}

	// Get categories for this post
	categories, err := r.getPostCategories(ctx, p.ID)
	if err != nil {
		return post.Post{}, err
	}
	p.Categories = categories

	return p, nil
}

func (r *Repository) getPostCategories(ctx context.Context, postID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT category FROM post_categories WHERE post_id = ?`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		err := rows.Scan(&category)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// Comment methods
func (r *Repository) CreateComment(ctx context.Context, c comment.Comment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO comments (id, content, post_id, author_id, likes, dislikes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Content, c.PostID, c.AuthorID, c.Likes, c.Dislikes, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *Repository) GetCommentsByPost(ctx context.Context, postID string) ([]comment.Comment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, content, post_id, author_id, likes, dislikes, created_at, updated_at 
		 FROM comments 
		 WHERE post_id = ? 
		 ORDER BY created_at ASC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []comment.Comment
	for rows.Next() {
		var c comment.Comment
		err := rows.Scan(&c.ID, &c.Content, &c.PostID, &c.AuthorID, &c.Likes, &c.Dislikes, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, nil
}
