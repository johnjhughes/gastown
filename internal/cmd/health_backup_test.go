package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCheckBackupHealthUsesTownRootJSONLGitRepo(t *testing.T) {
	townRoot := t.TempDir()
	setTestHome(t, t.TempDir())

	gitRepo := filepath.Join(townRoot, ".dolt-archive", "git")
	initTestGitRepoWithCommit(t, gitRepo, "hq.jsonl", "{\"id\":\"hq-abc1\"}\n")

	report := checkBackupHealth(townRoot)
	if report.JSONLFreshness == "" {
		t.Fatal("expected JSONL freshness when town-root archive repo exists")
	}
	if report.JSONLAgeSeconds < 0 {
		t.Fatalf("expected non-negative JSONL age, got %d", report.JSONLAgeSeconds)
	}
}

func TestDoltArchiveShellArtifactsUseTownRootPaths(t *testing.T) {
	repoRoot := repoRootFromCaller(t)

	runScript := mustReadFile(t, filepath.Join(repoRoot, "plugins", "dolt-archive", "run.sh"))
	if strings.Contains(runScript, "$HOME/gt/.dolt-archive/git") {
		t.Fatal("run.sh should not hard-code $HOME/gt for the JSONL git repo")
	}
	if !strings.Contains(runScript, `BACKUP_REPO="${BACKUP_REPO:-$TOWN_ROOT/.dolt-archive/git}"`) {
		t.Fatal("run.sh should resolve BACKUP_REPO from TOWN_ROOT")
	}

	pluginDoc := mustReadFile(t, filepath.Join(repoRoot, "plugins", "dolt-archive", "plugin.md"))
	if strings.Contains(pluginDoc, "$HOME/gt/.dolt-archive/git") {
		t.Fatal("plugin.md should not hard-code $HOME/gt for the JSONL git repo")
	}
	if !strings.Contains(pluginDoc, `BACKUP_REPO="${BACKUP_REPO:-$GT_TOWN_ROOT/.dolt-archive/git}"`) {
		t.Fatal("plugin.md should document GT_TOWN_ROOT-based BACKUP_REPO resolution")
	}
}

func initTestGitRepoWithCommit(t *testing.T, repoPath string, fileName string, contents string) {
	t.Helper()

	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	runGit(t, repoPath, "init", "-b", "main")
	runGit(t, repoPath, "config", "user.name", "Test User")
	runGit(t, repoPath, "config", "user.email", "test@example.com")

	filePath := filepath.Join(repoPath, fileName)
	if err := os.WriteFile(filePath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	runGit(t, repoPath, "add", fileName)
	runGit(t, repoPath, "commit", "-m", "test commit")
}

func runGit(t *testing.T, repoPath string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func repoRootFromCaller(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
