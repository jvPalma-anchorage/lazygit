package presentation

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/stretchr/testify/assert"
)

func fileNode(path string) *filetree.Node[models.CommitFile] {
	return &filetree.Node[models.CommitFile]{File: &models.CommitFile{Path: path}}
}

func folderNode(children ...*filetree.Node[models.CommitFile]) *filetree.Node[models.CommitFile] {
	return &filetree.Node[models.CommitFile]{Children: children}
}

func TestPrReviewViewedStyle(t *testing.T) {
	viewed := map[string]bool{"a": true, "b": true}

	// A file is green when viewed, default otherwise.
	assert.Equal(t, style.FgGreen, prReviewViewedStyle(fileNode("a"), viewed))
	assert.Equal(t, theme.DefaultTextColor, prReviewViewedStyle(fileNode("c"), viewed))

	// A folder: all viewed → green, some → yellow, none → default.
	allViewed := folderNode(fileNode("a"), fileNode("b"))
	someViewed := folderNode(fileNode("a"), fileNode("c"))
	noneViewed := folderNode(fileNode("c"), fileNode("d"))
	assert.Equal(t, style.FgGreen, prReviewViewedStyle(allViewed, viewed))
	assert.Equal(t, style.FgYellow, prReviewViewedStyle(someViewed, viewed))
	assert.Equal(t, theme.DefaultTextColor, prReviewViewedStyle(noneViewed, viewed))
}

func TestNodeHasUnresolvedComments(t *testing.T) {
	unresolved := map[string]bool{"a": true}

	assert.True(t, nodeHasUnresolvedComments(fileNode("a"), unresolved))
	assert.False(t, nodeHasUnresolvedComments(fileNode("b"), unresolved))

	// A folder inherits the flag from any descendant file.
	assert.True(t, nodeHasUnresolvedComments(folderNode(fileNode("b"), fileNode("a")), unresolved))
	assert.False(t, nodeHasUnresolvedComments(folderNode(fileNode("b"), fileNode("c")), unresolved))

	// No unresolved data → never marks.
	assert.False(t, nodeHasUnresolvedComments(fileNode("a"), map[string]bool{}))
}
