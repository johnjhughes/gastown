package deps

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

var pinnedBeadsVersionPattern = regexp.MustCompile(`github\.com/steveyegge/beads/cmd/bd@v(\d+\.\d+\.\d+)`)

func TestPinnedBeadsVersionsMeetMinimum(t *testing.T) {
	t.Parallel()

	files := []string{
		".github/workflows/ci.yml",
		".github/workflows/nightly-integration.yml",
		".github/workflows/windows-ci.yml",
	}

	repoRoot := repoRootFromDepsTest(t)

	for _, rel := range files {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(filepath.Join(repoRoot, rel))
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}

			matches := pinnedBeadsVersionPattern.FindAllSubmatch(content, -1)
			if len(matches) == 0 {
				t.Fatalf("no pinned bd version found in %s", rel)
			}

			for _, match := range matches {
				version := string(match[1])
				if CompareVersions(version, MinBeadsVersion) < 0 {
					t.Fatalf("%s pins bd %s, below minimum %s", rel, version, MinBeadsVersion)
				}
			}
		})
	}
}

func repoRootFromDepsTest(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
