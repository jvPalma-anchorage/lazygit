package context

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/icons"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// PrReviewContext is the dedicated PR review mode context. It is a file-tree list
// context (reusing the CommitFileTreeViewModel for nesting/collapse/cursor, fed
// synthetic CommitFiles built from the PR's changed files) whose side panel shows
// the changed-file tree with a per-file reviewed marker. The cursor file drives an
// inline diff (with review threads interleaved) rendered into the main view by
// PrReviewController.GetOnRenderToMain. The session data (threads, comments,
// reviewers, reviewed set, diff mode) lives here because this is the persistent
// context focused on boot.
type PrReviewContext struct {
	*filetree.CommitFileTreeViewModel
	*ListContextTrait

	c *ContextCommon

	// target identifies the pull request under review. Set by SetTarget from the
	// boot path before the context is focused.
	target prReviewTarget

	// loadStarted guards against re-fetching on every focus; the fetch runs once
	// per review session (reset by Reload after a write).
	loadStarted bool

	// Populated on the UI thread once the worker fetch completes.
	data    *git_commands.PullRequestReviewData
	files   []*models.GithubPullRequestFile
	loadErr error

	// renderMarkdown renders comment/description bodies through glow (Gui-level
	// dependency, injected from the boot path so the context package stays free of
	// the Gui type). May be nil, in which case bodies render as plain text.
	renderMarkdown func(body string, width int) string

	// reviewed is the session-scoped, in-memory set of files the user has marked
	// reviewed, keyed by file path (design D8). It is orthogonal to the git index
	// and not persisted across restarts; it only drives the tree/header indicator.
	reviewed map[string]bool

	// diffMode selects unified vs side-by-side rendering of the selected file's diff
	// (design D9). Unified is the default and the guaranteed path.
	diffMode presentation.DiffMode

	// syntheticFiles backs the CommitFileTreeViewModel: one synthetic CommitFile per
	// changed PR file (Path + single-letter status), rebuilt from files on load.
	// fileByPath maps a tree node's path back to the rich PR file for diff rendering.
	syntheticFiles []*models.CommitFile
	fileByPath     map[string]*models.GithubPullRequestFile
}

type prReviewTarget struct {
	owner  string
	repo   string
	number int
}

var (
	_ types.IListContext = (*PrReviewContext)(nil)
	_ types.Context      = (*PrReviewContext)(nil)
)

func NewPrReviewContext(c *ContextCommon) *PrReviewContext {
	self := &PrReviewContext{
		c:          c,
		reviewed:   map[string]bool{},
		diffMode:   presentation.DiffModeUnified,
		fileByPath: map[string]*models.GithubPullRequestFile{},
	}

	viewModel := filetree.NewCommitFileTreeViewModel(
		func() []*models.CommitFile { return self.syntheticFiles },
		c.Common,
		c.UserConfig().Gui.ShowFileTree,
	)
	self.CommitFileTreeViewModel = viewModel

	getDisplayStrings := func(_ int, _ int) [][]string {
		if self.loadErr != nil {
			return [][]string{{style.FgRed.Sprint(fmt.Sprintf(c.Tr.PrReviewLoadError, self.loadErr.Error()))}}
		}
		if self.data == nil {
			return [][]string{{style.FgYellow.Sprint(c.Tr.PrReviewLoading)}}
		}
		if viewModel.Len() == 0 {
			return [][]string{{style.FgRed.Sprint(c.Tr.PrReviewNoChangedFiles)}}
		}
		showFileIcons := icons.IsIconEnabled() && c.UserConfig().Gui.ShowFileIcons
		lines := presentation.RenderPrReviewFileTree(viewModel, self.reviewed, showFileIcons, &c.UserConfig().Gui.CustomIcons, c.Tr)
		return lo.Map(lines, func(line string, _ int) []string {
			return []string{line}
		})
	}

	baseContext := NewBaseContext(NewBaseContextOpts{
		Kind:       types.SIDE_CONTEXT,
		View:       c.Views().PrReview,
		WindowName: "files",
		Key:        PR_REVIEW_CONTEXT_KEY,
		Focusable:  true,
	})
	simpleContext := NewSimpleContext(baseContext)
	// Fetch the review data the first time the panel is focused (task 3.6).
	simpleContext.AddOnFocusFn(func(types.OnFocusOpts) { self.startLoad() })

	self.ListContextTrait = &ListContextTrait{
		Context: simpleContext,
		ListRenderer: ListRenderer{
			list:              viewModel,
			getDisplayStrings: getDisplayStrings,
		},
		c: c,
	}

	return self
}

// EnsureLoaded triggers the one-time review-data fetch if it has not started yet.
// It is idempotent (guarded by loadStarted) and is called from the controller's
// render-to-main hook, which is guaranteed to fire on boot — unlike the on-focus
// callback, which does not reliably run for the initial boot context.
func (self *PrReviewContext) EnsureLoaded() {
	self.startLoad()
}

// SetTarget records which pull request to review. Called from the boot path
// (resetState) before the context is focused so the on-focus load knows what to
// fetch.
func (self *PrReviewContext) SetTarget(owner string, repo string, number int) {
	self.target = prReviewTarget{owner: owner, repo: repo, number: number}
}

// SetMarkdownRenderer injects the glow-backed markdown renderer (a Gui method)
// from the boot path. Called once, before the context is focused.
func (self *PrReviewContext) SetMarkdownRenderer(renderMarkdown func(body string, width int) string) {
	self.renderMarkdown = renderMarkdown
}

// startLoad kicks off the combined review-data + changed-files fetch off the UI
// thread once per session, then hops back to rebuild the tree and render (D3 /
// task 3.6).
func (self *PrReviewContext) startLoad() {
	if self.loadStarted || self.target.owner == "" {
		return
	}
	self.loadStarted = true

	// Test seam: when LAZYGIT_PR_REVIEW_FIXTURE points at a JSON file, load the PR
	// data + changed files from it synchronously instead of hitting gh/GitHub. This
	// lets integration tests drive the full review UI headlessly.
	if path := os.Getenv("LAZYGIT_PR_REVIEW_FIXTURE"); path != "" {
		if err := self.loadFixture(path); err != nil {
			self.loadErr = err
		}
		self.rerender()
		return
	}

	target := self.target
	self.c.OnWorker(func(_ gocui.Task) error {
		// Host is github.com in v1 (gh-dash launches against github.com).
		token := self.c.Git().GitHub.GetAuthToken("github.com")

		data, dataErr := self.c.Git().GitHub.FetchPRReviewData(target.owner, target.repo, target.number, token)
		var files []*models.GithubPullRequestFile
		var filesErr error
		if dataErr == nil {
			files, filesErr = self.c.Git().GitHub.FetchPRChangedFiles(target.owner, target.repo, target.number, token)
		}

		self.c.OnUIThread(func() error {
			switch {
			case dataErr != nil:
				self.loadErr = dataErr
			case filesErr != nil:
				self.loadErr = filesErr
			default:
				// Clear any prior error so a successful Reload stops rendering a
				// stale failure message.
				self.loadErr = nil
				self.data = data
				self.setFiles(files)
			}
			self.rerender()
			return nil
		})
		return nil
	})
}

// loadFixture loads PR review data + changed files from a JSON fixture (test seam).
func (self *PrReviewContext) loadFixture(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var fx struct {
		Data  *git_commands.PullRequestReviewData
		Files []*models.GithubPullRequestFile
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		return err
	}
	self.data = fx.Data
	self.setFiles(fx.Files)
	return nil
}

// setFiles rebuilds the synthetic CommitFile slice (and the path->PR-file lookup)
// that backs the tree, then rebuilds the tree from it.
func (self *PrReviewContext) setFiles(files []*models.GithubPullRequestFile) {
	self.files = files
	self.syntheticFiles = make([]*models.CommitFile, 0, len(files))
	self.fileByPath = make(map[string]*models.GithubPullRequestFile, len(files))
	for _, f := range files {
		self.syntheticFiles = append(self.syntheticFiles, &models.CommitFile{
			Path:         f.Filename,
			ChangeStatus: changeStatusLetter(f.Status),
		})
		self.fileByPath[f.Filename] = f
	}
	self.SetTree()
}

// rerender re-renders the file tree (side view) and, via the standard refresh path,
// re-runs focus so the selected file's diff is pushed into the main view and the
// selection highlight stays correct. Must run on the UI thread.
func (self *PrReviewContext) rerender() {
	self.c.PostRefreshUpdate(self)
}

// RangeSelectEnabled disables the list controller's multi-item range selection in
// the review tree. v1 acts on a single file at a time, and leaving range-select on
// would also steal the `v` key (Universal.ToggleRangeSelect) from toggle-reviewed.
func (self *PrReviewContext) RangeSelectEnabled() bool {
	return false
}

// selectedPRFile returns the rich PR file the cursor is on, or nil when the cursor
// is on a directory node or no data is loaded.
func (self *PrReviewContext) selectedPRFile() *models.GithubPullRequestFile {
	node := self.GetSelected()
	if node == nil || node.File == nil {
		return nil
	}
	return self.fileByPath[node.File.Path]
}

// SelectedFile exposes the PR file the cursor is on (nil on a directory node or
// before data loads) so the controller can decide whether Enter has a file to open.
func (self *PrReviewContext) SelectedFile() *models.GithubPullRequestFile {
	return self.selectedPRFile()
}

// RenderSelectedFileDiff renders the inline diff (with review threads interleaved)
// for the file the cursor is on, to the given width. It returns a status string
// while data is loading / on error / when the cursor is not on a file.
func (self *PrReviewContext) RenderSelectedFileDiff(width int) string {
	if self.loadErr != nil {
		return fmt.Sprintf(self.c.Tr.PrReviewLoadError, self.loadErr.Error())
	}
	if self.data == nil {
		return self.c.Tr.PrReviewLoading
	}
	rendered, file := self.RenderedSelectedDiff(width)
	if file == nil {
		return self.c.Tr.PrReviewNoChangedFiles
	}
	return rendered.Content
}

// RenderedSelectedDiff renders the cursor file's inline diff and returns the full
// result (content + the RowKind/RowPatchIdx selection maps) plus the file, for the
// focusable diff context. Returns (nil, nil) when no data or no file is selected.
func (self *PrReviewContext) RenderedSelectedDiff(width int) (*presentation.RenderedReviewDiff, *models.GithubPullRequestFile) {
	if self.data == nil {
		return nil, nil
	}
	file := self.selectedPRFile()
	if file == nil {
		return nil, nil
	}
	rendered := presentation.RenderReviewFileDiff(presentation.ReviewFileDiffOpts{
		Tr:             self.c.Tr,
		Path:           file.Filename,
		Diff:           file.Patch,
		Threads:        self.data.Threads,
		Width:          width,
		RenderMarkdown: self.renderMarkdown,
		Reviewed:       self.reviewed[file.Filename],
		DiffMode:       self.diffMode,
	})
	return rendered, file
}

// RenderConversation renders the PR's global (issue-level) comments and reviewers
// for the secondary main view. Returns an empty string before data is loaded.
func (self *PrReviewContext) RenderConversation(width int) string {
	if self.data == nil {
		return ""
	}
	return presentation.RenderReviewConversation(
		self.c.Tr,
		self.data.IssueComments,
		self.data.Reviewers,
		width,
		self.renderMarkdown,
	)
}

// ToggleReviewedForSelectedFile flips the session-scoped reviewed flag for the file
// the cursor is on (design D8) and re-renders so both the tree marker and the diff
// header update. The reviewed set is in-memory only and orthogonal to the git index.
func (self *PrReviewContext) ToggleReviewedForSelectedFile() {
	file := self.selectedPRFile()
	if file == nil {
		return
	}
	if self.reviewed == nil {
		self.reviewed = map[string]bool{}
	}
	self.reviewed[file.Filename] = !self.reviewed[file.Filename]
	self.rerender()
}

// ToggleDiffMode cycles the selected file's diff layout between unified and
// side-by-side (design D9) and re-renders the main view. The tree is unaffected.
func (self *PrReviewContext) ToggleDiffMode() {
	if self.diffMode == presentation.DiffModeUnified {
		self.diffMode = presentation.DiffModeSideBySide
	} else {
		self.diffMode = presentation.DiffModeUnified
	}
	self.rerender()
}

// RenderedDescription returns the PR title and its body rendered through the
// markdown renderer (glow when available, plain text otherwise) for the
// description panel (design D10). Returns empty strings when no data is loaded.
func (self *PrReviewContext) RenderedDescription(width int) (title string, body string) {
	if self.data == nil {
		return "", ""
	}
	body = self.data.Body
	if self.renderMarkdown != nil && strings.TrimSpace(body) != "" {
		body = self.renderMarkdown(body, width)
	}
	return self.data.Title, body
}

// Target exposes the pull request under review to controllers.
func (self *PrReviewContext) Target() (owner string, repo string, number int) {
	return self.target.owner, self.target.repo, self.target.number
}

// Reload re-fetches the review data (e.g. after a write) so new comments appear.
func (self *PrReviewContext) Reload() {
	self.loadStarted = false
	self.startLoad()
}

// Data exposes the fetched review data to controllers (e.g. the description panel).
func (self *PrReviewContext) Data() *git_commands.PullRequestReviewData {
	return self.data
}

// Files exposes the fetched changed files.
func (self *PrReviewContext) Files() []*models.GithubPullRequestFile {
	return self.files
}

// changeStatusLetter maps a GitHub PR file status to the single-letter change
// status the tree presenter colours (mirroring git diff --name-status letters).
func changeStatusLetter(status string) string {
	switch status {
	case "added":
		return "A"
	case "removed":
		return "D"
	case "renamed":
		return "R"
	case "copied":
		return "C"
	default:
		return "M"
	}
}
