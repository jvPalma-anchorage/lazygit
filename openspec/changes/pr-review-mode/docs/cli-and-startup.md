# CLI entry and startup wiring

Self-contained reference for how `lazygit OWNER/REPO PR_NUMBER` is parsed and how
the GUI boots into PR review mode.

## Target invocation

```
lazygit <OWNER/REPO> <PR_NUMBER>      # two bare positionals
# e.g.
cd /home/user/anc-review && llg-dev anchorlabsinc/anchorage 4521
```

The current working directory MUST be inside a checkout of the target repo for
v1 (see design.md D2). gh-dash provides this via `cd {{.RepoPath}}`.

## Why this is non-breaking today

In `pkg/app/entry_point.go`:

- Only ONE positional is registered: `git-arg` at index 1 (entry_point.go:190).
- `parseGitArg` (entry_point.go:249) `log.Fatal`s on anything but
  `status`/`branch`/`log`/`stash`.
- flaggy's default `ShowHelpOnUnexpected=true` rejects an unregistered 2nd
  positional.

So `lazygit owner/repo` fatals in `parseGitArg`, and `lazygit owner/repo 123`
prints help / errors before that. Both two-positional and slash-positional forms
are hard errors today — claiming them for review mode breaks nothing.

> Correction to a common assumption: a bare positional is NOT a repo path. The
> repo path is the `-p`/`--path` FLAG (entry_point.go:184). The positional is the
> git-arg.

## Parsing changes (entry_point.go)

1. Register a second optional positional in `parseCliArgsAndEnvVars`:

```go
prNumberArg := ""
flaggy.AddPositionalValue(&prNumberArg, "pr-number", 2, false,
    "When OWNER/REPO is the first positional, the pull request number to open in PR review mode.")
```

2. Add fields to `cliArgs` and its returned struct:
   `ReviewOwner string`, `ReviewRepo string`, `ReviewPRNumber int`.

3. In `Start()` (entry_point.go:53), AFTER `parseCliArgsAndEnvVars()` and BEFORE
   the `cliArgs.RepoPath` block and `parseGitArg`:

```go
var reviewRe = regexp.MustCompile(`^[^/\s]+/[^/\s]+$`)
if reviewRe.MatchString(cliArgs.GitArg) && prNumberArg != "" {
    n, err := strconv.Atoi(prNumberArg)
    if err != nil {
        log.Fatalf("PR number must be numeric, got %q", prNumberArg)
    }
    repoSpec := strings.TrimSuffix(cliArgs.GitArg, ".git")
    parts := strings.SplitN(repoSpec, "/", 2)
    cliArgs.ReviewOwner, cliArgs.ReviewRepo, cliArgs.ReviewPRNumber = parts[0], parts[1], n
    cliArgs.GitArg = "" // do NOT run parseGitArg on owner/repo
}
```

The owner/repo regex cannot match the four git-arg keywords (none contain `/`),
so non-review invocations fall through unchanged.

## StartArgs threading (pkg/app/types/types.go)

```go
type ReviewTarget struct { Owner, Repo string; PRNumber int }

type StartArgs struct {
    GitArg          GitArg
    IntegrationTest integrationTypes.IntegrationTest
    FilterPath      string
    ScreenMode      string
    ReviewTarget    *ReviewTarget // nil = normal mode
}

func NewStartArgs(filterPath string, gitArg GitArg, screenMode string,
    reviewTarget *ReviewTarget, test integrationTypes.IntegrationTest) StartArgs {
    // ...
}
```

Build a `*ReviewTarget` in `Start()` when the review fields are set and pass it
into the `Run(...)` call's `NewStartArgs(...)`.

## Boot invariant (pkg/app/app.go)

`NewApp` calls `GetRepoPaths` (app.go:121), which shells `git rev-parse
--show-toplevel` in cwd; when absent, `setupRepo` (app.go:182-274) prompts to
`git init` or `os.Exit(1)` (app.go:250-251). Under gh-dash's non-interactive
exec a stdin prompt would HANG.

v1 fix (design.md D2): when `StartArgs.ReviewTarget != nil` and cwd is not a repo,
fail fast with a clear message BEFORE reaching `setupRepo`'s prompt:

```
PR review mode must be launched inside a checkout of OWNER/REPO.
Configure the repo in gh-dash's repoPaths so {{.RepoPath}} resolves.
```

v1 does NOT attempt repo-less boot (that would mean relaxing `resetState`'s
keying off `RepoPaths.WorktreePath()` and the per-repo `gui.git` construction —
deferred).

## GUI boot branch (pkg/gui/gui.go)

`initialContext()` (gui.go:703) returns `types.IListContext`; the PR review
context is NOT an `IListContext`. Branch ONE LEVEL UP, in `resetState`
(gui.go:638), where the initial context is chosen:

```go
ctx := initialContext(contextTree, startArgs)        // existing
if startArgs.ReviewTarget != nil {
    ctx = contextTree.PRReview                        // new review context
}
```

Do not force the review context to satisfy `IListContext`.

## Smoke checklist

- `cd <checkout> && llg-dev OWNER/REPO 123` → boots review mode.
- `lazygit` (no args), `lazygit status|branch|log|stash` → unchanged.
- `lazygit OWNER/REPO` (no number) → unchanged hard error (number is required to
  trigger review mode).
- `lazygit OWNER/REPO abc` → `log.Fatalf` "PR number must be numeric".
