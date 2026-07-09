package context

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
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

	// bypassCacheOnce forces the next load to skip the warm-cache render (manual
	// refresh semantics — D3.1: `R` always hits GitHub). Cleared by startLoad.
	bypassCacheOnce bool

	// loadGeneration invalidates in-flight loads: Retarget bumps it, and a worker
	// whose captured generation no longer matches abandons its work instead of
	// applying another PR's data (codex review finding: retarget mid-flight).
	loadGeneration int

	// pendingComments accumulates comments for a grouped review submission (Phase
	// 7): added from the diff surface, submitted as ONE review (with a summary body
	// and an event) from the tree, then cleared. Session-scoped, in memory only.
	// They deliberately survive a Reload: if the head moved and a line no longer
	// anchors, GitHub's 422 is surfaced at submit time and the queue is retained,
	// so no typed text is ever silently discarded. Retarget clears them (they
	// belong to the abandoned PR).
	pendingComments []git_commands.PendingReviewComment

	// reloadDone, when set by ReloadThen, fires once after the next reload's data is
	// applied (e.g. so the focused diff surface re-renders to show a just-posted
	// comment). Cleared after firing.
	reloadDone func()

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

	// Local-ref read model (DW2 / Phase 2). When active, the changed-file tree is
	// built from `git diff --name-status mergeBase..head` against the fetched
	// refs/lazygit-review/<pr>/{base,head} refs, and the per-file/-folder diff is
	// rendered through the user's pager (delta) rather than the bespoke presenter.
	localRefsActive bool   // the tree/diff are sourced from local refs
	localBase       string // merge-base commit (diff `from`)
	localHeadRef    string // refs/lazygit-review/<pr>/head (diff `to`)

	// fileOidByPath / fileStatusByPath / patchHashByPath carry each changed file's
	// head blob OID, git status, and (fallback) diff hash, used as the persisted
	// "viewed"-state content identity (Phase 4).
	fileOidByPath    map[string]string
	fileStatusByPath map[string]string
	patchHashByPath  map[string]string

	// localFiles is the full changed-file set (unfiltered). The tree
	// (syntheticFiles) is rebuilt from it, optionally hiding generated files.
	localFiles           []git_commands.ReviewChangedFile
	showGenerated        bool // when false, generated files (lockfiles, *.generated.*) are hidden
	hiddenGeneratedCount int  // number of generated files currently hidden from the tree
}

// reviewRefRetentionDays is the prune-on-boot retention window (D1.3): review refs
// whose commit is older than this are deleted when a new review session fetches.
const reviewRefRetentionDays = 30

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
		lines := presentation.RenderPrReviewFileTree(viewModel, self.reviewed, self.unresolvedThreadPaths(), showFileIcons, &c.UserConfig().Gui.CustomIcons)
		return lo.Map(lines, func(line string, _ int) []string {
			return []string{line}
		})
	}

	baseContext := NewBaseContext(NewBaseContextOpts{
		Kind:       types.SIDE_CONTEXT,
		View:       c.Views().PrReview,
		WindowName: "prContent",
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

// MarkdownRenderer exposes the glow-backed markdown renderer to sibling review
// contexts (e.g. the Conversation tab). May be nil, in which case bodies render plain.
func (self *PrReviewContext) MarkdownRenderer() func(body string, width int) string {
	return self.renderMarkdown
}

// startLoad kicks off the combined review-data + changed-files fetch off the UI
// thread once per session, then hops back to rebuild the tree and render (D3 /
// task 3.6).
func (self *PrReviewContext) startLoad() {
	if self.loadStarted || self.target.owner == "" {
		return
	}
	self.loadStarted = true

	useLocalRefs := self.localRefModeEnabled()

	// Test seam: when LAZYGIT_PR_REVIEW_FIXTURE points at a JSON file, load the PR
	// data (and, in legacy API mode, the changed files) from it synchronously
	// instead of hitting gh/GitHub. In local-ref mode the changed-file tree and diff
	// come from pre-fetched local refs/lazygit-review/<pr>/* (created by the test),
	// so we build the tree from `git diff` against those refs synchronously here.
	if path := os.Getenv("LAZYGIT_PR_REVIEW_FIXTURE"); path != "" {
		if err := self.loadFixture(path); err != nil {
			self.loadErr = err
		} else if useLocalRefs {
			// Refs already exist in the test repo; never fetch over the network.
			mergeBase, files, err := self.loadLocalRefModel(self.target.number, nil)
			if err != nil {
				self.loadErr = err
			} else {
				self.applyLocalRefModel(mergeBase, files)
			}
		}
		self.rerender()
		self.finishReload()
		return
	}

	target := self.target
	store := git_commands.NewReviewSnapshotStore(config.ConfigDir(), target.owner, target.repo, target.number)
	bypassCache := self.bypassCacheOnce
	self.bypassCacheOnce = false
	generation := self.loadGeneration

	self.c.OnWorker(func(_ gocui.Task) error {
		bootStart := time.Now()

		// Warm path (D3.1): render the whole workspace from the on-disk snapshots
		// immediately when they are valid for the refs we already hold locally, then
		// revalidate in the background. `R` (refresh) sets bypassCache so a manual
		// refresh always hits GitHub.
		warmApplied := false
		if useLocalRefs && !bypassCache {
			warmApplied = self.applyCachedSnapshots(store, target)
		}
		if warmApplied {
			self.c.Log.Infof("pr-review boot: warm render from cache in %s (%s)", time.Since(bootStart), store.Dir())
			// Test/airplane seam: with a valid warm cache, skip the background
			// revalidation entirely so tests can prove the render path needs no
			// network.
			if os.Getenv("LAZYGIT_PR_REVIEW_OFFLINE") != "" {
				self.c.OnUIThread(func() error {
					self.finishReload()
					return nil
				})
				return nil
			}
		}

		// Host is github.com in v1 (gh-dash launches against github.com).
		token := self.c.Git().GitHub.GetAuthToken("github.com")

		graphqlStart := time.Now()
		data, dataErr := self.c.Git().GitHub.FetchPRReviewData(target.owner, target.repo, target.number, token)
		graphqlDur := time.Since(graphqlStart)
		var files []*models.GithubPullRequestFile
		var filesErr error
		var localBase string
		var localFiles []git_commands.ReviewChangedFile
		var localErr error
		refsStart := time.Now()
		if dataErr == nil {
			if useLocalRefs {
				// Production local-ref path: fetch the PR's base+head into the
				// isolated namespace (skipped per side when the refs already point
				// at the wanted OIDs), verify the merge-base, list the changed files.
				// Everything the worker needs travels via captured locals — the
				// context's fields are only ever written on the UI thread.
				localBase, localFiles, localErr = self.loadLocalRefModel(target.number, data)
			} else {
				files, filesErr = self.c.Git().GitHub.FetchPRChangedFiles(target.owner, target.repo, target.number, token)
			}
		}
		self.c.Log.Infof("pr-review boot: graphql=%s refs+diff=%s total=%s (warm=%v)",
			graphqlDur, time.Since(refsStart), time.Since(bootStart), warmApplied)

		// Persist fresh snapshots (only on full success, so the cache never holds a
		// half-fetched state). Failures are logged, never surfaced — the cache is an
		// accelerator, not a dependency.
		if useLocalRefs && dataErr == nil && localErr == nil {
			if err := store.Write("meta", data.HeadRefOid, data); err != nil {
				self.c.Log.Warnf("pr-review cache: writing meta snapshot: %v", err)
			}
			if err := store.Write("files", data.HeadRefOid, reviewFilesSnapshot{BaseRefOid: data.BaseRefOid, MergeBase: localBase, Files: localFiles}); err != nil {
				self.c.Log.Warnf("pr-review cache: writing files snapshot: %v", err)
			}
		}

		self.c.OnUIThread(func() error {
			if generation != self.loadGeneration {
				// A Retarget superseded this load; dropping it keeps the new PR's
				// state from being overwritten by the old PR's result.
				return nil
			}
			loadFailed := dataErr != nil || localErr != nil || filesErr != nil
			if warmApplied && loadFailed {
				// The warm render already gave the user a working workspace; a
				// failed background revalidation (offline, rate-limited) must not
				// replace it with an error screen — surface it as a toast instead.
				firstErr := dataErr
				if firstErr == nil {
					firstErr = localErr
				}
				if firstErr == nil {
					firstErr = filesErr
				}
				self.c.ErrorToast(fmt.Sprintf(self.c.Tr.PrReviewRevalidateFailed, firstErr.Error()))
				self.finishReload()
				return nil
			}
			switch {
			case dataErr != nil:
				self.loadErr = dataErr
			case localErr != nil:
				self.loadErr = localErr
			case filesErr != nil:
				self.loadErr = filesErr
			default:
				// Clear any prior error so a successful Reload stops rendering a
				// stale failure message.
				self.loadErr = nil
				self.data = data
				if useLocalRefs {
					self.applyLocalRefModel(localBase, localFiles)
				} else {
					self.setFiles(files)
				}
			}
			self.rerender()
			self.finishReload()
			return nil
		})
		return nil
	})
}

// reviewFilesSnapshot is the payload of the "files" cache snapshot: the resolved
// merge-base plus the changed-file set with blob OIDs — everything applyLocalRefModel
// needs to rebuild the tree without git or network work. BaseRefOid pairs the
// snapshot with the base it was diffed against: the envelope's headRefOid alone can't
// catch a base branch that moved while the head stayed put (codex review finding).
type reviewFilesSnapshot struct {
	BaseRefOid string
	MergeBase  string
	Files      []git_commands.ReviewChangedFile
}

// applyCachedSnapshots renders the workspace from the on-disk snapshots when they are
// coherent: meta and files captured at the same head OID, and the local review refs
// still pointing at the snapshot's OIDs (so diffs/commits render from the object
// store). Returns false on any mismatch — a cold boot, never an error. Runs on the
// worker; the apply hops to the UI thread.
func (self *PrReviewContext) applyCachedSnapshots(store *git_commands.ReviewSnapshotStore, target prReviewTarget) bool {
	var cachedData git_commands.PullRequestReviewData
	metaOid, _, ok := store.Read("meta", &cachedData)
	if !ok || metaOid == "" || metaOid != cachedData.HeadRefOid {
		return false
	}
	var cachedFiles reviewFilesSnapshot
	filesOid, _, ok := store.Read("files", &cachedFiles)
	if !ok || filesOid != metaOid || cachedFiles.BaseRefOid != cachedData.BaseRefOid {
		return false
	}

	gh := self.c.Git().GitHub
	if gh.ReviewRefOid(git_commands.ReviewHeadRef(target.number)) != cachedData.HeadRefOid ||
		gh.ReviewRefOid(git_commands.ReviewBaseRef(target.number)) != cachedData.BaseRefOid {
		return false
	}

	generation := self.loadGeneration
	self.c.OnUIThread(func() error {
		if generation != self.loadGeneration {
			return nil
		}
		self.loadErr = nil
		self.data = &cachedData
		self.applyLocalRefModel(cachedFiles.MergeBase, cachedFiles.Files)
		self.rerender()
		return nil
	})
	return true
}

// localRefModeEnabled reports whether the changed-file tree and diff should be
// sourced from the local-ref spine (DW2 / Phase 2) rather than the legacy gh-API
// patch path. Production review sessions always use local refs; the legacy
// JSON-fixture-only integration tests opt out (they have no local refs) unless they
// explicitly set LAZYGIT_PR_REVIEW_LOCAL_REFS, which the local-ref test uses to
// drive the git-diff path against refs it created.
func (self *PrReviewContext) localRefModeEnabled() bool {
	if os.Getenv("LAZYGIT_PR_REVIEW_LOCAL_REFS") != "" {
		return true
	}
	// A pure JSON fixture (no local refs) means a legacy API-path test.
	return os.Getenv("LAZYGIT_PR_REVIEW_FIXTURE") == ""
}

// loadLocalRefModel performs the git side of the local-ref read model: optionally
// fetch the PR's base+head refs (skipped in tests where the refs already exist),
// then resolve the merge-base and list the changed files between it and the head.
// It runs blocking git commands and so must be called off the UI thread (or
// synchronously from a test).
func (self *PrReviewContext) loadLocalRefModel(pr int, fetchFor *git_commands.PullRequestReviewData) (string, []git_commands.ReviewChangedFile, error) {
	gh := self.c.Git().GitHub

	if fetchFor != nil {
		// Prune review refs older than the retention window before fetching new ones
		// (D1.3). Best-effort: a prune failure must never block the review session.
		if err := gh.PruneStaleReviewRefs(time.Now().Unix(), reviewRefRetentionDays); err != nil {
			self.c.Log.Errorf("pruning stale review refs: %v", err)
		}
		if err := gh.FetchReviewRefs(pr, fetchFor.HeadRepoURL, fetchFor.HeadRefOid, fetchFor.BaseRepoURL, fetchFor.BaseRefOid); err != nil {
			return "", nil, err
		}
	}

	mergeBase, err := gh.ReviewMergeBase(pr)
	if err != nil {
		return "", nil, err
	}

	headRef := git_commands.ReviewHeadRef(pr)
	files, err := gh.ReviewChangedFiles(mergeBase, headRef)
	if err != nil {
		return "", nil, err
	}

	// Resolve each file's content identity (head blob OID) for the persisted
	// viewed-state in batch (two git calls total, regardless of file count) — so the
	// tree is NEVER capped for the file count; every changed file is shown, matching
	// lazygit's own uncapped file panels. A deleted file has no head blob, so it is
	// keyed off the base blob. The rare file with no resolvable blob falls back to a
	// diff hash.
	var headPaths, basePaths []string
	for _, f := range files {
		if f.Status == "D" {
			basePaths = append(basePaths, f.Path)
		} else {
			headPaths = append(headPaths, f.Path)
		}
	}
	headOids := gh.ReviewFileBlobOids(headRef, headPaths)
	baseOids := gh.ReviewFileBlobOids(mergeBase, basePaths)
	for i := range files {
		if files[i].Status == "D" {
			files[i].HeadOid = baseOids[files[i].Path]
		} else {
			files[i].HeadOid = headOids[files[i].Path]
		}
		if files[i].HeadOid == "" {
			diff, _ := self.c.Git().WorkingTree.ShowFileDiff(mergeBase, headRef, false, files[i].Path, true)
			files[i].PatchHash = git_commands.HashReviewPatch(diff)
		}
	}
	return mergeBase, files, nil
}

// applyLocalRefModel installs the local-ref changed-file set as the tree's backing
// data (every changed file — the file list is not capped) and records the diff
// endpoints the controller renders through the pager. Must run on the UI thread.
func (self *PrReviewContext) applyLocalRefModel(mergeBase string, files []git_commands.ReviewChangedFile) {
	self.localRefsActive = true
	self.localBase = mergeBase
	self.localHeadRef = git_commands.ReviewHeadRef(self.target.number)

	self.localFiles = files
	self.fileByPath = map[string]*models.GithubPullRequestFile{}
	self.fileOidByPath = make(map[string]string, len(files))
	self.fileStatusByPath = make(map[string]string, len(files))
	self.patchHashByPath = make(map[string]string, len(files))
	for _, f := range files {
		self.fileOidByPath[f.Path] = f.HeadOid
		self.fileStatusByPath[f.Path] = f.Status
		self.patchHashByPath[f.Path] = f.PatchHash
	}
	self.files = nil
	self.pruneClosedPrReviewedState()
	self.applyPersistedReviewedState()
	self.rebuildLocalTree()
	// Now that the refs/data are resolved, (re)load the scoped Activity tabs so they
	// are ready and never stale — on boot and after every refresh.
	self.refreshScopedActivityTabs()
}

// rebuildLocalTree (re)builds the tree from the full local changed-file set, hiding
// generated files (lockfiles, *.generated.*, etc.) unless the user has toggled them on.
func (self *PrReviewContext) rebuildLocalTree() {
	self.hiddenGeneratedCount = 0
	self.syntheticFiles = make([]*models.CommitFile, 0, len(self.localFiles))
	for _, f := range self.localFiles {
		if !self.showGenerated && isGeneratedReviewFile(f.Path) {
			self.hiddenGeneratedCount++
			continue
		}
		self.syntheticFiles = append(self.syntheticFiles, &models.CommitFile{
			Path:         f.Path,
			ChangeStatus: f.Status,
		})
	}
	self.SetTree()
}

// ToggleShowGenerated flips whether generated files are shown in the tree and rebuilds
// it. Returns the new state so the controller can toast it.
func (self *PrReviewContext) ToggleShowGenerated() bool {
	self.showGenerated = !self.showGenerated
	self.rebuildLocalTree()
	self.rerender()
	return self.showGenerated
}

// HiddenGeneratedCount is the number of generated files currently hidden from the tree.
func (self *PrReviewContext) HiddenGeneratedCount() int {
	return self.hiddenGeneratedCount
}

// isGeneratedReviewFile reports whether a path looks machine-generated (dependency
// lockfiles, protobuf/codegen output, snapshots, minified assets) — files a reviewer
// usually skips. It matches on the base name so nested paths work.
func isGeneratedReviewFile(p string) bool {
	base := p
	if i := strings.LastIndexByte(p, '/'); i != -1 {
		base = p[i+1:]
	}
	switch base {
	case "yarn.lock", "package-lock.json", "pnpm-lock.yaml", "npm-shrinkwrap.json",
		"go.sum", "Cargo.lock", "Gemfile.lock", "poetry.lock", "composer.lock", "Pipfile.lock":
		return true
	}
	lower := strings.ToLower(base)
	switch {
	case strings.Contains(lower, ".generated."):
		return true
	case strings.HasSuffix(lower, "_generated.go"), strings.HasSuffix(lower, ".pb.go"), strings.HasSuffix(lower, ".gen.go"):
		return true
	case strings.HasSuffix(lower, ".snap"):
		return true
	case strings.HasSuffix(lower, ".min.js"), strings.HasSuffix(lower, ".min.css"):
		return true
	default:
		return false
	}
}

// refreshScopedActivityTabs tells the PR Commits and Conversation tabs to (re)load
// from the now-resolved refs/data. Done by key lookup so this context doesn't depend on
// the PrCommits/PrConversation types beyond their reload methods.
func (self *PrReviewContext) refreshScopedActivityTabs() {
	if ctx, ok := self.c.ContextForKey(PR_COMMITS_CONTEXT_KEY).(*PrCommitsContext); ok {
		ctx.Reload()
	}
	if ctx, ok := self.c.ContextForKey(PR_CONVERSATION_CONTEXT_KEY).(*PrConversationContext); ok {
		ctx.Reload()
	}
	if ctx, ok := self.c.ContextForKey(PR_CHECKS_CONTEXT_KEY).(*PrChecksContext); ok {
		ctx.Reload()
	}
	// The PR list's launched row is created before the title is known; refresh it
	// now that the data has arrived.
	if ctx, ok := self.c.ContextForKey(PR_LIST_CONTEXT_KEY).(*PrListContext); ok {
		ctx.RefreshLaunchedTitle()
	}
}

// repoKey is the runtime-resolved owner/repo used to key persisted viewed state.
func (self *PrReviewContext) repoKey() string {
	return self.target.owner + "/" + self.target.repo
}

// applyPersistedReviewedState rebuilds the in-memory reviewed set from AppState: a
// file is viewed only if a stored mark exists whose content identity still matches the
// current head blob OID (D1.4). A stored entry whose content no longer matches is
// auto-cleared and pruned from the state file, so a new commit that changes a file
// drops its "viewed" mark. Runs on the UI thread.
func (self *PrReviewContext) applyPersistedReviewedState() {
	self.reviewed = map[string]bool{}
	appState := self.c.GetAppState()
	if len(appState.ReviewedFiles) == 0 {
		return
	}

	repo := self.repoKey()
	changed := false
	for path, oid := range self.fileOidByPath {
		key := config.ReviewedFileKey(repo, self.target.number, path)
		entry, ok := appState.ReviewedFiles[key]
		if !ok {
			continue
		}
		if entry.MatchesContent(oid, self.patchHashByPath[path]) {
			self.reviewed[path] = true
		} else {
			delete(appState.ReviewedFiles, key)
			changed = true
		}
	}
	if changed {
		self.c.SaveAppStateAndLogError()
	}
}

// pruneClosedPrReviewedState drops persisted viewed marks for this PR once it is
// merged or closed (task 4.3); the review is done, so the marks are no longer useful.
func (self *PrReviewContext) pruneClosedPrReviewedState() {
	if self.data == nil || (self.data.State != "MERGED" && self.data.State != "CLOSED") {
		return
	}
	appState := self.c.GetAppState()
	if len(appState.ReviewedFiles) == 0 {
		return
	}
	repo := self.repoKey()
	changed := false
	for key, entry := range appState.ReviewedFiles {
		if entry.Repo == repo && entry.PR == self.target.number {
			delete(appState.ReviewedFiles, key)
			changed = true
		}
	}
	if changed {
		self.c.SaveAppStateAndLogError()
	}
}

// persistReviewed writes or clears the persisted "viewed" mark for a file, keyed by
// its current head blob OID so it auto-invalidates when the file later changes.
func (self *PrReviewContext) persistReviewed(path string, viewed bool) {
	appState := self.c.GetAppState()
	if appState.ReviewedFiles == nil {
		appState.ReviewedFiles = map[string]config.ReviewedFileState{}
	}
	key := config.ReviewedFileKey(self.repoKey(), self.target.number, path)
	if !viewed {
		delete(appState.ReviewedFiles, key)
	} else {
		appState.ReviewedFiles[key] = config.ReviewedFileState{
			Repo:      self.repoKey(),
			PR:        self.target.number,
			Path:      path,
			Status:    self.fileStatusByPath[path],
			FileOid:   self.fileOidByPath[path],
			PatchHash: self.patchHashByPath[path],
			ViewedAt:  time.Now().Unix(),
		}
	}
	self.c.SaveAppStateAndLogError()
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
	// Keep the Activity tabs (Conversation especially) in sync on the legacy gh-API
	// path too, so they aren't left stale if focused before the data arrived.
	self.refreshScopedActivityTabs()
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

// selectedFilePath returns the path of the file the cursor is on, or "" on a
// directory node or before data loads. Unlike selectedPRFile it does not depend on
// the gh-API file map (empty in local-ref mode), so it works in both modes.
func (self *PrReviewContext) selectedFilePath() string {
	node := self.GetSelected()
	if node == nil || node.File == nil {
		return ""
	}
	return node.File.Path
}

// SelectedIsFile reports whether the cursor is on a file (not a directory) node, so
// the controller can decide whether Enter opens the diff surface or collapses a folder.
func (self *PrReviewContext) SelectedIsFile() bool {
	node := self.GetSelected()
	return node != nil && node.IsFile()
}

// localFilePatch returns the plain (no-color) unified diff of one path between the
// PR's merge-base and head refs, for the focusable selection surface. Empty on error.
func (self *PrReviewContext) localFilePatch(path string) string {
	out, err := self.c.Git().WorkingTree.ShowFileDiff(self.localBase, self.localHeadRef, false, path, true)
	if err != nil {
		self.c.Log.Errorf("rendering local review diff for %s: %v", path, err)
		return ""
	}
	return out
}

// SelectedFileDiff renders the cursor file's diff for the focusable selection surface
// and returns the rendered row-map, the file path, and the raw unified patch. The
// patch is sourced from a plain local `git diff` in local-ref mode and from the gh-API
// patch otherwise; either way the same presenter (and the same patch string the caller
// maps line numbers from) is used, so RowPatchIdx stays aligned. Returns (nil,"","")
// when the cursor is not on a file or there is no data.
func (self *PrReviewContext) SelectedFileDiff(width int) (*presentation.RenderedReviewDiff, string, string) {
	if self.data == nil {
		return nil, "", ""
	}

	var path string
	var patchStr string
	if self.localRefsActive {
		if !self.SelectedIsFile() {
			return nil, "", ""
		}
		path = self.selectedFilePath()
		patchStr = self.localFilePatch(path)
	} else {
		file := self.selectedPRFile()
		if file == nil {
			return nil, "", ""
		}
		path = file.Filename
		patchStr = file.Patch
	}

	rendered := presentation.RenderReviewFileDiff(presentation.ReviewFileDiffOpts{
		Tr:             self.c.Tr,
		Path:           path,
		Diff:           patchStr,
		Threads:        self.data.Threads,
		Width:          width,
		RenderMarkdown: self.renderMarkdown,
		Reviewed:       self.reviewed[path],
		DiffMode:       self.diffMode,
	})
	return rendered, path, patchStr
}

// unresolvedThreadPaths is the set of changed-file paths that carry at least one
// unresolved review thread, for the tree's 💬 marker.
func (self *PrReviewContext) unresolvedThreadPaths() map[string]bool {
	result := map[string]bool{}
	if self.data == nil {
		return result
	}
	for i := range self.data.Threads {
		t := &self.data.Threads[i]
		if !t.IsResolved && t.Path != "" {
			result[t.Path] = true
		}
	}
	return result
}

// SelectedFileHasThreads reports whether the file under the cursor has any review
// thread (any author, resolved or not). The hover preview uses this to decide between
// the pager (clean delta, no comments) and the inline presenter (comments interleaved).
func (self *PrReviewContext) SelectedFileHasThreads() bool {
	if self.data == nil {
		return false
	}
	path := self.selectedFilePath()
	if path == "" {
		return false
	}
	for i := range self.data.Threads {
		if self.data.Threads[i].Path == path {
			return true
		}
	}
	return false
}

// RenderSelectedFileInlineDiff renders the cursor file's diff with EVERY reviewer's
// threads interleaved (not just the current user's), for the hover preview of a file
// that has comments. Works in both local-ref and legacy modes. Empty when the cursor
// is not on a file.
func (self *PrReviewContext) RenderSelectedFileInlineDiff(width int) string {
	rendered, _, _ := self.SelectedFileDiff(width)
	if rendered == nil {
		return ""
	}
	return rendered.Content
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

// ToggleReviewedForSelectedFile flips the viewed flag for the cursor selection
// (design D8) and re-renders so the tree colors and diff header update. On a FILE it
// toggles that file; on a FOLDER it toggles every changed file underneath as a group —
// if they are all already viewed the folder is unviewed, otherwise the whole folder is
// marked viewed (so one press clears a partially-viewed folder to fully viewed).
func (self *PrReviewContext) ToggleReviewedForSelectedFile() {
	node := self.GetSelected()
	if node == nil {
		return
	}
	if self.reviewed == nil {
		self.reviewed = map[string]bool{}
	}

	// ForEachFile yields the single file for a file node, or every descendant file
	// for a folder node — so both cases share one path.
	var paths []string
	_ = node.ForEachFile(func(f *models.CommitFile) error {
		paths = append(paths, f.Path)
		return nil
	})
	if len(paths) == 0 {
		return
	}

	allViewed := true
	for _, p := range paths {
		if !self.reviewed[p] {
			allViewed = false
			break
		}
	}
	nowViewed := !allViewed

	for _, p := range paths {
		self.reviewed[p] = nowViewed
		// In local-ref mode the mark is persisted to AppState (keyed by blob OID) so it
		// survives restarts and auto-clears when the file changes. The legacy gh-API
		// path keeps its session-only in-memory mark.
		if self.localRefsActive {
			self.persistReviewed(p, nowViewed)
		}
	}
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

// LocalRefsActive reports whether the Files-Changed tab is sourced from the local-ref
// spine (so the controller renders the selected file/folder diff through the pager).
func (self *PrReviewContext) LocalRefsActive() bool {
	return self.localRefsActive
}

// LocalDiffEndpoints returns the (from, to) revisions for the PR diff: the merge-base
// and the head ref. `git diff <mergeBase> <head>` reproduces GitHub's three-dot PR
// diff semantics.
func (self *PrReviewContext) LocalDiffEndpoints() (from string, to string) {
	return self.localBase, self.localHeadRef
}

// LocalCommitRange returns the base and head REFS (not the merge-base) for the PR's
// commit list: `git log <baseRef>..<headRef>` lists exactly the commits on head that
// are not on the base branch — GitHub's "Commits" tab semantics — which differs from
// the diff's merge-base when the PR has merged the base branch into itself. ok is
// false until the local refs are active.
func (self *PrReviewContext) LocalCommitRange() (base string, head string, ok bool) {
	if !self.localRefsActive {
		return "", "", false
	}
	return git_commands.ReviewBaseRef(self.target.number), git_commands.ReviewHeadRef(self.target.number), true
}

// IsPathViewed reports whether the given file path is currently marked viewed. Used by
// the controller to split a folder's aggregate diff into unviewed vs viewed panes,
// mirroring lazygit's unstaged/staged split.
func (self *PrReviewContext) IsPathViewed(path string) bool {
	return self.reviewed[path]
}

// Retarget points the whole review workspace at a different pull request (selected
// from the PR list window) and reloads: all per-PR state — data, tree, viewed marks,
// pending review comments, local-ref bookkeeping — is reset so nothing leaks across
// PRs, then the standard load path (warm cache included) runs for the new target.
func (self *PrReviewContext) Retarget(owner string, repo string, number int) {
	self.loadGeneration++
	self.target = prReviewTarget{owner: owner, repo: repo, number: number}
	self.data = nil
	self.files = nil
	self.loadErr = nil
	self.localRefsActive = false
	self.localBase = ""
	self.localHeadRef = ""
	self.localFiles = nil
	self.syntheticFiles = nil
	self.fileByPath = map[string]*models.GithubPullRequestFile{}
	self.fileOidByPath = map[string]string{}
	self.fileStatusByPath = map[string]string{}
	self.patchHashByPath = map[string]string{}
	self.reviewed = map[string]bool{}
	self.pendingComments = nil
	self.hiddenGeneratedCount = 0
	self.showGenerated = false
	self.SetTree()
	self.refreshScopedActivityTabs()
	self.loadStarted = false
	self.startLoad()
	self.rerender()
}

// AddPendingComment queues a comment for the grouped review submission and returns
// the new pending count.
func (self *PrReviewContext) AddPendingComment(comment git_commands.PendingReviewComment) int {
	self.pendingComments = append(self.pendingComments, comment)
	return len(self.pendingComments)
}

// PendingComments returns the accumulated (not yet submitted) review comments.
func (self *PrReviewContext) PendingComments() []git_commands.PendingReviewComment {
	return self.pendingComments
}

// ClearPendingComments empties the pending queue (after a successful submission).
func (self *PrReviewContext) ClearPendingComments() {
	self.pendingComments = nil
}

// Reload re-fetches the review data (e.g. after a write) so new comments appear.
// A reload is an explicit "give me fresh data" action, so it bypasses the warm cache.
func (self *PrReviewContext) Reload() {
	self.loadStarted = false
	self.bypassCacheOnce = true
	self.startLoad()
}

// ReloadThen re-fetches the review data and invokes onDone once on the UI thread
// after the new data is applied. The diff surface uses it to re-render and surface a
// just-posted comment (the data alone reloading doesn't repaint the focused view).
func (self *PrReviewContext) ReloadThen(onDone func()) {
	self.reloadDone = onDone
	self.loadStarted = false
	self.bypassCacheOnce = true
	self.startLoad()
}

// finishReload fires (and clears) the pending ReloadThen callback. Called after each
// load completes; a no-op unless a ReloadThen is pending, so it never fires on the
// initial session load.
func (self *PrReviewContext) finishReload() {
	if self.reloadDone == nil {
		return
	}
	cb := self.reloadDone
	self.reloadDone = nil
	cb()
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
