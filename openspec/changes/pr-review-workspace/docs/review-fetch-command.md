# Review-fetch git command (local-ref spine, DW2)

The new dedicated git command that makes `git diff base...head` real and lets the
preview render through the user's pager (delta). There is **no** existing wrapper
for fetching arbitrary PR refspecs — `sync.go:57` is a generic remote fetch.
Add a new func in `pkg/commands/git_commands` (tasks 2.3). Everything below is
project-agnostic: all values are resolved at runtime (see REFERENCES.md).

## Exact argv

Fetch head and base separately, by **explicit repo URL + OID** (not a remote name,
because the head may live in a fork whose URL differs from base):

```sh
git fetch --no-write-fetch-head <head-repo-url> \
  <headOid>:refs/lazygit-review/<pr>/head

git fetch --no-write-fetch-head <base-repo-url> \
  <baseOid>:refs/lazygit-review/<pr>/base
```

- `<head-repo-url>` / `<base-repo-url>` ← `headRepository.url` / `baseRepository.url`
  from the read query (data-shapes.md (a)). For a non-fork PR they are equal; for a
  fork they differ — always use the per-side URL.
- `<headOid>` / `<baseOid>` ← `headRefOid` / `baseRefOid` (the API gives exact
  OIDs, so the fetch is reproducible and never races a moving branch tip).
- `<pr>` ← the runtime PR number.
- The refspec writes ONLY into the isolated `refs/lazygit-review/<pr>/*` namespace.

## Why these flags / what we do NOT do

- `--no-write-fetch-head` — do not clobber `.git/FETCH_HEAD`; this is a background
  read, not a user-initiated fetch, and other lazygit code reads FETCH_HEAD.
- **No checkout** — refs only; the working tree and index are untouched (so the
  user's in-progress work is safe; DW2 must not mutate the worktree).
- **No branch writes** — we write under `refs/lazygit-review/*`, never
  `refs/heads/*` or `refs/remotes/*`, so the PR refs never appear as branches.
- **No new remote** — fetch by URL, so we don't pollute `git remote`.
- **No separate worktree** — a worktree is unnecessary for read-only diffs (extra
  complexity); plain ref fetch suffices. (The old gh-dash `L` flow used an
  `anc-review` worktree + soft-reset — explicitly retired here.)

## Runtime repo-identity derivation (NOT hardcoded)

Before any fetch or comment post, verify the launched checkout actually IS the
target `owner/repo` (DW2 repo-safety P0; the current boot guard `app.go:184-199`
only checks "is a work tree"). A gh-dash misconfig could otherwise write refs into
the wrong repo or post comments against mismatched files.

1. Resolve the checkout path from cwd / `-p` flag.
2. Resolve the target `owner/repo` from the CLI positional (`ReviewTarget`).
3. Confirm a remote of the checkout resolves to that host/owner/repo — e.g.
   `gh repo view --json nameWithOwner` run in the checkout, or match a
   `git remote get-url` against the target. Fail fast with a clear message on
   mismatch.
4. Use the resolved identity to key the review refs and viewed-state. Two
   different repos → different ref namespaces, no collision.

## Merge-base verification

After fetching both refs, confirm they share history before rendering:

```sh
git merge-base refs/lazygit-review/<pr>/base refs/lazygit-review/<pr>/head
```

A non-zero exit means the OIDs don't share an ancestor (bad data / wrong repo) →
surface an error instead of rendering a garbage diff. Diffs then use the
three-dot form `git diff <base>...<head>` (merge-base..head), matching GitHub's PR
diff semantics; `WorkingTree.ShowFileDiffCmdObj(base, head, …, path)`
(`working_tree.go:439`) drives the pager preview.

## Isolation from normal views

`git log --all` would otherwise surface `refs/lazygit-review/*` in the normal
commit graph (`commit_loader.go:589-603`). Exclude the ref glob from the normal
loaders (e.g. `--exclude=refs/lazygit-review/*` before `--all`, or only honor the
glob in review mode). The PR-commits context (DW2 / tasks 5.1) reads
`base..head` explicitly via `CommitLoader.RefName`, so it doesn't need `--all`.

## Prune-on-boot policy (>30 days)

Review refs accumulate. On boot, prune `refs/lazygit-review/*` whose underlying
commit (or a stored creation timestamp) is older than ~30 days:

```sh
git for-each-ref --format='%(refname) %(committerdate:unix)' refs/lazygit-review/
# delete refs older than now-30d:
git update-ref -d <stale-refname>
```

Also prune refs for PRs that are merged/closed (state from the read query),
mirroring the viewed-state pruning (`viewed-state-schema.md`). Pruning is
best-effort: failure to prune must never block the review session.
