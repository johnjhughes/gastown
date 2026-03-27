package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/gastown/internal/tmux"
)

func TestEnsureMayorCommandContextLoadsPatrolSocketEnv(t *testing.T) {
	townRoot := t.TempDir()

	if err := os.MkdirAll(filepath.Join(townRoot, "mayor"), 0o755); err != nil {
		t.Fatalf("mkdir mayor: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(townRoot, "settings"), 0o755); err != nil {
		t.Fatalf("mkdir settings: %v", err)
	}

	writeFile := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(filepath.Join(townRoot, "mayor", "town.json"), `{"type":"town","version":2,"name":"gt"}`)
	writeFile(filepath.Join(townRoot, "mayor", "rigs.json"), `{"version":1,"rigs":{}}`)
	writeFile(filepath.Join(townRoot, "mayor", "daemon.json"), `{"type":"daemon-patrols","version":1,"env":{"GT_TMUX_SOCKET":"gt"}}`)

	origSocket := tmux.GetDefaultSocket()
	origTmuxSocket := os.Getenv("GT_TMUX_SOCKET")
	origTownRoot := os.Getenv("GT_TOWN_ROOT")
	origRoot := os.Getenv("GT_ROOT")
	defer func() {
		tmux.SetDefaultSocket(origSocket)
		_ = os.Setenv("GT_TMUX_SOCKET", origTmuxSocket)
		_ = os.Setenv("GT_TOWN_ROOT", origTownRoot)
		_ = os.Setenv("GT_ROOT", origRoot)
	}()

	tmux.SetDefaultSocket("")
	_ = os.Setenv("GT_TMUX_SOCKET", "")
	_ = os.Setenv("GT_TOWN_ROOT", "")
	_ = os.Setenv("GT_ROOT", "")

	ensureMayorCommandContext(townRoot)

	if got := os.Getenv("GT_TMUX_SOCKET"); got != "gt" {
		t.Fatalf("GT_TMUX_SOCKET = %q, want %q", got, "gt")
	}
	if got := os.Getenv("GT_TOWN_ROOT"); got != townRoot {
		t.Fatalf("GT_TOWN_ROOT = %q, want %q", got, townRoot)
	}
	if got := tmux.GetDefaultSocket(); got != "gt" {
		t.Fatalf("tmux.GetDefaultSocket() = %q, want %q", got, "gt")
	}
}
