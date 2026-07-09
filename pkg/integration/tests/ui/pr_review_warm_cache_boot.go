package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewWarmCacheBoot proves the D3.1 warm-boot path: with valid per-PR snapshots on
// disk (meta.json + files.json captured at the OIDs the local review refs point at),
// the whole workspace renders from the cache with ZERO gh/network invocations. No
// fixture env is set — the only non-network data source is the cache — and
// LAZYGIT_PR_REVIEW_OFFLINE skips the background revalidation. The changed-file list
// in the cache deliberately differs from the real refs' diff (CACHED_MARKER.go does
// not exist in git) so the rendered tree can only have come from the snapshots.
var PrReviewWarmCacheBoot = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: a warm boot renders the workspace from the per-PR cache without any network fetch",
	ExtraCmdArgs: []string{"cachetest/cachetest", "7"},
	ExtraEnvVars: map[string]string{
		"LAZYGIT_PR_REVIEW_LOCAL_REFS": "7",
		"LAZYGIT_PR_REVIEW_CACHE_DIR":  "prcache",
		"LAZYGIT_PR_REVIEW_OFFLINE":    "1",
	},
	Skip:        false,
	SetupConfig: func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("seed", "seed\n")
		shell.Commit("init")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/base", "HEAD"})
		shell.CreateFileAndAdd("real.go", "package real\n")
		shell.Commit("pr head")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/head", "HEAD"})

		// Seed the snapshot store with envelopes whose OIDs match the refs above.
		// Built with shell substitution because the OIDs are only known at run time.
		shell.RunShellCommand(`
HEAD_OID=$(git rev-parse refs/lazygit-review/7/head)
BASE_OID=$(git rev-parse refs/lazygit-review/7/base)
mkdir -p prcache/cachetest/cachetest/7
cat > prcache/cachetest/cachetest/7/meta.json <<EOF
{"fetchedAt":"2026-07-01T00:00:00Z","headRefOid":"$HEAD_OID","payload":{
  "ID":"PR_7","Number":7,"Title":"CACHED_TITLE","State":"OPEN",
  "HeadRefOid":"$HEAD_OID","BaseRefOid":"$BASE_OID",
  "Reviewers":[{"Login":"cached-reviewer","State":"APPROVED"}]
}}
EOF
cat > prcache/cachetest/cachetest/7/files.json <<EOF
{"fetchedAt":"2026-07-01T00:00:00Z","headRefOid":"$HEAD_OID","payload":{
  "BaseRefOid":"$BASE_OID","MergeBase":"$BASE_OID",
  "Files":[{"Status":"A","Path":"CACHED_MARKER.go","HeadOid":"nonexistent"}]
}}
EOF
cat > prcache/cachetest/cachetest/7/checks.json <<EOF
{"fetchedAt":"2026-07-01T00:00:00Z","headRefOid":"","payload":[
  {"Workflow":"CI","Name":"CACHED_CHECK","Bucket":"pass","State":"SUCCESS"}
]}
EOF
`)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// The tree renders the cache's changed-file set — a file that does NOT exist
		// in the real refs' diff, so it can only have come from the snapshots.
		t.Views().PrReview().
			IsFocused().
			ContainsLines(Contains("CACHED_MARKER.go")).
			Content(DoesNotContain("real.go"))

		// The Activity window is fed from the cached meta snapshot too.
		t.GlobalPress(keys.Universal.JumpToBlock[2])
		t.Views().PrConversation().
			IsFocused().
			ContainsLines(Contains("cached-reviewer").Contains("APPROVED"))

		// The Checks tab renders from its own snapshot — no `gh pr checks` run.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PrChecks().
			IsFocused().
			ContainsLines(
				Contains("CI"),
				Contains("CACHED_CHECK"),
			)
	},
})
