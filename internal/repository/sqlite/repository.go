package sqlite

import (
	"context"
	"database/sql"
	"forum/internal/domain/account"
	"forum/internal/domain/comment"
	"forum/internal/domain/post"
	"forum/internal/domain/session"
)

type Repository struct {
	db *sql.DB
	mu *DatabaseMutex
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

	_, err = db.Exec(
		`CREATE TABLE IF NOT EXISTS sessions (
                id TEXT PRIMARY KEY,
                user_id TEXT NOT NULL,
                expires_at DATETIME NOT NULL,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (user_id) REFERENCES accounts (id)
        );`)
	if err != nil {
		return nil, err
	}

	return &Repository{
		db: db,
		mu: NewDatabaseMutex(),
	}, nil
}

// Account methods
func (r *Repository) SignUp(ctx context.Context, a account.Account) (string, error) {
	r.mu.LockForWrite("account_write")
	defer r.mu.UnlockForWrite("account_write")

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO accounts (id, email, username, password, created_at)
                 VALUES (?, ?, ?, ?, ?)`,
		a.ID, a.Email, a.Username, a.Password, a.CreateAt)
	return a.ID, err
}

func (r *Repository) GetAccountByEmail(ctx context.Context, email string) (account.Account, error) {
	r.mu.LockForRead("account_read")
	defer r.mu.UnlockForRead("account_read")

	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at FROM accounts WHERE email = ?`,
		email).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt)
	return a, err
}

func (r *Repository) GetAccountByID(ctx context.Context, id string) (account.Account, error) {
	r.mu.LockForRead("account_read")
	defer r.mu.UnlockForRead("account_read")

	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at FROM accounts WHERE id = ?`,
		id).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt)
	return a, err
}

func (r *Repository) GetAccountByUsername(ctx context.Context, username string) (account.Account, error) {
	r.mu.LockForRead("account_read")
	defer r.mu.UnlockForRead("account_read")

	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at FROM accounts WHERE username = ?`,
		username).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt)
	return a, err
}

// Post methods
func (r *Repository) CreatePost(ctx context.Context, p post.Post) error {
	r.mu.LockForWrite("post_write")
	defer r.mu.UnlockForWrite("post_write")

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

func (r *Repository) DeletePost(ctx context.Context, postID string) error {
	r.mu.LockForWrite("global_write")
	defer r.mu.UnlockForWrite("global_write")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = r.GetPostByID(ctx, postID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM comment_likes WHERE comment_id IN (SELECT id FROM comments WHERE post_id = ?)`, postID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM post_likes WHERE post_id = ?`, postID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM post_categories WHERE post_id = ?`, postID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM comments WHERE post_id = ?`, postID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM posts WHERE id = ?`, postID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetPosts(ctx context.Context) ([]post.Post, error) {
	r.mu.LockForRead("post_read")
	defer r.mu.UnlockForRead("post_read")

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
	r.mu.LockForRead("post_read")
	defer r.mu.UnlockForRead("post_read")

	var p post.Post
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, content, author_id, likes, dislikes, created_at, updated_at 
		 FROM posts WHERE id = ?`, postID).Scan(&p.ID, &p.Title, &p.Content, &p.AuthorID, &p.Likes, &p.Dislikes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return post.Post{}, err
	}

	categories, err := r.getPostCategories(ctx, p.ID)
	if err != nil {
		return post.Post{}, err
	}
	p.Categories = categories

	return p, nil
}

func (r *Repository) LikePost(ctx context.Context, postID, userID string) error {
	r.mu.LockForWrite("like_operation")
	defer r.mu.UnlockForWrite("like_operation")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingLike *bool
	err = tx.QueryRowContext(ctx,
		`SELECT is_like FROM post_likes WHERE post_id = ? AND user_id = ?`,
		postID, userID).Scan(&existingLike)

	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO post_likes (post_id, user_id, is_like) VALUES (?, ?, ?)`,
			postID, userID, true)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx,
			`UPDATE posts SET likes = likes + 1 WHERE id = ?`,
			postID)
		if err != nil {
			return err
		}
	} else if err == nil && existingLike != nil {
		if *existingLike {
			// User already liked, remove like
			_, err = tx.ExecContext(ctx,
				`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`,
				postID, userID)
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE posts SET likes = likes - 1 WHERE id = ?`,
				postID)
			if err != nil {
				return err
			}
		} else {
			// User disliked, change to like
			_, err = tx.ExecContext(ctx,
				`UPDATE post_likes SET is_like = ? WHERE post_id = ? AND user_id = ?`,
				true, postID, userID)
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE posts SET likes = likes + 1, dislikes = dislikes - 1 WHERE id = ?`,
				postID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) DislikePost(ctx context.Context, postID, userID string) error {
	r.mu.LockForWrite("like_operation")
	defer r.mu.UnlockForWrite("like_operation")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingLike *bool
	err = tx.QueryRowContext(ctx,
		`SELECT is_like from post_likes WHERE post_id = ? AND user_id = ?`,
		postID, userID).Scan(&existingLike)

	if err == sql.ErrNoRows {
		// User hasn't liked/disliked this post yet
		_, err = tx.ExecContext(ctx,
			`INSERT INTO post_likes (post_id, user_id, is_like) VALUES (?, ?, ?)`,
			postID, userID, false) // false = dislike
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx,
			`UPDATE posts SET dislikes = dislikes + 1 WHERE id = ?`,
			postID)
		if err != nil {
			return err
		}
	} else if err == nil && existingLike != nil {
		if !*existingLike {
			// User already disliked, remove dislike
			_, err = tx.ExecContext(ctx,
				`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`,
				postID, userID)
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE posts SET dislikes = dislikes - 1 WHERE id = ?`,
				postID)
			if err != nil {
				return err
			}
		} else {
			// User liked, change to dislike
			_, err = tx.ExecContext(ctx,
				`UPDATE post_likes SET is_like = ? WHERE post_id = ? AND user_id = ?`,
				false, postID, userID) // false = dislike
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE posts SET likes = likes - 1, dislikes = dislikes + 1 WHERE id = ?`,
				postID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) getPostCategories(ctx context.Context, postID string) ([]string, error) {
	r.mu.LockForRead("post_read")
	defer r.mu.UnlockForRead("post_read")

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

func (r *Repository) FilterPosts(ctx context.Context, sortMethod string) ([]post.Post, error) {
	r.mu.LockForRead("post_read")
	defer r.mu.UnlockForRead("post_read")

	var query string
	switch sortMethod {
	case "oldest":
		query = `SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at 
		         FROM posts p ORDER BY p.created_at ASC`
	case "newest":
		query = `SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at 
		         FROM posts p ORDER BY p.created_at DESC`
	case "updated":
		query = `SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at 
		         FROM posts p ORDER BY p.updated_at DESC`
	case "likes":
		query = `SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at 
		         FROM posts p ORDER BY p.likes DESC`
	case "dislikes":
		query = `SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at 
		         FROM posts p ORDER BY p.dislikes DESC`
	default:
		// For any invalid sort method, default to newest
		query = `SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at 
		         FROM posts p ORDER BY p.created_at DESC`
	}

	rows, err := r.db.QueryContext(ctx, query)
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

		categories, err := r.getPostCategories(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		p.Categories = categories

		posts = append(posts, p)
	}

	return posts, nil
}

func (r *Repository) UpdatePost(ctx context.Context, postID, authorID, newTitle, newContent string, newCategories []string) error {
	r.mu.LockForWrite("post_write")
	defer r.mu.UnlockForWrite("post_write")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`UPDATE posts SET title = ?, content = ?, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ? AND author_id = ?`,
		newTitle, newContent, postID, authorID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM post_categories WHERE post_id = ?`, postID)
	if err != nil {
		return err
	}

	for _, category := range newCategories {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO post_categories (post_id, category) VALUES (?, ?)`,
			postID, category)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Comment methods
func (r *Repository) CreateComment(ctx context.Context, c comment.Comment) error {
	r.mu.LockForWrite("comment_write")
	defer r.mu.UnlockForWrite("comment_write")

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO comments (id, content, post_id, author_id, likes, dislikes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Content, c.PostID, c.AuthorID, c.Likes, c.Dislikes, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *Repository) GetCommentByID(ctx context.Context, commentID string) (comment.Comment, error) {
	r.mu.LockForRead("comment_read")
	defer r.mu.UnlockForRead("comment_read")

	var c comment.Comment
	err := r.db.QueryRowContext(ctx,
		`SELECT id, content, post_id, author_id, likes, dislikes, created_at, updated_at 
		 FROM comments WHERE id = ?`, commentID).Scan(&c.ID, &c.Content, &c.PostID, &c.AuthorID, &c.Likes, &c.Dislikes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return comment.Comment{}, err
	}

	return c, nil
}

func (r *Repository) DeleteComment(ctx context.Context, commentID string) error {
	r.mu.LockForWrite("comment_write")
	defer r.mu.UnlockForWrite("comment_write")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = r.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM comment_likes WHERE comment_id = ?`, commentID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, commentID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetCommentsByPost(ctx context.Context, postID string) ([]comment.Comment, error) {
	r.mu.LockForRead("comment_read")
	defer r.mu.UnlockForRead("comment_read")

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

func (r *Repository) LikeComment(ctx context.Context, commentID, userID string) error {
	r.mu.LockForWrite("like_operation")
	defer r.mu.UnlockForWrite("like_operation")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingLike *bool
	err = tx.QueryRowContext(ctx,
		`SELECT is_like FROM comment_likes WHERE comment_id = ? AND user_id = ?`,
		commentID, userID).Scan(&existingLike)

	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO comment_likes (comment_id, user_id, is_like) VALUES (?, ?, ?)`,
			commentID, userID, true)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx,
			`UPDATE comments SET likes = likes + 1 WHERE id = ?`,
			commentID)
		if err != nil {
			return err
		}
	} else if err == nil && existingLike != nil {
		if *existingLike {
			// User already liked, remove like
			_, err = tx.ExecContext(ctx,
				`DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?`,
				commentID, userID)
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE comments SET likes = likes - 1 WHERE id = ?`,
				commentID)
			if err != nil {
				return err
			}
		} else {
			// User disliked, change to like
			_, err = tx.ExecContext(ctx,
				`UPDATE comment_likes SET is_like = ? WHERE comment_id = ? AND user_id = ?`,
				true, commentID, userID)
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE comments SET likes = likes + 1, dislikes = dislikes - 1 WHERE id = ?`,
				commentID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) DislikeComment(ctx context.Context, commentID, userID string) error {
	r.mu.LockForWrite("like_operation")
	defer r.mu.UnlockForWrite("like_operation")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingLike *bool
	err = tx.QueryRowContext(ctx,
		`SELECT is_like from comment_likes WHERE comment_id = ? AND user_id = ?`,
		commentID, userID).Scan(&existingLike)

	if err == sql.ErrNoRows {
		// User hasn't liked/disliked this post yet
		_, err = tx.ExecContext(ctx,
			`INSERT INTO comment_likes (comment_id, user_id, is_like) VALUES (?, ?, ?)`,
			commentID, userID, false) // false = dislike
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx,
			`UPDATE comments SET dislikes = dislikes + 1 WHERE id = ?`,
			commentID)
		if err != nil {
			return err
		}
	} else if err == nil && existingLike != nil {
		if !*existingLike {
			// User already disliked, remove dislike
			_, err = tx.ExecContext(ctx,
				`DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?`,
				commentID, userID)
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE comments SET dislikes = dislikes - 1 WHERE id = ?`,
				commentID)
			if err != nil {
				return err
			}
		} else {
			// User liked, change to dislike
			_, err = tx.ExecContext(ctx,
				`UPDATE comment_likes SET is_like = ? WHERE comment_id = ? AND user_id = ?`,
				false, commentID, userID) // false = dislike
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE comments SET likes = likes - 1, dislikes = dislikes + 1 WHERE id = ?`,
				commentID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) UpdateComment(ctx context.Context, id, authorID, content string) error {
	r.mu.LockForWrite("comment_write")
	defer r.mu.UnlockForWrite("comment_write")

	result, err := r.db.ExecContext(ctx,
		`UPDATE comments SET content = ?, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ? AND author_id = ?`,
		content, id, authorID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Session methods

func (r *Repository) CreateSession(ctx context.Context, s session.Session) error {
	r.mu.LockForWrite("session_write")
	defer r.mu.UnlockForWrite("session_write")

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at, created_at) VALUES
                (?, ?, ?, ?)`,
		s.GetID(), s.GetUserID(), s.GetExpiresAt(), s.GetCreatedAt())

	return err
}

func (r *Repository) GetSession(ctx context.Context, sessionID string) (session.Session, error) {
	r.mu.LockForRead("session_read")
	defer r.mu.UnlockForRead("session_read")

	var s session.Session
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?`,
		sessionID).Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)

	return s, err
}

func (r *Repository) DeleteSession(ctx context.Context, sessionID string) error {
	r.mu.LockForWrite("session_write")
	defer r.mu.UnlockForWrite("session_write")

	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)

	return err
}
