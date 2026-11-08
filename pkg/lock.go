package pkg

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// DefaultLockTimeout is the maximum time to wait for a file lock before giving up.
	DefaultLockTimeout = 15 * time.Second

	// StaleLockThreshold is the age after which a lock file is considered abandoned if process is dead.
	StaleLockThreshold = 30 * time.Second
)

type procLock struct {
	ch chan struct{}
}

var (
	procLockMu     sync.Mutex
	inProcessLocks = make(map[string]*procLock)
)

func getInProcessLock(key string) *procLock {
	procLockMu.Lock()
	defer procLockMu.Unlock()
	pl, ok := inProcessLocks[key]
	if !ok {
		pl = &procLock{ch: make(chan struct{}, 1)}
		inProcessLocks[key] = pl
	}
	return pl
}

// FileLock represents an inter-process and inter-thread file lock.
type FileLock struct {
	path     string
	file     *os.File
	lockKey  string
	procLock *procLock
}

// AcquireFileLock acquires an exclusive lock file at lockPath.
// If the lock is held by another process or thread, it waits up to timeout.
func AcquireFileLock(lockPath string, timeout ...time.Duration) (*FileLock, error) {
	to := DefaultLockTimeout
	if len(timeout) > 0 && timeout[0] > 0 {
		to = timeout[0]
	}

	cleanPath := filepath.Clean(lockPath)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// 1. In-process channel-based lock with timeout
	pl := getInProcessLock(cleanPath)
	timer := time.NewTimer(to)
	defer timer.Stop()

	start := time.Now()
	select {
	case pl.ch <- struct{}{}:
	case <-timer.C:
		return nil, fmt.Errorf("timeout (%v) waiting for in-process lock '%s'", to, cleanPath)
	}

	// 2. Inter-process filesystem lock
	remaining := to - time.Since(start)
	if remaining <= 0 {
		remaining = 50 * time.Millisecond
	}
	deadline := time.Now().Add(remaining)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		// Attempt atomic file creation (O_CREATE | O_EXCL)
		f, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err == nil {
			// Successfully created and locked
			_, _ = fmt.Fprintf(f, "%d\n%d\n", os.Getpid(), time.Now().Unix())
			_ = f.Sync()
			return &FileLock{
				path:     cleanPath,
				file:     f,
				lockKey:  cleanPath,
				procLock: pl,
			}, nil
		}

		if !os.IsExist(err) {
			select {
			case <-pl.ch:
			default:
			}
			return nil, fmt.Errorf("failed to open lock file '%s': %w", cleanPath, err)
		}

		// File exists - check for stale lock
		if isLockStale(cleanPath) {
			_ = os.Remove(cleanPath)
			continue
		}

		if time.Now().After(deadline) {
			select {
			case <-pl.ch:
			default:
			}
			return nil, fmt.Errorf("timeout (%v) waiting for file lock '%s'", to, cleanPath)
		}

		// Jittered backoff between 15ms and 35ms
		sleepMs := 15 + rng.Intn(20)
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
	}
}

// Release releases the lock file and in-process mutex.
func (l *FileLock) Release() error {
	if l == nil {
		return nil
	}
	defer func() {
		if l.procLock != nil {
			select {
			case <-l.procLock.ch:
			default:
			}
		}
	}()

	var firstErr error
	if l.file != nil {
		if err := l.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		l.file = nil
	}

	if l.path != "" {
		if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// WithFileLock executes fn within an acquired file lock.
func WithFileLock(lockPath string, timeout time.Duration, fn func() error) error {
	lock, err := AcquireFileLock(lockPath, timeout)
	if err != nil {
		return err
	}
	defer func() {
		_ = lock.Release()
	}()

	return fn()
}

// LockIntegrity locks the global integrity database for read/write modifications.
func LockIntegrity(baseDir string, timeout ...time.Duration) (*FileLock, error) {
	lockPath := filepath.Join(GetLocksDir(baseDir), "integrity.lock")
	return AcquireFileLock(lockPath, timeout...)
}

// LockPackage locks a package name/target during download or extraction.
func LockPackage(target, baseDir string, timeout ...time.Duration) (*FileLock, error) {
	sanitized := sanitizeLockName(target)
	lockPath := filepath.Join(GetLocksDir(baseDir), "pkg_"+sanitized+".lock")
	return AcquireFileLock(lockPath, timeout...)
}

func sanitizeLockName(name string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "@", "_", " ", "_")
	return r.Replace(name)
}

func isLockStale(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	age := time.Since(fi.ModTime())
	if age < 5*time.Second {
		// Definitely not stale if created less than 5 seconds ago
		return false
	}

	// Read lock file content to inspect PID
	data, err := os.ReadFile(path)
	if err != nil {
		return age > StaleLockThreshold
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		pid, err := strconv.Atoi(lines[0])
		if err == nil && pid > 0 {
			// Check if process is still alive
			if !isProcessAlive(pid) {
				return true
			}
		}
	}

	return age > StaleLockThreshold
}

func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix/POSIX, signal 0 checks for process existence without sending a signal
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
