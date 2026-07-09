package presentation

import (
	"strings"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/icons"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// unresolvedCommentMarker leads a row that has unresolved review comments; a plain-width
// blank keeps rows without comments aligned under it.
const (
	unresolvedCommentMarker = "💬 "
	noCommentMarker         = "   "
)

// RenderPrReviewFileTree renders the pull request's changed files as a nested,
// collapsible tree (reusing the generic filetree traversal in renderAux). The
// per-file/-folder VIEWED state is conveyed by the NAME's color — mirroring lazygit's
// own staged/unstaged coloring (unviewed = default, fully viewed = green, a folder
// with a mix = yellow) — rather than a checkbox column. The leading column instead
// shows a 💬 marker on any file or folder that has unresolved review comments.
//
// It deliberately does NOT reuse RenderCommitFileTree: that one resolves each file's
// custom-patch status through the PatchBuilder + a commit ref, neither of which exists
// in PR review mode (and dereferencing a nil ref would panic).
func RenderPrReviewFileTree(
	tree *filetree.CommitFileTreeViewModel,
	reviewed map[string]bool,
	unresolved map[string]bool,
	showFileIcons bool,
	customIconsConfig *config.CustomIconsConfig,
) []string {
	collapsedPaths := tree.CollapsedPaths()
	return renderAux(tree.GetRoot().Raw(), collapsedPaths, -1, -1, func(node *filetree.Node[models.CommitFile], treeDepth int, visualDepth int, isCollapsed bool) string {
		return getPrReviewFileLine(isCollapsed, treeDepth, visualDepth, node, reviewed, unresolved, showFileIcons, customIconsConfig)
	})
}

// getPrReviewFileLine renders one tree row: a 💬 marker column (when the file/folder
// has unresolved comments), then the change-status letter (files) or the collapse
// arrow (folders), then the name colored by viewed state.
func getPrReviewFileLine(
	isCollapsed bool,
	treeDepth int,
	visualDepth int,
	node *filetree.Node[models.CommitFile],
	reviewed map[string]bool,
	unresolved map[string]bool,
	showFileIcons bool,
	customIconsConfig *config.CustomIconsConfig,
) string {
	indentation := strings.Repeat("  ", visualDepth)
	name := utils.EscapeSpecialChars(commitFileNameAtDepth(node, treeDepth))
	file := node.File
	isDirectory := file == nil

	output := indentation

	// 💬 marker: on a file with unresolved comments, or a folder any of whose
	// descendant files has them.
	if nodeHasUnresolvedComments(node, unresolved) {
		output += unresolvedCommentMarker
	} else {
		output += noCommentMarker
	}

	nameStyle := prReviewViewedStyle(node, reviewed)

	if isDirectory {
		arrow := EXPANDED_ARROW
		if isCollapsed {
			arrow = COLLAPSED_ARROW
		}
		output += theme.DefaultTextColor.Sprint(arrow) + " "
	} else {
		status := file.ChangeStatus
		output += getColorForChangeStatus(status).Sprint(status) + " "
	}

	if showFileIcons {
		icon := icons.IconForFile(name, false, false, isDirectory, customIconsConfig)
		paint := color.HEX(icon.Color, false)
		output += paint.Sprint(icon.Icon) + " "
	}

	output += nameStyle.Sprint(name)
	return output
}

// prReviewViewedStyle picks the name color from the viewed state: a file is green when
// viewed, default otherwise; a folder is green when ALL its changed files are viewed,
// yellow when only SOME are, and default when none are — the same all/partial/none
// coloring lazygit uses for staged/unstaged files.
func prReviewViewedStyle(node *filetree.Node[models.CommitFile], reviewed map[string]bool) style.TextStyle {
	if node.File != nil {
		if reviewed[node.File.Path] {
			return style.FgGreen
		}
		return theme.DefaultTextColor
	}

	total, viewedCount := 0, 0
	_ = node.ForEachFile(func(f *models.CommitFile) error {
		total++
		if reviewed[f.Path] {
			viewedCount++
		}
		return nil
	})
	switch {
	case total > 0 && viewedCount == total:
		return style.FgGreen
	case viewedCount > 0:
		return style.FgYellow
	default:
		return theme.DefaultTextColor
	}
}

// nodeHasUnresolvedComments reports whether a file has unresolved comments, or a
// folder has any descendant file that does.
func nodeHasUnresolvedComments(node *filetree.Node[models.CommitFile], unresolved map[string]bool) bool {
	if len(unresolved) == 0 {
		return false
	}
	if node.File != nil {
		return unresolved[node.File.Path]
	}
	found := false
	_ = node.ForEachFile(func(f *models.CommitFile) error {
		if unresolved[f.Path] {
			found = true
		}
		return nil
	})
	return found
}
