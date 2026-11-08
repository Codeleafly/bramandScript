package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// IntegrityRecord represents a single package or tarball checksum entry.
type IntegrityRecord struct {
	Checksum  string `json:"checksum"`  // SHA-256 hex string
	Algorithm string `json:"algorithm"` // Always "sha256"
	Size      int64  `json:"size"`      // Content size in bytes
	Timestamp int64  `json:"timestamp"` // Unix epoch timestamp
}

// IntegrityDB represents a thread-safe registry of SHA-256 package checksums.
type IntegrityDB struct {
	mu      sync.RWMutex
	path    string
	Records map[string]IntegrityRecord `json:"records"`
}

// ComputeSHA256 returns the hex-encoded SHA-256 checksum of data.
func ComputeSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// ComputeFileSHA256 returns the hex-encoded SHA-256 checksum and size of a file on disk.
func ComputeFileSHA256(filePath string) (string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, f)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hasher.Sum(nil)), size, nil
}

// NewIntegrityDB creates an empty IntegrityDB.
func NewIntegrityDB(dbPath ...string) *IntegrityDB {
	p := ""
	if len(dbPath) > 0 && dbPath[0] != "" {
		p = dbPath[0]
	} else {
		p = GetIntegrityDBPath()
	}
	return &IntegrityDB{
		path:    p,
		Records: make(map[string]IntegrityRecord),
	}
}

// LoadIntegrityDB loads or initializes the integrity database from disk.
// If the database file does not exist, an empty database is returned.
func LoadIntegrityDB(customPath ...string) (*IntegrityDB, error) {
	dbPath := ""
	if len(customPath) > 0 && customPath[0] != "" {
		dbPath = customPath[0]
	} else {
		dbPath = GetIntegrityDBPath()
	}

	db := &IntegrityDB{
		path:    dbPath,
		Records: make(map[string]IntegrityRecord),
	}

	data, err := os.ReadFile(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return db, nil
		}
		return nil, fmt.Errorf("failed to read integrity.db: %w", err)
	}

	if len(data) == 0 {
		return db, nil
	}

	if err := json.Unmarshal(data, &db.Records); err != nil {
		// Fallback for wrapped format {"records": {...}}
		var wrapper struct {
			Records map[string]IntegrityRecord `json:"records"`
		}
		if err2 := json.Unmarshal(data, &wrapper); err2 == nil && wrapper.Records != nil {
			db.Records = wrapper.Records
			return db, nil
		}
		return nil, fmt.Errorf("failed to parse integrity.db: %w", err)
	}

	return db, nil
}

// Save serializes the integrity database to disk atomically.
func (db *IntegrityDB) Save(customPath ...string) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	targetPath := db.path
	if len(customPath) > 0 && customPath[0] != "" {
		targetPath = customPath[0]
	}
	if targetPath == "" {
		targetPath = GetIntegrityDBPath()
	}

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(db.Records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize integrity.db: %w", err)
	}

	// Write to temporary file in the same directory for atomic rename
	tmpFile := filepath.Join(targetDir, fmt.Sprintf(".integrity_%d_%d.tmp", os.Getpid(), time.Now().UnixNano()))
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temporary integrity file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to write integrity data: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to sync integrity file: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to close temporary integrity file: %w", err)
	}

	// Atomic replace
	if err := os.Rename(tmpFile, targetPath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to commit integrity.db: %w", err)
	}

	return nil
}

// UpdateIntegrityDB performs a thread-safe and process-safe transactional mutation
// on the integrity database under an exclusive file lock.
func UpdateIntegrityDB(baseDir string, fn func(db *IntegrityDB) error) error {
	lock, err := LockIntegrity(baseDir)
	if err != nil {
		return fmt.Errorf("failed to acquire integrity lock: %w", err)
	}
	defer func() {
		_ = lock.Release()
	}()

	dbPath := GetIntegrityDBPath(baseDir)
	db, err := LoadIntegrityDB(dbPath)
	if err != nil {
		return err
	}

	if err := fn(db); err != nil {
		return err
	}

	return db.Save(dbPath)
}

// Set records or updates a checksum entry in the integrity database.
func (db *IntegrityDB) Set(key, checksum string, size int64) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.Records[key] = IntegrityRecord{
		Checksum:  checksum,
		Algorithm: "sha256",
		Size:      size,
		Timestamp: time.Now().Unix(),
	}
}

// RecordFile calculates the file's SHA-256 and registers it in the database.
func (db *IntegrityDB) RecordFile(key, filePath string) (IntegrityRecord, error) {
	chk, size, err := ComputeFileSHA256(filePath)
	if err != nil {
		return IntegrityRecord{}, err
	}
	rec := IntegrityRecord{
		Checksum:  chk,
		Algorithm: "sha256",
		Size:      size,
		Timestamp: time.Now().Unix(),
	}

	db.mu.Lock()
	db.Records[key] = rec
	db.mu.Unlock()

	return rec, nil
}

// AtomicRecordFile calculates file SHA-256 and transactionally writes it to integrity.db under file lock.
func AtomicRecordFile(baseDir, key, filePath string) (IntegrityRecord, error) {
	chk, size, err := ComputeFileSHA256(filePath)
	if err != nil {
		return IntegrityRecord{}, err
	}
	rec := IntegrityRecord{
		Checksum:  chk,
		Algorithm: "sha256",
		Size:      size,
		Timestamp: time.Now().Unix(),
	}

	err = UpdateIntegrityDB(baseDir, func(db *IntegrityDB) error {
		db.Records[key] = rec
		return nil
	})
	if err != nil {
		return IntegrityRecord{}, err
	}
	return rec, nil
}

// Get retrieves a record for the specified key.
func (db *IntegrityDB) Get(key string) (IntegrityRecord, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rec, ok := db.Records[key]
	return rec, ok
}

// Has checks if a key exists in the database.
func (db *IntegrityDB) Has(key string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	_, ok := db.Records[key]
	return ok
}

// Delete removes a key from the database.
func (db *IntegrityDB) Delete(key string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, ok := db.Records[key]; ok {
		delete(db.Records, key)
		return true
	}
	return false
}

// Keys returns all registered keys sorted alphabetically.
func (db *IntegrityDB) Keys() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	keys := make([]string, 0, len(db.Records))
	for k := range db.Records {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Verify checks whether data matches the registered checksum for key.
func (db *IntegrityDB) Verify(key string, data []byte) (bool, error) {
	rec, ok := db.Get(key)
	if !ok {
		return false, fmt.Errorf("integrity record not found for '%s'", key)
	}
	computed := ComputeSHA256(data)
	if computed != rec.Checksum {
		return false, fmt.Errorf("checksum mismatch for '%s': expected %s, got %s", key, rec.Checksum, computed)
	}
	return true, nil
}

// VerifyFile checks whether the file at filePath matches the registered checksum for key.
func (db *IntegrityDB) VerifyFile(key, filePath string) (bool, error) {
	rec, ok := db.Get(key)
	if !ok {
		return false, fmt.Errorf("integrity record not found for '%s'", key)
	}
	computed, size, err := ComputeFileSHA256(filePath)
	if err != nil {
		return false, err
	}
	if rec.Size > 0 && size != rec.Size {
		return false, fmt.Errorf("size mismatch for '%s': expected %d bytes, got %d bytes", key, rec.Size, size)
	}
	if computed != rec.Checksum {
		return false, fmt.Errorf("checksum mismatch for '%s': expected %s, got %s", key, rec.Checksum, computed)
	}
	return true, nil
}
