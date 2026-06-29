# gh-dash → lazygit handoff

Self-contained reference for launching PR review mode from gh-dash
(dlvhdr/gh-dash v4.24.1). **No gh-dash code change is required** — it already
exposes the needed template variables and runs arbitrary shell commands with full
TTY handoff (proven by the user's existing `L`/`E`/`D`/`B` lazygit bindings).

## How gh-dash runs custom PR commands

From dlvhdr/gh-dash `internal/tui/modelUtils.go` `runCustomPRCommand` +
`resolveTemplateInput`: a `prs:` keybinding command is a Go template, executed via
`shell.Command` + `tea.ExecProcess` (so the string is shell-interpreted and the
child process inherits the full TTY).

Available template variables for a `prs:` command:

| Variable          | Type   | Value                                                            |
|-------------------|--------|-----------------------------------------------------------------|
| `{{.RepoName}}`   | string | `OWNER/REPO`, e.g. `anchorlabsinc/anchorage`                     |
| `{{.PrNumber}}`   | int    | the PR number                                                   |
| `{{.HeadRefName}}`| string | PR source branch                                                |
| `{{.BaseRefName}}`| string | PR target branch                                                |
| `{{.Author}}`     | string | PR author login                                                 |
| `{{.RepoPath}}`   | string | local checkout path from the top-level `repoPaths:` map keyed by `RepoName`; **EMPTY** if neither the map nor gh-dash's launch repo matches |

`RepoName`/`PrNumber`/`BaseRefName` are proven by the user's working config;
`HeadRefName`/`Author`/`RepoPath` are from source (low risk — predate v4).

## Recommended keybinding

Add under `keybindings.prs` in `~/.config/gh-dash/config.yml`. Use a free key
(`R` for "review"; `E`/`L`/`D`/`B` are taken by the user's existing bindings).
`cd` into the resolved checkout first (v1 requires a real work tree), then invoke
the dev build with two positionals. Use the ABSOLUTE binary path because PATH
under gh-dash's `shell.Command` is not guaranteed to include `~/.local/bin`.

```yaml
keybindings:
  prs:
    - key: R
      name: review PR in lazygit
      command: >
        cd {{.RepoPath}} &&
        /home/user/.local/bin/llg-dev {{.RepoName}} {{.PrNumber}}

# repoPaths MUST contain the repo so {{.RepoPath}} resolves:
repoPaths:
  anchorlabsinc/anchorage: /home/user/anc-review
```

Concrete argv gh-dash will exec for PR 4521:

```
cd /home/user/anc-review && /home/user/.local/bin/llg-dev anchorlabsinc/anchorage 4521
```

## Hardening alternative: `-p` instead of `cd`

Equivalent, avoids relying on shell cwd, fails more cleanly on a bad path, single
process:

```yaml
command: /home/user/.local/bin/llg-dev -p {{.RepoPath}} {{.RepoName}} {{.PrNumber}}
```

`-p`/`--path` (registered in `parseCliArgsAndEnvVars`, entry_point.go:232) is a
pre-existing lazygit flag that `Chdir`s into the repo before the GUI boots.

## Gotchas

- **Empty `{{.RepoPath}}`** → `cd ` becomes `cd $HOME`, opening the wrong repo.
  The repo MUST be in `repoPaths:` (or gh-dash launched inside the checkout).
  lazygit's v1 boot guard (design.md D2) fails fast with a clear message if cwd
  is not the repo, rather than prompting for `git init`.
- **PATH** may not include `~/.local/bin` under gh-dash's shell — use the absolute
  `llg-dev` path.
- **Version drift** — verified against v4.24.1; smoke-test by pressing the key and
  confirming the spawned argv.
