package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"forum/internal/domain/account"
	"time"
	"forum/internal/domain/comment"
	"forum/internal/domain/post"
	"forum/internal/domain/session"
	"strings"
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
                created_at DATETIME NOT NULL,
                is_admin BOOLEAN NOT NULL DEFAULT 0
        );`)
	if err != nil {
		return nil, err
	}

	// Attempt to add is_admin column for existing databases; ignore error if it exists
	_, _ = db.Exec(`ALTER TABLE accounts ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT 0`)

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
		`INSERT INTO accounts (id, email, username, password, created_at, is_admin)
                 VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.Email, a.Username, a.Password, a.CreateAt, a.IsAdmin)
	return a.ID, err
}

func (r *Repository) GetAccountByEmail(ctx context.Context, email string) (account.Account, error) {
	r.mu.LockForRead("account_read")
	defer r.mu.UnlockForRead("account_read")

	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at, is_admin FROM accounts WHERE email = ?`,
		email).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt, &a.IsAdmin)
	return a, err
}

func (r *Repository) GetAccountByID(ctx context.Context, id string) (account.Account, error) {
	r.mu.LockForRead("account_read")
	defer r.mu.UnlockForRead("account_read")

	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at, is_admin FROM accounts WHERE id = ?`,
		id).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt, &a.IsAdmin)
	return a, err
}

func (r *Repository) GetAccountByUsername(ctx context.Context, username string) (account.Account, error) {
	r.mu.LockForRead("account_read")
	defer r.mu.UnlockForRead("account_read")

	var a account.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, username, password, created_at, is_admin FROM accounts WHERE username = ?`,
		username).Scan(&a.ID, &a.Email, &a.Username, &a.Password, &a.CreateAt, &a.IsAdmin)
	return a, err
}

func (r *Repository) GetAccountsCount(ctx context.Context) (int, error) {
	r.mu.LockForRead("account_read")
	defer r.mu.UnlockForRead("account_read")

	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM accounts`).Scan(&n)
	return n, err
}

func (r *Repository) UpdateAccountFields(ctx context.Context, id, newEmail, newUsername, newHashedPassword string) error {
	if newEmail == "" && newUsername == "" && newHashedPassword == "" {
		return nil
	}

	r.mu.LockForWrite("account_write")
	defer r.mu.UnlockForWrite("account_write")

	_, err := r.db.ExecContext(ctx, `
		UPDATE accounts SET
			email    = CASE WHEN ? != '' THEN ? ELSE email    END,
			username = CASE WHEN ? != '' THEN ? ELSE username END,
			password = CASE WHEN ? != '' THEN ? ELSE password END
		WHERE id = ?`,
		newEmail, newEmail,
		newUsername, newUsername,
		newHashedPassword, newHashedPassword,
		id,
	)
	return err
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

	result, err := tx.ExecContext(ctx, `DELETE FROM posts WHERE id = ?`, postID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
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

	return r.scanPosts(ctx, rows)
}

func (r *Repository) GetPostsByAuthor(ctx context.Context, authorID string) ([]post.Post, error) {
	r.mu.LockForRead("post_read")
	defer r.mu.UnlockForRead("post_read")

	rows, err := r.db.QueryContext(ctx,
		`SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at
         FROM posts p WHERE p.author_id = ? ORDER BY p.created_at DESC`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPosts(ctx, rows)
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
	return r.voteOnEntity(ctx, "post_likes", "post_id", "posts", postID, userID, true)
}

func (r *Repository) DislikePost(ctx context.Context, postID, userID string) error {
	return r.voteOnEntity(ctx, "post_likes", "post_id", "posts", postID, userID, false)
}

// scanPosts reads all post rows and attaches categories to each.
// no lock, Caller holds Lock.
func (r *Repository) scanPosts(ctx context.Context, rows *sql.Rows) ([]post.Post, error) {
	var posts []post.Post
	for rows.Next() {
		var p post.Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.AuthorID, &p.Likes, &p.Dislikes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		categories, err := r.getPostCategories(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		p.Categories = categories
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// don't need lock. Caller alr has lock. Will be deadlock.
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
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
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
		query = `SELECT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at 
		         FROM posts p ORDER BY p.created_at DESC`
	}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPosts(ctx, rows)
}

func (r *Repository) FilterPostsByCategory(ctx context.Context, category string) ([]post.Post, error) {
	r.mu.LockForRead("post_read")
	defer r.mu.UnlockForRead("post_read")

	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT p.id, p.title, p.content, p.author_id, p.likes, p.dislikes, p.created_at, p.updated_at
		 FROM posts p
		 JOIN post_categories pc ON p.id = pc.post_id
		 WHERE pc.category = ?
		 ORDER BY p.created_at DESC`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPosts(ctx, rows)
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
		`UPDATE posts SET title = ?, content = ?, updated_at = ?
		 WHERE id = ? AND author_id = ?`,
		newTitle, newContent, time.Now().UTC(), postID, authorID)
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

func (r *Repository) GetPostVoteByUser(ctx context.Context, postID, userID string) (bool, bool, error) {
	r.mu.LockForRead("like_operation")
	defer r.mu.UnlockForRead("like_operation")

	var isLike bool
	err := r.db.QueryRowContext(ctx,
		`SELECT is_like FROM post_likes WHERE post_id = ? AND user_id = ?`,
		postID, userID).Scan(&isLike)
	if err == sql.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return isLike, true, nil
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

	_, err = tx.ExecContext(ctx, `DELETE FROM comment_likes WHERE comment_id = ?`, commentID)
	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, commentID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
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
		if err := rows.Scan(&c.ID, &c.Content, &c.PostID, &c.AuthorID, &c.Likes, &c.Dislikes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, rows.Err()
}

func (r *Repository) GetCommentsByAuthor(ctx context.Context, authorID string) ([]comment.Comment, error) {
	r.mu.LockForRead("comment_read")
	defer r.mu.UnlockForRead("comment_read")

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, content, post_id, author_id, likes, dislikes, created_at, updated_at 
         FROM comments 
         WHERE author_id = ? 
         ORDER BY created_at DESC`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []comment.Comment
	for rows.Next() {
		var c comment.Comment
		if err := rows.Scan(&c.ID, &c.Content, &c.PostID, &c.AuthorID, &c.Likes, &c.Dislikes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, rows.Err()
}

func (r *Repository) LikeComment(ctx context.Context, commentID, userID string) error {
	return r.voteOnEntity(ctx, "comment_likes", "comment_id", "comments", commentID, userID, true)
}

func (r *Repository) DislikeComment(ctx context.Context, commentID, userID string) error {
	return r.voteOnEntity(ctx, "comment_likes", "comment_id", "comments", commentID, userID, false)
}

// voteOnEntity handles the lock and transaction for a like/dislike toggle.
// table/idCol/entityTable are internal constants — not user input.
func (r *Repository) voteOnEntity(ctx context.Context, table, idCol, entityTable, id, userID string, isLike bool) error {
	r.mu.LockForWrite("like_operation")
	defer r.mu.UnlockForWrite("like_operation")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.toggleVote(ctx, tx, table, idCol, entityTable, id, userID, isLike); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) toggleVote(ctx context.Context, tx *sql.Tx, table, idCol, entityTable, id, userID string, isLike bool) error {
	var existing *bool
	err := tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT is_like FROM %s WHERE %s = ? AND user_id = ?`, table, idCol),
		id, userID).Scan(&existing)

	addCol := voteCol(isLike)
	removeCol := voteCol(!isLike)

	if err == sql.ErrNoRows {
		if _, err = tx.ExecContext(ctx,
			fmt.Sprintf(`INSERT INTO %s (%s, user_id, is_like) VALUES (?, ?, ?)`, table, idCol),
			id, userID, isLike); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf(`UPDATE %s SET %s = %s + 1 WHERE id = ?`, entityTable, addCol, addCol),
			id)
		return err
	}
	if err != nil || existing == nil {
		return err
	}

	if *existing == isLike {
		// same vote: remove it
		if _, err = tx.ExecContext(ctx,
			fmt.Sprintf(`DELETE FROM %s WHERE %s = ? AND user_id = ?`, table, idCol),
			id, userID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf(`UPDATE %s SET %s = %s - 1 WHERE id = ?`, entityTable, addCol, addCol),
			id)
		return err
	}

	// opposite vote: flip it
	if _, err = tx.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s SET is_like = ? WHERE %s = ? AND user_id = ?`, table, idCol),
		isLike, id, userID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s SET %s = %s + 1, %s = %s - 1 WHERE id = ?`, entityTable, addCol, addCol, removeCol, removeCol),
		id)
	return err
}

func voteCol(isLike bool) string {
	if isLike {
		return "likes"
	}
	return "dislikes"
}

func (r *Repository) UpdateComment(ctx context.Context, id, authorID, content string) error {
	r.mu.LockForWrite("comment_write")
	defer r.mu.UnlockForWrite("comment_write")

	result, err := r.db.ExecContext(ctx,
		`UPDATE comments SET content = ?, updated_at = ?
		 WHERE id = ? AND author_id = ?`,
		content, time.Now().UTC(), id, authorID)
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

func inPlaceholders(n int) string {
	ph := make([]string, n)
	for i := range ph {
		ph[i] = "?"
	}
	return strings.Join(ph, ",")
}

func (r *Repository) GetCommentVotesByUserForComments(ctx context.Context, userID string, commentIDs []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(commentIDs) == 0 {
		return result, nil
	}

	r.mu.LockForRead("like_operation")
	defer r.mu.UnlockForRead("like_operation")

	args := make([]any, 0, len(commentIDs)+1)
	args = append(args, userID)
	for _, id := range commentIDs {
		args = append(args, id)
	}
	query := `SELECT comment_id, is_like FROM comment_likes WHERE user_id = ? AND comment_id IN (` + inPlaceholders(len(commentIDs)) + `)`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var isLike bool
		if err := rows.Scan(&id, &isLike); err != nil {
			return nil, err
		}
		result[id] = isLike
	}
	return result, nil
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

func (r *Repository) DeleteSessionsByUser(ctx context.Context, userId string) error {
	r.mu.LockForWrite("session_write")
	defer r.mu.UnlockForWrite("session_write")

	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userId)

	return err
}

func (r *Repository) DeleteExpiredSessions(ctx context.Context) error {
	r.mu.LockForWrite("session_write")
	defer r.mu.UnlockForWrite("session_write")

	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= datetime('now')`)
	return err
}
