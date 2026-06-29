package presentation

import (
	"strings"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/icons"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// RenderPrReviewFileTree renders the pull request's changed files as a nested,
// collapsible tree (reusing the generic filetree traversal in renderAux) with a
// per-file reviewed marker (design D7/D8). It deliberately does NOT reuse
// RenderCommitFileTree: that one resolves each file's custom-patch status through
// the PatchBuilder + a commit ref, neither of which exists in PR review mode (and
// dereferencing a nil ref would panic), and it has no reviewed-marker column.
func RenderPrReviewFileTree(
	tree *filetree.CommitFileTreeViewModel,
	reviewed map[string]bool,
	showFileIcons bool,
	customIconsConfig *config.CustomIconsConfig,
	tr *i18n.TranslationSet,
) []string {
	collapsedPaths := tree.CollapsedPaths()
	return renderAux(tree.GetRoot().Raw(), collapsedPaths, -1, -1, func(node *filetree.Node[models.CommitFile], treeDepth int, visualDepth int, isCollapsed bool) string {
		return getPrReviewFileLine(isCollapsed, treeDepth, visualDepth, node, reviewed, showFileIcons, customIconsConfig, tr)
	})
}

// getPrReviewFileLine renders one tree row. File rows lead with a reviewed marker
// ([x]/[ ]) followed by the single-letter change status; directory rows pad that
// column so file names stay aligned under their parent.
func getPrReviewFileLine(
	isCollapsed bool,
	treeDepth int,
	visualDepth int,
	node *filetree.Node[models.CommitFile],
	reviewed map[string]bool,
	showFileIcons bool,
	customIconsConfig *config.CustomIconsConfig,
	tr *i18n.TranslationSet,
) string {
	indentation := strings.Repeat("  ", visualDepth)
	name := utils.EscapeSpecialChars(commitFileNameAtDepth(node, treeDepth))
	file := node.File
	isDirectory := file == nil

	output := indentation

	if isDirectory {
		arrow := EXPANDED_ARROW
		if isCollapsed {
			arrow = COLLAPSED_ARROW
		}
		output += theme.DefaultTextColor.Sprint(arrow) + " "
	} else {
		marker := tr.PrReviewUnreviewedMarker
		markerStyle := theme.DefaultTextColor
		if reviewed[file.Path] {
			marker = tr.PrReviewReviewedMarker
			markerStyle = style.FgGreen
		}
		status := file.ChangeStatus
		output += markerStyle.Sprint(marker) + " " + getColorForChangeStatus(status).Sprint(status) + " "
	}

	if showFileIcons {
		icon := icons.IconForFile(name, false, false, isDirectory, customIconsConfig)
		paint := color.HEX(icon.Color, false)
		output += paint.Sprint(icon.Icon) + " "
	}

	output += theme.DefaultTextColor.Sprint(name)
	return output
}
