**Findings**

P0: DW1 is not just `SideWindowNames()` + `viewTabMap()`. `SideWindowNames` only accepts `UserConfig`, but review mode is session state, and this function is used by layout, focus redirection, and panel numbering: [/home/user/projects/lazygit/pkg/gui/controllers/helpers/window_helper.go:145](/home/user/projects/lazygit/pkg/gui/controllers/helpers/window_helper.go:145), [/home/user/projects/lazygit/pkg/gui/context_config.go:18](/home/user/projects/lazygit/pkg/gui/context_config.go:18), [/home/user/projects/lazygit/pkg/gui/views.go:231](/home/user/projects/lazygit/pkg/gui/views.go:231). If you branch the free function without threading review state through these call sites, focus can still redirect hidden review contexts back to `Files`. Cleanest flag location is `GuiRepoState` or `StartArgs` copied into `GuiRepoState`, then expose a `gui.sideWindowNames()` method and add `IsReviewMode` to `WindowArrangementArgs`.

P0: DW2 overstates “reuse staging/patch-explorer.” The read-only pager preview is realistic; direct staging reuse is not. `StagingHelper.RefreshStagingPanel` reads `Contexts().Files.GetSelected()` and calls `WorkingTree.WorktreeFileDiff` against index/worktree state: [/home/user/projects/lazygit/pkg/gui/controllers/helpers/staging_helper.go:42](/home/user/projects/lazygit/pkg/gui/controllers/helpers/staging_helper.go:42). `StagingController.ToggleStaged` applies patches to the index: [/home/user/projects/lazygit/pkg/gui/controllers/staging_controller.go:204](/home/user/projects/lazygit/pkg/gui/controllers/staging_controller.go:204). The existing PR review code already created a custom row-map diff context because interleaved comments break `patch_exploring.State`: [/home/user/projects/lazygit/pkg/gui/context/pr_review_diff_context.go:10](/home/user/projects/lazygit/pkg/gui/context/pr_review_diff_context.go:10), [/home/user/projects/lazygit/pkg/gui/presentation/pr_review_diff.go:14](/home/user/projects/lazygit/pkg/gui/presentation/pr_review_diff.go:14). Keep the interaction model; do not reuse the staging controller.

P1: Local refs are acceptable for read-only preview, but only with a narrower claim. `ShowFileDiffCmdObj(from, to, paths...)` can render fetched refs through the pager: [/home/user/projects/lazygit/pkg/commands/git_commands/working_tree.go:439](/home/user/projects/lazygit/pkg/commands/git_commands/working_tree.go:439), and the pty path will engage the configured pager: [/home/user/projects/lazygit/pkg/gui/pty.go:46](/home/user/projects/lazygit/pkg/gui/pty.go:46). But there is no existing command for fetching arbitrary PR refspecs into `refs/lazygit-review/*`; current fetch wrappers are generic remote fetches: [/home/user/projects/lazygit/pkg/commands/git_commands/sync.go:57](/home/user/projects/lazygit/pkg/commands/git_commands/sync.go:57). Add an explicit review-fetch command with `--no-write-fetch-head`, no checkout, no branch namespace writes, and no reliance on index/worktree state. A separate worktree is safer only if you intend to mutate/index-stage; for read-only diffs it is extra complexity.

P1: Local refs do not solve GitHub comment anchoring by themselves. Pager output is not line-addressable, and `ShowFileDiffCmdObj` passes `--no-renames`: [/home/user/projects/lazygit/pkg/commands/git_commands/working_tree.go:451](/home/user/projects/lazygit/pkg/commands/git_commands/working_tree.go:451). GitHub comments need `path`, `side`, `line`, `start_line`, and live `headRefOid`, as the current write path shows: [/home/user/projects/lazygit/pkg/commands/git_commands/github_review.go:97](/home/user/projects/lazygit/pkg/commands/git_commands/github_review.go:97). You still need a plain, parsed diff source parallel to the pager preview. Treat delta as display only.

P1: The PR commits panel is not “nearly free.” `LocalCommitsContext` is hardwired to `Model().Commits`, checked-out branch rendering, branch metadata, rebase state, and global refresh: [/home/user/projects/lazygit/pkg/gui/context/local_commits_context.go:28](/home/user/projects/lazygit/pkg/gui/context/local_commits_context.go:28), [/home/user/projects/lazygit/pkg/gui/controllers/helpers/refresh_helper.go:351](/home/user/projects/lazygit/pkg/gui/controllers/helpers/refresh_helper.go:351). `CommitLoader` can probably load `base..head` via `RefName`, but the UI should use a separate PR commits model/context or it will clobber normal commits state and inherit irrelevant branch status.

P1: Panel numbering and tabs are hardcoded for the normal window set. `configureViewProperties` maps prefixes only for `status/files/branches/commits/stash`: [/home/user/projects/lazygit/pkg/gui/views.go:231](/home/user/projects/lazygit/pkg/gui/views.go:231). New windows `prList/prContent/prActivity` need explicit views, title prefixes, context window names, and tab map entries. Existing `PrReviewContext` still lives in the `"files"` window: [/home/user/projects/lazygit/pkg/gui/context/pr_review_context.go:111](/home/user/projects/lazygit/pkg/gui/context/pr_review_context.go:111).

P2: The spec requires grouped reviews, checks tree, gh-dash tabs, overview, persisted viewed state, and PR commits, but there is no `tasks.md` in this change. The design refers to phasing, yet the only delta spec is all-or-nothing. That underestimates the feature by a lot.

**What Is Underestimated**

The 3x items are layout-state threading, focus/default-context behavior, PR commits isolation, check-run/job normalization, grouped review lifecycle, and line mapping for local diff versus GitHub anchors. The design also underestimates gh-dash config parsing and absent/malformed config handling.

**What I’d Cut**

Cut full gh-dash PR browser in v1, checks logs/tree, grouped review submission, focus split inversion, and side-by-side diff. Keep single launched PR, files tree, pager preview, custom selectable comment surface, and persisted viewed state.

**Recommended Phasing**

1. Layout foundation only: add review-mode side-window plumbing, panel labels, initial focus, and tab cycling with stub contexts.
2. Local-ref read model: fetch base/head refs, render file and folder diffs through pager, no commenting yet.
3. Parsed-diff selection surface: custom row map over the same local diff, with GitHub line mapping tests.
4. Persist viewed state keyed by repo + PR + path + blob/head content identity.
5. PR commits as a separate scoped commits context/model.
6. Conversation display.
7. Comment submission, first standalone, then grouped review.
8. gh-dash multi-section PR list and checks tree last.
