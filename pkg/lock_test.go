package pkg

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileLock_MutualExclusion(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	var counter int64
	var wg sync.WaitGroup
	workers := 10
	iterations := 5

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				lock, err := AcquireFileLock(lockPath, 5*time.Second)
				if err != nil {
					t.Errorf("worker %d failed to acquire lock: %v", workerID, err)
					return
				}

				// Critical section: read counter, sleep briefly, increment
				current := atomic.LoadInt64(&counter)
				time.Sleep(2 * time.Millisecond)
				atomic.StoreInt64(&counter, current+1)

				_ = lock.Release()
				time.Sleep(1 * time.Millisecond)
			}
		}(w)
	}

	wg.Wait()

	expected := int64(workers * iterations)
	if counter != expected {
		t.Fatalf("race condition detected: expected counter %d, got %d", expected, counter)
	}
}

func TestFileLock_Timeout(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "timeout.lock")

	lock1, err := AcquireFileLock(lockPath, 2*time.Second)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer lock1.Release()

	// In a separate goroutine or thread, trying to acquire with short timeout should timeout
	start := time.Now()
	_, err = AcquireFileLock(lockPath, 150*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected timeout error, but lock was acquired")
	}
	if elapsed < 100*time.Millisecond {
		t.Fatalf("timeout returned too quickly: %v", elapsed)
	}
}

func TestFileLock_StaleLockRecovery(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "stale.lock")

	// Write a mock stale lock file with an obsolete PID (e.g. 9999999) and old mtime
	_ = os.WriteFile(lockPath, []byte("9999999\n1000000000\n"), 0600)
	oldTime := time.Now().Add(-2 * time.Minute)
	_ = os.Chtimes(lockPath, oldTime, oldTime)

	// AcquireFileLock should detect the dead PID / stale lock and acquire successfully
	lock, err := AcquireFileLock(lockPath, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to recover from stale lock: %v", err)
	}
	defer lock.Release()

	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file should exist after acquisition: %v", err)
	}
}

func TestLockIntegrityAndPackage(t *testing.T) {
	tempDir := t.TempDir()

	lock1, err := LockIntegrity(tempDir, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to lock integrity: %v", err)
	}
	_ = lock1.Release()

	lock2, err := LockPackage("github.com/alice/pkg@1.0.0", tempDir, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to lock package: %v", err)
	}
	_ = lock2.Release()
}
