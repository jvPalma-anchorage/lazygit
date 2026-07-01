package git_commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// The local-ref spine (design DW2 / Phase 2). PR review mode fetches the PR's base
// and head commits into an isolated, per-PR ref namespace so the Files-Changed tab
// can render a real `git diff base...head` through the user's pager (delta) and the
// PR-commits tab can walk `base..head` — all without touching the working tree,
// index, HEAD, or any branch. Everything here is project-agnostic: the repo URLs and
// OIDs are resolved at runtime from the PR read query, never hardcoded.
//
// These are git operations (fetch / merge-base / diff / ref maintenance) rather than
// GitHub API calls, but they are hung off GitHubCommands because they are exclusive
// to the PR review feature and need the same GitCommon (cmd builder, config).

// ReviewRefNamespace is the isolated ref namespace all review refs live under. It is
// deliberately NOT refs/heads/* or refs/remotes/* so the fetched commits never show
// up as branches and are easy to exclude from the normal commit graph.
const ReviewRefNamespace = "refs/lazygit-review"

// ReviewHeadRef / ReviewBaseRef are the two refs written for a given PR number.
func ReviewHeadRef(prNumber int) string {
	return fmt.Sprintf("%s/%d/head", ReviewRefNamespace, prNumber)
}

func ReviewBaseRef(prNumber int) string {
	return fmt.Sprintf("%s/%d/base", ReviewRefNamespace, prNumber)
}

// FetchReviewRefCmdObj builds the fetch for a single side (head or base) by explicit
// repo URL + commit OID. Fetching by URL (not a remote name) is what makes this
// fork-aware: a fork head lives at a different URL than the base. The refspec writes
// ONLY into the isolated namespace, and the flags keep it a pure ref write:
//   - --no-write-fetch-head: don't clobber .git/FETCH_HEAD (other lazygit code reads it)
//   - no checkout / no branch refspec: the working tree, index and HEAD are untouched
func (self *GitHubCommands) FetchReviewRefCmdObj(repoURL string, oid string, destRef string) *oscommands.CmdObj {
	cmdArgs := NewGitCmd("fetch").
		Arg("--no-write-fetch-head").
		Arg("--no-tags").
		Arg(repoURL).
		// The leading '+' forces the ref update: re-reviewing a force-pushed PR points
		// the existing review ref at a new, non-descendant OID, which a non-forced
		// refspec would reject as a non-fast-forward.
		Arg(fmt.Sprintf("+%s:%s", oid, destRef)).
		ToArgv()

	return self.cmd.New(cmdArgs).DontLog()
}

// FetchReviewRefs fetches both the base and head commits for a PR into the isolated
// namespace. Head and base may live in different repositories (fork PRs), so each is
// fetched by its own URL. A failure on either side is returned immediately so the
// caller can fail fast rather than render a half-populated diff.
func (self *GitHubCommands) FetchReviewRefs(prNumber int, headURL string, headOid string, baseURL string, baseOid string) error {
	if err := self.FetchReviewRefCmdObj(baseURL, baseOid, ReviewBaseRef(prNumber)).Run(); err != nil {
		return fmt.Errorf("fetching PR base into %s: %w", ReviewBaseRef(prNumber), err)
	}
	if err := self.FetchReviewRefCmdObj(headURL, headOid, ReviewHeadRef(prNumber)).Run(); err != nil {
		return fmt.Errorf("fetching PR head into %s: %w", ReviewHeadRef(prNumber), err)
	}
	return nil
}

// ReviewMergeBaseCmdObj builds `git merge-base <base> <head>`. Used both to verify
// the two fetched refs share history (a non-zero exit means bad data / wrong repo)
// and to obtain the merge-base commit that the diffs are taken from, so that
// `git diff <mergeBase> <head>` reproduces GitHub's three-dot PR diff semantics.
func (self *GitHubCommands) ReviewMergeBaseCmdObj(prNumber int) *oscommands.CmdObj {
	cmdArgs := NewGitCmd("merge-base").
		Arg(ReviewBaseRef(prNumber)).
		Arg(ReviewHeadRef(prNumber)).
		ToArgv()

	return self.cmd.New(cmdArgs).DontLog()
}

// ReviewMergeBase returns the merge-base commit of the PR's base and head refs, or an
// error when they don't share history (which must be surfaced rather than rendering a
// garbage diff).
func (self *GitHubCommands) ReviewMergeBase(prNumber int) (string, error) {
	out, err := self.ReviewMergeBaseCmdObj(prNumber).RunWithOutput()
	if err != nil {
		return "", fmt.Errorf("PR base and head do not share history: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// ReviewChangedFile is one entry of the PR's changed-file set, parsed from
// `git diff --name-status`. Status is the single-letter git change status; OldPath is
// set only for renames/copies (status R/C). HeadOid is the blob SHA on the head side
// (or the base side for a deletion), used as the persisted "viewed"-state identity.
type ReviewChangedFile struct {
	Status    string
	Path      string
	OldPath   string
	HeadOid   string
	PatchHash string // fallback content identity when no blob OID resolves
}

// HashReviewPatch returns a stable hex digest of a file's unified diff, used as the
// viewed-state content identity in the rare case where no blob OID resolves.
func HashReviewPatch(patch string) string {
	sum := sha256.Sum256([]byte(patch))
	return hex.EncodeToString(sum[:])
}

// ReviewFileBlobOid returns the blob SHA of a path at the given ref
// (`git rev-parse <ref>:<path>`). Empty (no error surfaced) when the path does not
// exist at that ref (e.g. a file added on head won't exist at base, and vice versa).
func (self *GitHubCommands) ReviewFileBlobOid(ref string, path string) string {
	cmdArgs := NewGitCmd("rev-parse").
		Arg(fmt.Sprintf("%s:%s", ref, path)).
		ToArgv()
	out, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// ReviewFileBlobOids resolves the blob SHA of many paths at a ref in a SINGLE
// `git ls-tree` call, so resolving the viewed-state identity for a large PR costs one
// git invocation rather than one per file. Paths absent at the ref are simply missing
// from the returned map.
func (self *GitHubCommands) ReviewFileBlobOids(ref string, paths []string) map[string]string {
	if len(paths) == 0 {
		return map[string]string{}
	}
	cmdArgs := NewGitCmd("ls-tree").
		Arg(ref).
		Arg("--").
		Arg(paths...).
		ToArgv()
	out, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		return map[string]string{}
	}
	return parseLsTreeBlobOids(out)
}

// parseLsTreeBlobOids parses `git ls-tree` output lines of the form
// "<mode> blob <oid>\t<path>" into a path->oid map (blobs only).
func parseLsTreeBlobOids(out string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		tab := strings.IndexByte(line, '\t')
		if tab == -1 {
			continue
		}
		meta := strings.Fields(line[:tab])
		if len(meta) >= 3 && meta[1] == "blob" {
			result[line[tab+1:]] = meta[2]
		}
	}
	return result
}

// ReviewChangedFilesCmdObj builds `git diff --name-status <from> <to>`. The caller
// passes the merge-base as `from` and the head ref as `to` so the result matches the
// PR's three-dot diff. --no-renames keeps the tree consistent with the per-file
// preview, which also forces --no-renames (working_tree.go ShowFileDiffCmdObj): a
// rename shows as a delete + an add in both, so the tree and the diff agree.
func (self *GitHubCommands) ReviewChangedFilesCmdObj(from string, to string) *oscommands.CmdObj {
	cmdArgs := NewGitCmd("diff").
		Arg("--name-status").
		Arg("--no-renames").
		Arg(from).
		Arg(to).
		ToArgv()

	return self.cmd.New(cmdArgs).DontLog()
}

// ReviewChangedFiles returns the parsed changed-file set for the PR diff between the
// given refs (typically merge-base..head).
func (self *GitHubCommands) ReviewChangedFiles(from string, to string) ([]ReviewChangedFile, error) {
	out, err := self.ReviewChangedFilesCmdObj(from, to).RunWithOutput()
	if err != nil {
		return nil, err
	}
	return parseReviewChangedFiles(out), nil
}

// parseReviewChangedFiles parses `git diff --name-status` output. Each line is a
// status letter, a tab, then the path — except renames/copies (R###/C###) which carry
// an extra tab-separated old path. Blank lines are ignored.
func parseReviewChangedFiles(out string) []ReviewChangedFile {
	lines := strings.Split(out, "\n")
	files := make([]ReviewChangedFile, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		// R100/C75 etc. carry a similarity score after the letter; keep just the letter.
		letter := status[:1]
		file := ReviewChangedFile{Status: letter}
		if (letter == "R" || letter == "C") && len(parts) >= 3 {
			file.OldPath = parts[1]
			file.Path = parts[2]
		} else {
			file.Path = parts[len(parts)-1]
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		// Preserve a nil result for empty/whitespace output (no changed files).
		return nil
	}
	return files
}

// PruneStaleReviewRefsCmdObj lists every review ref with its commit's unix
// timestamp, so the caller can delete the ones older than the retention window.
func (self *GitHubCommands) PruneStaleReviewRefsListCmdObj() *oscommands.CmdObj {
	cmdArgs := NewGitCmd("for-each-ref").
		Arg("--format=%(refname) %(committerdate:unix)").
		Arg(ReviewRefNamespace + "/").
		ToArgv()

	return self.cmd.New(cmdArgs).DontLog()
}

// DeleteReviewRefCmdObj builds `git update-ref -d <ref>` to remove a single review
// ref. Deleting a ref never touches the working tree or index.
func (self *GitHubCommands) DeleteReviewRefCmdObj(ref string) *oscommands.CmdObj {
	cmdArgs := NewGitCmd("update-ref").
		Arg("-d").
		Arg(ref).
		ToArgv()

	return self.cmd.New(cmdArgs).DontLog()
}

// PruneStaleReviewRefs deletes review refs whose commit is older than maxAgeDays,
// relative to nowUnix (passed in so the caller controls the clock and tests stay
// deterministic). Pruning is best-effort: any error is returned but callers treat it
// as non-fatal — a failure to prune must never block the review session.
func (self *GitHubCommands) PruneStaleReviewRefs(nowUnix int64, maxAgeDays int) error {
	out, err := self.PruneStaleReviewRefsListCmdObj().RunWithOutput()
	if err != nil {
		return err
	}
	cutoff := nowUnix - int64(maxAgeDays)*24*60*60
	for _, ref := range staleReviewRefs(out, cutoff) {
		if delErr := self.DeleteReviewRefCmdObj(ref).Run(); delErr != nil && err == nil {
			err = delErr
		}
	}
	return err
}

// staleReviewRefs parses `for-each-ref` output and returns the refnames whose commit
// timestamp is at or before cutoff. A ref with an unparseable timestamp is left alone.
func staleReviewRefs(out string, cutoff int64) []string {
	var stale []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		ts, convErr := strconv.ParseInt(fields[1], 10, 64)
		if convErr != nil {
			continue
		}
		if ts <= cutoff {
			stale = append(stale, fields[0])
		}
	}
	return stale
}
