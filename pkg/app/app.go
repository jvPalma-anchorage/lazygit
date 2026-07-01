package app

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	appTypes "github.com/jesseduffield/lazygit/pkg/app/types"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/constants"
	"github.com/jesseduffield/lazygit/pkg/env"
	"github.com/jesseduffield/lazygit/pkg/gui"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	integrationTypes "github.com/jesseduffield/lazygit/pkg/integration/types"
	"github.com/jesseduffield/lazygit/pkg/logs"
	"github.com/jesseduffield/lazygit/pkg/updates"
)

// App is the struct that's instantiated from within main.go and it manages
// bootstrapping and running the application.
type App struct {
	*common.Common
	closers   []io.Closer
	Config    config.AppConfigurer
	OSCommand *oscommands.OSCommand
	Gui       *gui.Gui
}

func Run(
	config config.AppConfigurer,
	common *common.Common,
	startArgs appTypes.StartArgs,
) {
	// PR review mode (v1) requires a local checkout. Fail fast here rather than
	// letting setupRepo prompt for `git init` on stdin (which would hang under a
	// non-interactive launch from gh-dash).
	if startArgs.ReviewTarget != nil {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		// Integration tests boot review mode against a hermetic local fixture whose
		// remotes can't (and shouldn't) resolve to the launched owner/repo, and have
		// no gh CLI; skip the live identity/gh checks there.
		skipLiveChecks := startArgs.IntegrationTest != nil
		if err := ensureReviewModeRepo(startArgs.ReviewTarget, cwd, common.Tr, skipLiveChecks); err != nil {
			log.Fatal(err)
		}
	}

	app, err := NewApp(config, startArgs.IntegrationTest, common)

	if err == nil {
		err = app.Run(startArgs)
	}

	if err != nil {
		if errorMessage, known := knownError(common.Tr, err); known {
			log.Fatal(errorMessage)
		}
		newErr := errors.Wrap(err, 0)
		stackTrace := newErr.ErrorStack()
		app.Log.Error(stackTrace)

		log.Fatalf("%s: %s\n\n%s", common.Tr.ErrorOccurred, constants.Links.Issues, stackTrace)
	}
}

func NewCommon(config config.AppConfigurer) (*common.Common, error) {
	userConfig := config.GetUserConfig()
	appState := config.GetAppState()
	log := newLogger(config)
	// Initialize with English for the time being; the real translation set for
	// the configured language will be read after reading the user config
	tr := i18n.EnglishTranslationSet()

	cmn := &common.Common{
		Log:      log,
		Tr:       tr,
		AppState: appState,
		Debug:    config.GetDebug(),
		Fs:       afero.NewOsFs(),
	}
	cmn.SetUserConfig(userConfig)
	return cmn, nil
}

func newLogger(cfg config.AppConfigurer) *logrus.Entry {
	if cfg.GetDebug() {
		logPath, err := config.LogPath()
		if err != nil {
			log.Fatal(err)
		}
		return logs.NewDevelopmentLogger(logPath)
	}

	return logs.NewProductionLogger()
}

// NewApp bootstrap a new application
func NewApp(config config.AppConfigurer, test integrationTypes.IntegrationTest, common *common.Common) (*App, error) {
	app := &App{
		closers: []io.Closer{},
		Config:  config,
		Common:  common,
	}

	app.OSCommand = oscommands.NewOSCommand(common, config, oscommands.GetPlatform(), oscommands.NewNullGuiIO(app.Log))

	updater, err := updates.NewUpdater(common, config, app.OSCommand)
	if err != nil {
		return app, err
	}

	dirName, err := os.Getwd()
	if err != nil {
		return app, err
	}

	gitVersion, err := app.validateGitVersion()
	if err != nil {
		return app, err
	}

	// If we're not in a repo, GetRepoPaths will return an error. The error is moot for us
	// at this stage, since we'll try to init a new repo in setupRepo(), below
	repoPaths, err := git_commands.GetRepoPaths(app.OSCommand.Cmd, gitVersion)
	if err != nil {
		common.Log.Infof("Error getting repo paths: %v", err)
	}

	showRecentRepos, err := app.setupRepo(repoPaths)
	if err != nil {
		return app, err
	}

	// used for testing purposes
	if os.Getenv("SHOW_RECENT_REPOS") == "true" {
		showRecentRepos = true
	}

	app.Gui, err = gui.NewGui(common, config, gitVersion, updater, showRecentRepos, dirName, test)
	if err != nil {
		return app, err
	}
	return app, nil
}

const minGitVersionStr = "2.32.0"

func minGitVersionErrorMessage(tr *i18n.TranslationSet) string {
	return fmt.Sprintf(tr.MinGitVersionError, minGitVersionStr)
}

func (app *App) validateGitVersion() (*git_commands.GitVersion, error) {
	version, err := git_commands.GetGitVersion(app.OSCommand)
	// if we get an error anywhere here we'll show the same status
	minVersionError := errors.New(minGitVersionErrorMessage(app.Tr))
	if err != nil {
		return nil, minVersionError
	}

	minRequiredVersion, _ := git_commands.ParseGitVersion(minGitVersionStr)
	if version.IsOlderThanVersion(minRequiredVersion) {
		return nil, minVersionError
	}

	return version, nil
}

func isDirectoryAGitRepository(dir string) (bool, error) {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return info != nil, err
}

// ensureReviewModeRepo returns an error when PR review mode is requested from a
// directory that is not inside a git work tree. v1 requires launching inside a
// checkout (gh-dash does `cd {{.RepoPath}}` before launching). We use git's own
// work-tree detection (not a shallow `.git` stat) so launching from a subdirectory
// of the checkout works, matching how lazygit resolves the repo normally.
func ensureReviewModeRepo(target *appTypes.ReviewTarget, cwd string, tr *i18n.TranslationSet, skipLiveChecks bool) error {
	if target == nil {
		return nil
	}

	if !isInsideGitWorkTree(cwd) {
		return fmt.Errorf(tr.PrReviewMustBeInCheckout, fmt.Sprintf("%s/%s", target.Owner, target.Repo))
	}

	if skipLiveChecks {
		return nil
	}

	// Fail fast (D1.5) when the gh CLI is missing or unauthenticated: every data
	// fetch and comment post shells out to it, so otherwise the session could only
	// ever show load errors. `gh auth status` is bounded by gh's own timeout, so this
	// check cannot hang the boot.
	if _, err := exec.LookPath("gh"); err != nil {
		return errors.New(tr.PrReviewGhNotFound)
	}
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		return errors.New(tr.PrReviewGhNotAuthenticated)
	}

	// Repo-safety P0: confirm the checkout actually IS the target owner/repo, so a
	// gh-dash misconfig can't write review refs into — or post comments against — the
	// wrong repository. The head may live in a fork, but gh-dash cds into the BASE
	// checkout, so a remote must resolve to the launched owner/repo.
	if !checkoutMatchesTarget(gitRemoteURLs(cwd), target.Owner, target.Repo) {
		return fmt.Errorf(tr.PrReviewRepoMismatch, fmt.Sprintf("%s/%s", target.Owner, target.Repo))
	}

	return nil
}

// gitRemoteURLs returns the fetch/push URLs of every remote configured in dir.
// Returns nil when git is unavailable or there are no remotes.
func gitRemoteURLs(dir string) []string {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	urls := make([]string, 0, len(lines))
	seen := map[string]bool{}
	for _, line := range lines {
		fields := strings.Fields(line)
		// Format: "<name>\t<url> (fetch|push)"
		if len(fields) < 2 || seen[fields[1]] {
			continue
		}
		seen[fields[1]] = true
		urls = append(urls, fields[1])
	}
	return urls
}

// checkoutMatchesTarget reports whether any of the given remote URLs resolves to the
// target owner/repo on GitHub (case-insensitive). The host is checked too: a remote
// like https://evil.example/owner/repo must NOT satisfy a github.com target. v1 is
// github.com-only (matching the hard-coded auth host), so any github.com host
// (including a *.github.com alias) is accepted and everything else is rejected.
func checkoutMatchesTarget(remoteURLs []string, owner string, repo string) bool {
	for _, url := range remoteURLs {
		host, o, r, ok := parseRepoIdentity(url)
		if ok && isGitHubHost(host) && strings.EqualFold(o, owner) && strings.EqualFold(r, repo) {
			return true
		}
	}
	return false
}

func isGitHubHost(host string) bool {
	host = strings.ToLower(host)
	return host == "github.com" || strings.HasSuffix(host, ".github.com")
}

// parseRepoIdentity extracts (host, owner, repo) from a git remote URL across the
// common forms: https://host/owner/repo(.git), git@host:owner/repo(.git),
// ssh://git@host/owner/repo(.git). A trailing ".git" is stripped. Returns ok=false
// when the URL doesn't look like an owner/repo remote.
func parseRepoIdentity(remoteURL string) (string, string, string, bool) {
	s := strings.TrimSpace(remoteURL)
	s = strings.TrimSuffix(s, ".git")

	// Normalise scp-like syntax (git@host:owner/repo) and URL schemes to
	// "host/owner/repo".
	if i := strings.Index(s, "://"); i != -1 {
		s = s[i+3:] // strip scheme
	}
	if at := strings.LastIndex(s, "@"); at != -1 {
		s = s[at+1:] // strip user@
	}
	// Now s is "host[:port]/owner/repo" or "host:owner/repo" (scp form). Fold the
	// first ':' (scp separator or port) into a '/' so a uniform split yields the
	// host first and owner/repo last.
	if colon := strings.Index(s, ":"); colon != -1 {
		s = s[:colon] + "/" + s[colon+1:]
	}

	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return "", "", "", false
	}
	host := parts[0]
	repo := parts[len(parts)-1]
	owner := parts[len(parts)-2]
	if host == "" || owner == "" || repo == "" {
		return "", "", "", false
	}
	return host, owner, repo, true
}

// isInsideGitWorkTree reports whether dir is anywhere inside a git work tree,
// walking up the directory hierarchy the way git does (so subdirectories, linked
// worktrees, and submodules all resolve correctly). Falls back to false when the
// git binary is unavailable or errors.
func isInsideGitWorkTree(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func openRecentRepo(app *App) bool {
	for _, repoDir := range app.Config.GetAppState().RecentRepos {
		if isRepo, _ := isDirectoryAGitRepository(repoDir); isRepo {
			if err := os.Chdir(repoDir); err == nil {
				return true
			}
		}
	}

	return false
}

func (app *App) setupRepo(
	repoPaths *git_commands.RepoPaths,
) (bool, error) {
	if env.GetGitDirEnv() != "" {
		// we've been given the git dir directly. Skip setup
		return false, nil
	}

	// if we are not in a git repo, we ask if we want to `git init`
	if repoPaths == nil {
		cwd, err := os.Getwd()
		if err != nil {
			return false, err
		}

		if isRepo, err := isDirectoryAGitRepository(cwd); isRepo {
			return false, err
		}

		var shouldInitRepo bool
		initialBranchArg := ""
		switch app.UserConfig().NotARepository {
		case "prompt":
			// Offer to initialize a new repository in current directory.
			fmt.Print(app.Tr.CreateRepo)
			response, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			shouldInitRepo = (strings.Trim(response, " \r\n") == "y")
			if shouldInitRepo {
				// Ask for the initial branch name
				fmt.Print(app.Tr.InitialBranch)
				response, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				if trimmedResponse := strings.Trim(response, " \r\n"); len(trimmedResponse) > 0 {
					initialBranchArg += "--initial-branch=" + trimmedResponse
				}
			}
		case "create":
			shouldInitRepo = true
		case "skip":
			shouldInitRepo = false
		case "quit":
			fmt.Fprintln(os.Stderr, app.Tr.NotARepository)
			os.Exit(1)
		default:
			fmt.Fprintln(os.Stderr, app.Tr.IncorrectNotARepository)
			os.Exit(1)
		}

		if shouldInitRepo {
			args := []string{"git", "init"}
			if initialBranchArg != "" {
				args = append(args, initialBranchArg)
			}
			if err := app.OSCommand.Cmd.New(args).Run(); err != nil {
				return false, err
			}

			return false, nil
		}

		// check if we have a recent repo we can open
		for _, repoDir := range app.Config.GetAppState().RecentRepos {
			if isRepo, _ := isDirectoryAGitRepository(repoDir); isRepo {
				if err := os.Chdir(repoDir); err == nil {
					return true, nil
				}
			}
		}

		fmt.Fprintln(os.Stderr, app.Tr.NoRecentRepositories)
		os.Exit(1)
	}

	// Run this afterward so that the previous repo creation steps can run without this interfering
	if repoPaths.IsBareRepo() {

		fmt.Print(app.Tr.BareRepo)

		response, _ := bufio.NewReader(os.Stdin).ReadString('\n')

		if shouldOpenRecent := strings.Trim(response, " \r\n") == "y"; !shouldOpenRecent {
			os.Exit(0)
		}

		if didOpenRepo := openRecentRepo(app); didOpenRepo {
			return true, nil
		}

		fmt.Println(app.Tr.NoRecentRepositories)
		os.Exit(1)
	}

	return false, nil
}

func (app *App) Run(startArgs appTypes.StartArgs) error {
	err := app.Gui.RunAndHandleError(startArgs)
	return err
}

// Close closes any resources
func (app *App) Close() error {
	for _, closer := range app.closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}

	return nil
}
