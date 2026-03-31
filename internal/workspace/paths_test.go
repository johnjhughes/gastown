package workspace

import (
	"path/filepath"
	"testing"
)

func TestJSONLGitBackupRepo(t *testing.T) {
	townRoot := filepath.Join(string(filepath.Separator), "tmp", "town")
	want := filepath.Join(townRoot, ".dolt-archive", "git")

	if got := JSONLGitBackupRepo(townRoot); got != want {
		t.Fatalf("JSONLGitBackupRepo(%q) = %q, want %q", townRoot, got, want)
	}
}
