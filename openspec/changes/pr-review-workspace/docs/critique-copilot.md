The design is plausible only if DW1/DW2 are treated as foundation rewrites, not seams that make the feature “mostly free.” The two biggest risks are: **the staging machinery is not reusable read-only as written**, and **the layout swap leaks normal `Files` assumptions through focus/default-window paths.**

## P0 blockers

1. **DW2 overclaims reuse of staging/patch-explorer: the actual staging controller mutates the index/worktree.**  
   `pkg/gui/controllers/staging_controller.go:204-210` binds select to `ToggleStaged`, and `pkg/gui/controllers/staging_controller.go:246-269` transforms the selected patch and calls `Git().Patch.ApplyPatch(...)`. It also derives the target path from the normal files context at `pkg/gui/controllers/staging_controller.go:351-352`. `pkg/gui/controllers/helpers/staging_helper.go:45-57` rebuilds staging diffs from `Contexts().Files.GetSelected()` and `WorkingTree.WorktreeFileDiff(...)`.  
   **Impact:** “enter-to-focus line selection reuses patch-explorer/staging machinery” (`openspec/changes/pr-review-workspace/design.md:104-106`) will either stage/unstage user files or require a separate read-only controller/context. Reuse the `patch_exploring.State` selection model only after extracting it behind a non-mutating interface; do not reuse `StagingController`.

2. **DW1 cannot be just `SideWindowNames()` + `viewTabMap()` without fixing default/fallback focus.**  
   `SideWindowNames` is a free function that only accepts `UserConfig` (`pkg/gui/controllers/helpers/window_helper.go:145-162`), but review mode is session state. Worse, default focus paths still hard-code Files: `layout.go:17`, `context_config.go:27-31`, `context_config.go:34-42`, `context.go:254`, and `context.go:281`. If normal windows are suppressed, these paths can push or restore a hidden `Files` context. `WindowHelper.GetViewNameForWindow` also panics for an unmapped window (`pkg/gui/controllers/helpers/window_helper.go:31-35`).  
   **Impact:** branching only the layout list will misfocus or panic during startup, config reload, Escape/pop fallback, and panel jumps. The clean flag belongs in `GuiRepoState`/`StartArgs`-derived repo state, then must be threaded into `WindowArrangementArgs`, `WindowHelper.SideWindows`, `isSideWindowVisible`, and `defaultSideContext`.

3. **DW2 must validate the checkout is actually `OWNER/REPO`, not merely “some git worktree.”**  
   The current guard only checks `git rev-parse --is-inside-work-tree` (`pkg/app/app.go:184-199`). DW2 plans to fetch and diff inside the existing checkout (`design.md:94-101`).  
   **Impact:** a gh-dash misconfiguration can write review refs into the wrong repository and render/post comments against a PR whose files do not match the local repo. Before any fetch, verify remotes match the target host/owner/repo or require an explicit remote/URL from `gh`.

## P1 important

4. **Local-ref fetch is under-specified for forks, moved bases, and shallow history.**  
   The current review data has `headRefOid` but no base OID, base repo, or head repo URL (`pkg/commands/git_commands/github_review.go:34-52`, `:161-170`). PR heads often live in forks; fetching `origin` is not enough. Three-dot diff also needs enough history to compute the merge-base, so aggressive shallow/filter fetches can silently produce wrong diffs.  
   **Recommendation:** fetch by explicit repository URL + OID/refspec for both base and head, store fetched OIDs, and verify `git merge-base` succeeds before rendering.

5. **Review refs will pollute normal lazygit history unless excluded.**  
   `git log --all` is supported by the normal commits loader (`pkg/commands/git_commands/commit_loader.go:589-603`), so `refs/lazygit-review/*` will appear in normal all-graph views unless hidden/excluded. Fetch also writes `FETCH_HEAD`.  
   **Recommendation:** either use a separate worktree/bare cache, or consistently exclude `refs/lazygit-review/*` from normal loaders and prune stale refs on boot.

6. **The normal diff helper won’t automatically target PR files/folders.**  
   `DiffHelper.DiffArgs` is wired to `Modes().Diffing.Ref`, current side context, and the normal file/commit-file contexts (`pkg/gui/controllers/helpers/diff_helper.go:26-47`, `:148-159`). `DiffCmdObj` can run the right command (`pkg/commands/git_commands/diff.go:19-39`), but `RenderDiff()` itself is not PR-aware.  
   **Recommendation:** make PR preview call `DiffCmdObj([]string{base+"..."+head, "--", path})` directly, or add a real `DiffableContext` abstraction for PR files.

7. **The commit panel is not reusable “for free.”**  
   Normal commit refresh is hardwired through `refreshCommitsWithLimit` with `RefName: self.refForLog()` (`pkg/gui/controllers/helpers/refresh_helper.go:351-367`), and `refForLog()` returns `HEAD` outside bisect (`pkg/gui/controllers/helpers/refresh_helper.go:769-775`).  
   **Impact:** PR commits need a separate model/context or parameterized refresh path for `base..head`; otherwise the normal commits list is overwritten or wrong.

8. **Local git diffs may not match GitHub’s comment coordinate system.**  
   The design maps selected local diff lines to GitHub line/side (`design.md:108-111`), but user diff config, rename settings, and external diff can change the rendered shape. Existing commit-file diff code even forces `--no-renames` (`pkg/commands/git_commands/working_tree.go:451-457`). Pager preview via PTY is fine for reading (`pkg/gui/pty.go:40-74`), but the focusable comment surface must be generated from a controlled, GitHub-compatible raw diff, not delta output.  
   **Recommendation:** separate preview diff from selection diff, and test line mapping against GitHub REST `pulls/{n}/files` patches.

9. **Dynamic PR-list tabs are not a safe `viewTabMap()` concern.**  
   Tabs are assigned during view configuration (`pkg/gui/views.go:278-291`) and tab click bindings during keybinding reset (`pkg/gui/keybindings.go:368-376`). `viewTabMap()` is currently a cheap static method (`pkg/gui/gui.go:877-932`).  
   **Impact:** reading gh-dash config or loaded PR sections inside `viewTabMap()` risks stale tabs, missing click bindings, or hot-path shelling. Keep `viewTabMap()` model-backed and cheap; load gh-dash sections elsewhere.

10. **Viewed-state persistence is more than a map key.**  
   DW4 proposes `repoPath + prNumber + path + blob SHA` (`design.md:159-166`), but deleted files have no head blob, renames need old/new path handling, and binary/large files may not have useful patches. AppState is available (`pkg/config/app_config.go:697-713`), but the invalidation model needs explicit status-aware keys.  
   **Recommendation:** use `{repo, pr, path, status, previousPath?, headOid/fileOid or patchHash}` and prune closed PRs.

## P2 nits / cuts

11. **Panel numbers/title prefixes are hard-coded to normal windows.**  
   `views.go:231-255` maps jump labels only for `status/files/branches/commits/stash`. New `prList/prContent/prActivity` views will not get correct `[1]/[2]/[3]` labels without another branch.

12. **Normal refresh work will keep running unless review mode gates it.**  
   Default refresh includes commits, branches, files, stash, status, and pull requests (`pkg/gui/controllers/helpers/refresh_helper.go:86-104`) and always refreshes status (`:219`). This may be harmless but is wasted work and can race with review-specific state.

13. **Cut or defer the full gh-dash PR browser, checks tree logs, batched reviews, and split inversion.**  
   The launched-PR-only workflow already ships value. Full saved-query tabs (`design.md:175-185`), checks logs (`:209-221`), batched review submission (`:223-232`), and focus split tuning (`:234-242`) are separate products.

## Phase ordering

1. **Layout skeleton first:** add review-mode repo state, three dummy review windows/contexts, default-side/focus fixes, panel jumps, and tab cycling. Prove `[`/`]`, `1/2/3`, Escape/pop, config reload, and startup never return to hidden `Files`.
2. **Local-ref read-only spike:** validate checkout remote, fetch base/head into isolated refs, render PR file tree from `git diff --name-status base...head`, and preview file/folder diffs through the pager. No comments yet.
3. **Focusable selection surface:** extract/rebuild a read-only patch-explorer controller over controlled raw diffs; prove line/side mapping with tests before posting.
4. **Standalone comments + viewed persistence:** add GitHub posting and status-aware viewed-state invalidation.
5. **Then add overview/conversation/commits/checks:** start with PR commits as a separate scoped context; leave gh-dash multi-tab PR browsing, checks logs, and batched reviews for later.

