package memory

import (
	"context"
	"sync"

	"forum/internal/domain/account"
)

type Memory struct {
	mu       sync.Mutex // prevent race condition
	accounts map[int64]account.Account
	lastID   int64
}

func New(size int) *Memory {
	return &Memory{accounts: make(map[int64]account.Account, size)}
}

func (m *Memory) SignUp(ctx context.Context, a account.Account) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.lastID + 1
	m.accounts[id] = a
	m.lastID = id
	return id, nil
}
