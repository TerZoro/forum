package account

import "errors"

type Account struct {
	name     string
	password string
}

func New(name, password string) (Account, error) {
	if name == "" {
		return Account{}, errors.New("empty username")
	}
	if password == "" {
		return Account{}, errors.New("empty password")
	}

	return Account{
		name:     name,
		password: password,
	}, nil
}

func (a *Account) Name() string {
	return a.name
}

func (a *Account) Password() string {
	return a.password
}
