package git_commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Per-PR filesystem cache (design D3.1 / Phase 9). Each pull request gets a snapshot
// directory `<lazygit-config-dir>/prReview/{owner}/{repo}/{number}/` holding one JSON
// file per data type (meta.json = PR data + conversation, files.json = changed files,
// checks.json = checks). Every snapshot is wrapped in an envelope recording when it
// was fetched and which head OID it corresponds to, so each data type independently
// knows how fresh it is. Corrupt or missing snapshots are cache misses, never errors;
// writes are atomic (temp file + rename) so a crash can't leave a torn snapshot.

// ReviewSnapshotStore reads/writes the snapshot files for one pull request.
type ReviewSnapshotStore struct {
	dir string
}

// reviewSnapshotEnvelope wraps every snapshot payload with its freshness metadata.
type reviewSnapshotEnvelope struct {
	FetchedAt  string          `json:"fetchedAt"`
	HeadRefOid string          `json:"headRefOid"`
	Payload    json.RawMessage `json:"payload"`
}

// NewReviewSnapshotStore builds the store for one PR. configDir is lazygit's resolved
// config directory (so the cache lands in ~/.config/lazygit/prReview/... on Linux).
// LAZYGIT_PR_REVIEW_CACHE_DIR overrides the root for tests.
func NewReviewSnapshotStore(configDir string, owner string, repo string, number int) *ReviewSnapshotStore {
	root := filepath.Join(configDir, "prReview")
	if override := os.Getenv("LAZYGIT_PR_REVIEW_CACHE_DIR"); override != "" {
		root = override
	}
	return &ReviewSnapshotStore{dir: filepath.Join(root, owner, repo, strconv.Itoa(number))}
}

// Dir exposes the PR's snapshot directory (for logging/inspection).
func (self *ReviewSnapshotStore) Dir() string {
	return self.dir
}

// Read loads the named snapshot ("meta", "files", "checks") into payload and returns
// the head OID it was captured at. ok is false on any miss: absent file, unreadable
// JSON, or a payload that doesn't unmarshal — a stale/corrupt cache must never be an
// error, just a cold boot.
func (self *ReviewSnapshotStore) Read(name string, payload any) (headOid string, fetchedAt string, ok bool) {
	raw, err := os.ReadFile(self.path(name))
	if err != nil {
		return "", "", false
	}
	var envelope reviewSnapshotEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", "", false
	}
	if err := json.Unmarshal(envelope.Payload, payload); err != nil {
		return "", "", false
	}
	return envelope.HeadRefOid, envelope.FetchedAt, true
}

// Write atomically persists the named snapshot: marshal to a temp file in the same
// directory, then rename over the target so readers never observe a torn write.
func (self *ReviewSnapshotStore) Write(name string, headOid string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	envelope, err := json.Marshal(reviewSnapshotEnvelope{
		FetchedAt:  time.Now().UTC().Format(time.RFC3339),
		HeadRefOid: headOid,
		Payload:    raw,
	})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(self.dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(self.dir, name+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(envelope); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, self.path(name))
}

func (self *ReviewSnapshotStore) path(name string) string {
	return filepath.Join(self.dir, name+".json")
}
