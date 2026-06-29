# Persisted viewed-state schema (DW4)

"Viewed" marks persist across leaving/returning to a PR and auto-invalidate when a
new commit changes the file's content (fixes bug #2; mirrors GitHub's "viewed"
checkbox). Persisted in `AppState` (YAML at `$XDG_STATE_HOME/lazygit/state.yml`),
following the existing `GithubPullRequests` precedent (`app_config.go:712`).
All keys are runtime-resolved — no hardcoded repo/path (REFERENCES.md mandate).

## Go struct — add to `AppState` (`pkg/config/app_config.go:697`)

```go
type AppState struct {
    // ...existing fields (LastUpdateCheck, RecentRepos, GithubPullRequests, ...)

    // ReviewedFiles tracks per-file "viewed" marks for PR review mode, keyed by
    // a stable composite key (see reviewedFileKey). A mark is honored only while
    // the stored content identity still matches the working diff, so a new commit
    // that touches the file auto-clears it.
    ReviewedFiles map[string]ReviewedFileState `yaml:"reviewedFiles"`
}

// ReviewedFileState is one persisted "viewed" mark. Identity is content-based:
// FileOid (the head blob SHA) for text/binary files, or PatchHash (hash of the
// unified diff) when no single blob applies. Path/PreviousPath/Status disambiguate
// renames and deletions.
type ReviewedFileState struct {
    Repo         string `yaml:"repo"`         // runtime owner/repo, e.g. resolved nameWithOwner
    PR           int    `yaml:"pr"`           // PR number
    Path         string `yaml:"path"`         // new-file path (tree key)
    Status       string `yaml:"status"`       // added|modified|removed|renamed|binary
    PreviousPath string `yaml:"previousPath,omitempty"` // old path for renamed/copied
    FileOid      string `yaml:"fileOid,omitempty"`      // head blob SHA when one exists
    PatchHash    string `yaml:"patchHash,omitempty"`    // hash of the diff when no blob
    ViewedAt     int64  `yaml:"viewedAt"`     // unix seconds, for pruning/debug
}
```

Key builder (status-aware; the map key, not stored on the struct):

```go
func reviewedFileKey(repo string, pr int, path string) string {
    return fmt.Sprintf("%s#%d#%s", repo, pr, path)
}
```

`{repo, pr, path}` identifies the row; `{status, previousPath, fileOid|patchHash}`
in the value is the **content identity** checked for invalidation. Read via
`GetAppState`, write via `SaveAppState` on toggle.

## YAML serialization (state.yml)

```yaml
reviewedFiles:
  myorg/myrepo#4521#pkg/gui/gui.go:
    repo: myorg/myrepo
    pr: 4521
    path: pkg/gui/gui.go
    status: modified
    fileOid: 9f1c2ab3e4d5...
    viewedAt: 1750000000
  myorg/myrepo#4521#pkg/old/name.go:
    repo: myorg/myrepo
    pr: 4521
    path: pkg/new/name.go
    status: renamed
    previousPath: pkg/old/name.go
    fileOid: 1a2b3c4d...
    viewedAt: 1750000050
```

(`myorg/myrepo` is illustrative — the real value is the runtime-resolved
`nameWithOwner`.)

## Invalidation rule

On load, a file shows as **viewed** only if its *current* content identity still
matches the stored one:

```
current fileOid (git rev-parse <head>:<path>)  != stored FileOid   → NOT viewed
current patchHash (binary / no-blob case)       != stored PatchHash → NOT viewed
```

A new commit that changes the file produces a new blob SHA → the mark auto-clears.
`FileOid` is preferred (one `git rev-parse <headRef>:<path>` against the fetched
review ref — cheap, and it's what GitHub keys on); `PatchHash` is the fallback.

## Per-status handling

| status | content identity | notes |
|---|---|---|
| `added` | `FileOid` of the new blob (base side absent) | normal case |
| `modified` | `FileOid` of the head blob | normal case |
| `removed` (deleted) | `FileOid` = **base** blob (no head blob exists) | key off the deleted-from-base OID; or `PatchHash` |
| `renamed` | `FileOid` of head blob + `PreviousPath` set | path changed → key on new `Path`, retain old via `PreviousPath` |
| `binary` | `FileOid` of head blob (no `patch`) | no `PatchHash`; rely on blob SHA |

For deletions and pure-rename-no-content-change cases where no useful head blob
exists, fall back to `PatchHash` over the `git diff <base>...<head> -- <path>`
output.

## Pruning (state-file hygiene)

On boot / opportunistically:

1. **Merged/closed PRs** — drop every `ReviewedFiles` entry whose PR `state` (from
   the read query) is `MERGED`/`CLOSED`. Done alongside review-ref pruning
   (`review-fetch-command.md`).
2. Optionally drop entries whose `ViewedAt` is very old, in line with the ~30-day
   review-ref prune.

## Backward compatibility

`ReviewedFiles` is a new top-level field. An old `state.yml` written before this
change simply lacks the `reviewedFiles:` key → YAML unmarshals it to a `nil` map
(zero value), exactly like `GithubPullRequests` did when it was added. No
migration needed; `getDefaultAppState()` may initialize it to an empty map for
write-safety. Old lazygit versions reading a new state.yml ignore the unknown key.
