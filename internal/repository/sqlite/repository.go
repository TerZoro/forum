package sqlite

import (
	"context"
	"database/sql"
	"forum/internal/domain/account"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) (*Repository, error) {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS accounts (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                name TEXT NOT NULL UNIQUE,
                password TEXT NOT NULL,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`)
	if err != nil {
		return nil, err
	}

	return &Repository{db: db}, nil
}

// AccountData can be data model
// type AccountData struct{}

func (r *Repository) SignUp(ctx context.Context, a account.Account) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO accounts (name, password) VALUES (?, ?)`,
		a.Name(), a.Password())
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}
