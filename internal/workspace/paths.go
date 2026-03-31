package workspace

import "path/filepath"

// JSONLGitBackupRepo returns the canonical town-local git repo for JSONL backups.
func JSONLGitBackupRepo(townRoot string) string {
	return filepath.Join(townRoot, ".dolt-archive", "git")
}
