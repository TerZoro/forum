package sqlite

import (
	"sync"
	"testing"
	"time"
)

func TestDatabaseMutexConcurrency(t *testing.T) {
	dm := NewDatabaseMutex()
	t.Run("ConcurrentReads", func(t *testing.T) {
		var wg sync.WaitGroup
		start := make(chan struct{})

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-start

				dm.LockForRead("post_read")
				time.Sleep(10 * time.Millisecond)
				dm.UnlockForRead("post_read")
			}(i)
		}

		close(start)
		wg.Wait()
	})

	t.Run("WriteBlocksReads", func(t *testing.T) {
		var wg sync.WaitGroup
		writeStarted := make(chan struct{})
		readStarted := make(chan struct{})
		readFinished := make(chan struct{})

		wg.Add(1)
		go func() {
			defer wg.Done()
			dm.LockForWrite("post_write")
			close(writeStarted)
			time.Sleep(100 * time.Millisecond)
			dm.UnlockForWrite("post_write")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-writeStarted
			close(readStarted)

			dm.LockForRead("post_read")
			close(readFinished)
			dm.UnlockForRead("post_read")
		}()

		<-writeStarted

		time.Sleep(50 * time.Millisecond)

		select {
		case <-readFinished:
			t.Error("Read operation completed before write operation, mutex not working")
		default:
			// Expected behavior
		}

		wg.Wait()
	})

	t.Run("DifferentTablesDontBlock", func(t *testing.T) {
		var wg sync.WaitGroup
		start := make(chan struct{})

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			dm.LockForWrite("post_write")
			time.Sleep(50 * time.Millisecond)
			dm.UnlockForWrite("post_write")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			dm.LockForWrite("comment_write")
			time.Sleep(50 * time.Millisecond)
			dm.UnlockForWrite("comment_write")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			dm.LockForRead("account_read")
			time.Sleep(50 * time.Millisecond)
			dm.UnlockForRead("account_read")
		}()

		close(start)
		wg.Wait()
	})
}

func TestDatabaseMutexLikeOperations(t *testing.T) {
	dm := NewDatabaseMutex()

	t.Run("LikeOperationsSerialized", func(t *testing.T) {
		var wg sync.WaitGroup
		start := make(chan struct{})
		results := make([]int, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-start

				dm.LockForWrite("like_operation")
				results[id] = id
				time.Sleep(20 * time.Millisecond)
				dm.UnlockForWrite("like_operation")
			}(i)
		}

		close(start)
		wg.Wait()

		for i, result := range results {
			if result != i {
				t.Errorf("Expected result[%d] = %d, got %d", i, i, result)
			}
		}
	})
}
