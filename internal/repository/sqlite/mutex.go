package sqlite

import (
	"sync"
)

type DatabaseMutex struct {
	global sync.RWMutex

	accounts sync.RWMutex
	posts    sync.RWMutex
	comments sync.RWMutex
	sessions sync.RWMutex

	likes sync.Mutex
}

func NewDatabaseMutex() *DatabaseMutex {
	return &DatabaseMutex{}
}

func (dm *DatabaseMutex) LockForWrite(operation string) {
	switch operation {
	case "account_write":
		dm.accounts.Lock()
	case "post_write":
		dm.posts.Lock()
	case "comment_write":
		dm.comments.Lock()
	case "session_write":
		dm.sessions.Lock()
	case "like_operation":
		dm.likes.Lock()
	case "global_write":
		dm.global.Lock()
	default:
		dm.global.Lock()
	}
}

func (dm *DatabaseMutex) UnlockForWrite(operation string) {
	switch operation {
	case "account_write":
		dm.accounts.Unlock()
	case "post_write":
		dm.posts.Unlock()
	case "comment_write":
		dm.comments.Unlock()
	case "session_write":
		dm.sessions.Unlock()
	case "like_operation":
		dm.likes.Unlock()
	case "global_write":
		dm.global.Unlock()
	default:
		dm.global.Unlock()
	}
}

func (dm *DatabaseMutex) LockForRead(operation string) {
	switch operation {
	case "account_read":
		dm.accounts.RLock()
	case "post_read":
		dm.posts.RLock()
	case "comment_read":
		dm.comments.RLock()
	case "session_read":
		dm.sessions.RLock()
	case "global_read":
		dm.global.RLock()
	default:
		dm.global.RLock()
	}
}

func (dm *DatabaseMutex) UnlockForRead(operation string) {
	switch operation {
	case "account_read":
		dm.accounts.RUnlock()
	case "post_read":
		dm.posts.RUnlock()
	case "comment_read":
		dm.comments.RUnlock()
	case "session_read":
		dm.sessions.RUnlock()
	case "global_read":
		dm.global.RUnlock()
	default:
		dm.global.RUnlock()
	}
}
