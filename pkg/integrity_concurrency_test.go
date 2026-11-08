package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestIntegrityDB_ConcurrentUpdates(t *testing.T) {
	tempDir := t.TempDir()
	_ = EnsureDirectoryStructure(tempDir)

	dbPath := GetIntegrityDBPath(tempDir)
	workers := 20
	recordsPerWorker := 5

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for r := 0; r < recordsPerWorker; r++ {
				key := fmt.Sprintf("pkg_worker_%d_rec_%d", workerID, r)
				checksum := fmt.Sprintf("sha256_mock_hash_%d_%d", workerID, r)

				err := UpdateIntegrityDB(tempDir, func(db *IntegrityDB) error {
					db.Set(key, checksum, int64(workerID*100+r))
					return nil
				})
				if err != nil {
					t.Errorf("worker %d failed to update integrity db: %v", workerID, err)
				}
			}
		}(w)
	}

	wg.Wait()

	// Load resulting database and verify all records are intact
	db, err := LoadIntegrityDB(dbPath)
	if err != nil {
		t.Fatalf("failed to load final integrity db: %v", err)
	}

	expectedCount := workers * recordsPerWorker
	if len(db.Records) != expectedCount {
		t.Fatalf("expected %d records in integrity db, got %d", expectedCount, len(db.Records))
	}

	// Verify every single record
	for w := 0; w < workers; w++ {
		for r := 0; r < recordsPerWorker; r++ {
			key := fmt.Sprintf("pkg_worker_%d_rec_%d", w, r)
			rec, ok := db.Get(key)
			if !ok {
				t.Fatalf("missing record '%s'", key)
			}
			expectedChecksum := fmt.Sprintf("sha256_mock_hash_%d_%d", w, r)
			if rec.Checksum != expectedChecksum {
				t.Fatalf("record '%s' checksum mismatch: expected %s, got %s", key, expectedChecksum, rec.Checksum)
			}
		}
	}

	// Verify file is valid JSON directly
	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("failed to read raw integrity.db file: %v", err)
	}
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("integrity.db file contains corrupted JSON: %v", err)
	}
}
