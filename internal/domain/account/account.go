package account

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Account struct {
	ID       string
	Email    string
	Username string
	Password string
	CreateAt time.Time
}

func New(email, name, password string) (Account, error) {
	if email == "" {
		return Account{}, errors.New("empty email")
	}
	if name == "" {
		return Account{}, errors.New("empty username")
	}
	if password == "" {
		return Account{}, errors.New("empty password")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return Account{}, errors.New("password hashing lost")
	}

	return Account{
		ID:       uuid.New().String(),
		Email:    email,
		Username: name,
		Password: string(bytes),
		CreateAt: time.Now(),
	}, nil
}
