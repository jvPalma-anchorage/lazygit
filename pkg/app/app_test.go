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

	t.Run("nil target is always fine", func(t *testing.T) {
		assert.NoError(t, ensureReviewModeRepo(nil, t.TempDir(), tr))
	})

	t.Run("review target outside a git work tree fails fast", func(t *testing.T) {
		assert.Error(t, ensureReviewModeRepo(target, t.TempDir(), tr))
	})

	t.Run("review target at the work tree root is fine", func(t *testing.T) {
		repoDir := t.TempDir()
		gitInit(t, repoDir)
		assert.NoError(t, ensureReviewModeRepo(target, repoDir, tr))
	})

	t.Run("review target in a subdirectory of the work tree is fine", func(t *testing.T) {
		// The real-world case: gh-dash leaves the user (or a manual launch) deep
		// inside a monorepo checkout, not at the repo root. git's own work-tree
		// detection walks up, so this must succeed.
		repoDir := t.TempDir()
		gitInit(t, repoDir)
		subDir := filepath.Join(repoDir, "src", "js", "app")
		assert.NoError(t, os.MkdirAll(subDir, 0o700))
		assert.NoError(t, ensureReviewModeRepo(target, subDir, tr))
	})
}
