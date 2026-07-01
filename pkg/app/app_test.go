package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	appTypes "github.com/jesseduffield/lazygit/pkg/app/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestEnsureReviewModeRepo(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	target := &appTypes.ReviewTarget{Owner: "dlvhdr", Repo: "gh-dash", PRNumber: 123}

	gitInit := func(t *testing.T, dir string) {
		t.Helper()
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		assert.NoError(t, cmd.Run())
	}

	// These cases exercise only the work-tree gate (skipLiveChecks=true), since the
	// live gh/identity checks need the gh CLI and a matching remote.
	t.Run("nil target is always fine", func(t *testing.T) {
		assert.NoError(t, ensureReviewModeRepo(nil, t.TempDir(), tr, true))
	})

	t.Run("review target outside a git work tree fails fast", func(t *testing.T) {
		assert.Error(t, ensureReviewModeRepo(target, t.TempDir(), tr, true))
	})

	t.Run("review target at the work tree root is fine", func(t *testing.T) {
		repoDir := t.TempDir()
		gitInit(t, repoDir)
		assert.NoError(t, ensureReviewModeRepo(target, repoDir, tr, true))
	})

	t.Run("review target in a subdirectory of the work tree is fine", func(t *testing.T) {
		// The real-world case: gh-dash leaves the user (or a manual launch) deep
		// inside a monorepo checkout, not at the repo root. git's own work-tree
		// detection walks up, so this must succeed.
		repoDir := t.TempDir()
		gitInit(t, repoDir)
		subDir := filepath.Join(repoDir, "src", "js", "app")
		assert.NoError(t, os.MkdirAll(subDir, 0o700))
		assert.NoError(t, ensureReviewModeRepo(target, subDir, tr, true))
	})
}

func TestParseRepoIdentity(t *testing.T) {
	scenarios := []struct {
		url   string
		host  string
		owner string
		repo  string
		ok    bool
	}{
		{"https://github.com/owner/repo.git", "github.com", "owner", "repo", true},
		{"https://github.com/owner/repo", "github.com", "owner", "repo", true},
		{"git@github.com:owner/repo.git", "github.com", "owner", "repo", true},
		{"ssh://git@github.com/owner/repo.git", "github.com", "owner", "repo", true},
		{"https://ghe.example.com/Org/Project.git", "ghe.example.com", "Org", "Project", true},
		{"https://github.com/owner/repo/", "github.com", "owner", "", false},
		{"not-a-url", "", "", "", false},
		{"", "", "", "", false},
	}

	for _, s := range scenarios {
		t.Run(s.url, func(t *testing.T) {
			host, owner, repo, ok := parseRepoIdentity(s.url)
			assert.Equal(t, s.ok, ok)
			if s.ok {
				assert.Equal(t, s.host, host)
				assert.Equal(t, s.owner, owner)
				assert.Equal(t, s.repo, repo)
			}
		})
	}
}

func TestCheckoutMatchesTarget(t *testing.T) {
	// Matches case-insensitively against any github.com remote (the base checkout for
	// a fork PR).
	assert.True(t, checkoutMatchesTarget(
		[]string{"git@github.com:other/thing.git", "https://github.com/dlvhdr/gh-dash.git"},
		"dlvhdr", "gh-dash"))
	assert.True(t, checkoutMatchesTarget(
		[]string{"https://github.com/DLVHDR/GH-DASH.git"}, "dlvhdr", "gh-dash"))
	// No remote resolves to the target → mismatch.
	assert.False(t, checkoutMatchesTarget(
		[]string{"git@github.com:someoneelse/otherrepo.git"}, "dlvhdr", "gh-dash"))
	assert.False(t, checkoutMatchesTarget(nil, "dlvhdr", "gh-dash"))
	// A look-alike path on a non-GitHub host must NOT match (the host is verified).
	assert.False(t, checkoutMatchesTarget(
		[]string{"https://evil.example/dlvhdr/gh-dash.git"}, "dlvhdr", "gh-dash"))
}
