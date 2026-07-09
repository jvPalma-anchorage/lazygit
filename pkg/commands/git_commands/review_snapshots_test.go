package git_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviewSnapshotStoreRoundtrip(t *testing.T) {
	t.Setenv("LAZYGIT_PR_REVIEW_CACHE_DIR", t.TempDir())
	store := NewReviewSnapshotStore("unused-config-dir", "owner", "repo", 7)

	type payload struct {
		Name  string
		Count int
	}
	assert.NoError(t, store.Write("meta", "headoid123", payload{Name: "x", Count: 3}))

	var got payload
	oid, fetchedAt, ok := store.Read("meta", &got)
	assert.True(t, ok)
	assert.Equal(t, "headoid123", oid)
	assert.NotEmpty(t, fetchedAt)
	assert.Equal(t, payload{Name: "x", Count: 3}, got)
}

func TestReviewSnapshotStoreMissIsNotAnError(t *testing.T) {
	t.Setenv("LAZYGIT_PR_REVIEW_CACHE_DIR", t.TempDir())
	store := NewReviewSnapshotStore("unused", "owner", "repo", 7)

	var got struct{ Name string }

	// Absent file.
	_, _, ok := store.Read("meta", &got)
	assert.False(t, ok)

	// Corrupt envelope.
	assert.NoError(t, os.MkdirAll(store.Dir(), 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(store.Dir(), "meta.json"), []byte("{not json"), 0o644))
	_, _, ok = store.Read("meta", &got)
	assert.False(t, ok)

	// Envelope whose payload doesn't unmarshal into the target.
	assert.NoError(t, os.WriteFile(filepath.Join(store.Dir(), "meta.json"),
		[]byte(`{"fetchedAt":"x","headRefOid":"h","payload":[1,2,3]}`), 0o644))
	_, _, ok = store.Read("meta", &got)
	assert.False(t, ok)
}

func TestReviewSnapshotStoreWriteReplacesAtomically(t *testing.T) {
	t.Setenv("LAZYGIT_PR_REVIEW_CACHE_DIR", t.TempDir())
	store := NewReviewSnapshotStore("unused", "o", "r", 1)

	type payload struct{ V int }
	assert.NoError(t, store.Write("files", "oid1", payload{V: 1}))
	assert.NoError(t, store.Write("files", "oid2", payload{V: 2}))

	var got payload
	oid, _, ok := store.Read("files", &got)
	assert.True(t, ok)
	assert.Equal(t, "oid2", oid)
	assert.Equal(t, 2, got.V)

	// No temp litter left behind.
	entries, err := os.ReadDir(store.Dir())
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestReviewSnapshotStoreScopesByPr(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LAZYGIT_PR_REVIEW_CACHE_DIR", root)

	type payload struct{ V int }
	assert.NoError(t, NewReviewSnapshotStore("unused", "own", "rep", 1).Write("meta", "a", payload{V: 1}))
	assert.NoError(t, NewReviewSnapshotStore("unused", "own", "rep", 2).Write("meta", "b", payload{V: 2}))

	var got payload
	oid, _, ok := NewReviewSnapshotStore("unused", "own", "rep", 1).Read("meta", &got)
	assert.True(t, ok)
	assert.Equal(t, "a", oid)
	assert.Equal(t, 1, got.V)

	// Layout is {owner}/{repo}/{number}.
	assert.FileExists(t, filepath.Join(root, "own", "rep", "2", "meta.json"))
}
