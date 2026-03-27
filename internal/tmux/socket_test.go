package tmux

import (
	"os"
	"testing"
)

func TestSetGetDefaultSocket(t *testing.T) {
	// Save and restore
	orig := defaultSocket
	defer func() { defaultSocket = orig }()

	// Initially empty
	SetDefaultSocket("")
	if got := GetDefaultSocket(); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	SetDefaultSocket("mytown")
	if got := GetDefaultSocket(); got != "mytown" {
		t.Errorf("expected %q, got %q", "mytown", got)
	}
}

func TestNewTmuxInheritsSocket(t *testing.T) {
	orig := defaultSocket
	defer func() { defaultSocket = orig }()

	SetDefaultSocket("testtown")
	tmx := NewTmux()
	if tmx.socketName != "testtown" {
		t.Errorf("NewTmux() socketName = %q, want %q", tmx.socketName, "testtown")
	}
}

func TestNewTmuxWithSocket(t *testing.T) {
	tmx := NewTmuxWithSocket("custom")
	if tmx.socketName != "custom" {
		t.Errorf("NewTmuxWithSocket() socketName = %q, want %q", tmx.socketName, "custom")
	}
}

func TestBuildCommandNoSocket(t *testing.T) {
	orig := defaultSocket
	origTownSocket := os.Getenv("GT_TOWN_SOCKET")
	origTmuxSocket := os.Getenv("GT_TMUX_SOCKET")
	defer func() {
		defaultSocket = orig
		_ = os.Setenv("GT_TOWN_SOCKET", origTownSocket)
		_ = os.Setenv("GT_TMUX_SOCKET", origTmuxSocket)
	}()

	SetDefaultSocket("")
	_ = os.Setenv("GT_TOWN_SOCKET", "")
	_ = os.Setenv("GT_TMUX_SOCKET", "")
	cmd := BuildCommand("list-sessions")
	args := cmd.Args
	// Should be: tmux -u list-sessions
	expected := []string{"tmux", "-u", "list-sessions"}
	if len(args) != len(expected) {
		t.Fatalf("args = %v, want %v", args, expected)
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestBuildCommandWithSocket(t *testing.T) {
	orig := defaultSocket
	origTownSocket := os.Getenv("GT_TOWN_SOCKET")
	origTmuxSocket := os.Getenv("GT_TMUX_SOCKET")
	defer func() {
		defaultSocket = orig
		_ = os.Setenv("GT_TOWN_SOCKET", origTownSocket)
		_ = os.Setenv("GT_TMUX_SOCKET", origTmuxSocket)
	}()

	SetDefaultSocket("mytown")
	_ = os.Setenv("GT_TOWN_SOCKET", "")
	_ = os.Setenv("GT_TMUX_SOCKET", "")
	cmd := BuildCommand("has-session", "-t", "hq-mayor")
	args := cmd.Args
	// Should be: tmux -u -L mytown has-session -t hq-mayor
	expected := []string{"tmux", "-u", "-L", "mytown", "has-session", "-t", "hq-mayor"}
	if len(args) != len(expected) {
		t.Fatalf("args = %v, want %v", args, expected)
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestEffectiveSocketFallsBackToEnv(t *testing.T) {
	origSocket := defaultSocket
	origTownSocket := os.Getenv("GT_TOWN_SOCKET")
	origTmuxSocket := os.Getenv("GT_TMUX_SOCKET")
	defer func() {
		defaultSocket = origSocket
		_ = os.Setenv("GT_TOWN_SOCKET", origTownSocket)
		_ = os.Setenv("GT_TMUX_SOCKET", origTmuxSocket)
	}()

	SetDefaultSocket("")
	_ = os.Setenv("GT_TOWN_SOCKET", "town-binding")
	_ = os.Setenv("GT_TMUX_SOCKET", "town-shell")

	if got := EffectiveSocket(); got != "town-shell" {
		t.Fatalf("EffectiveSocket() = %q, want %q", got, "town-shell")
	}

	_ = os.Setenv("GT_TMUX_SOCKET", "")
	if got := EffectiveSocket(); got != "town-binding" {
		t.Fatalf("EffectiveSocket() with only GT_TOWN_SOCKET = %q, want %q", got, "town-binding")
	}
}

func TestBuildCommandUsesGTTmuxSocketFallback(t *testing.T) {
	origSocket := defaultSocket
	origTownSocket := os.Getenv("GT_TOWN_SOCKET")
	origTmuxSocket := os.Getenv("GT_TMUX_SOCKET")
	defer func() {
		defaultSocket = origSocket
		_ = os.Setenv("GT_TOWN_SOCKET", origTownSocket)
		_ = os.Setenv("GT_TMUX_SOCKET", origTmuxSocket)
	}()

	SetDefaultSocket("")
	_ = os.Setenv("GT_TOWN_SOCKET", "")
	_ = os.Setenv("GT_TMUX_SOCKET", "gt")

	cmd := BuildCommand("list-sessions")
	expected := []string{"tmux", "-u", "-L", "gt", "list-sessions"}
	if len(cmd.Args) != len(expected) {
		t.Fatalf("args = %v, want %v", cmd.Args, expected)
	}
	for i, a := range cmd.Args {
		if a != expected[i] {
			t.Fatalf("args[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestIsInSameSocketUsesGTTmuxSocketFallback(t *testing.T) {
	origSocket := defaultSocket
	origTownSocket := os.Getenv("GT_TOWN_SOCKET")
	origTmuxSocket := os.Getenv("GT_TMUX_SOCKET")
	origTMUX := os.Getenv("TMUX")
	defer func() {
		defaultSocket = origSocket
		_ = os.Setenv("GT_TOWN_SOCKET", origTownSocket)
		_ = os.Setenv("GT_TMUX_SOCKET", origTmuxSocket)
		_ = os.Setenv("TMUX", origTMUX)
	}()

	SetDefaultSocket("")
	_ = os.Setenv("GT_TOWN_SOCKET", "")
	_ = os.Setenv("GT_TMUX_SOCKET", "gt")
	_ = os.Setenv("TMUX", "/tmp/tmux-1000/gt,123,0")

	if !IsInSameSocket() {
		t.Fatal("IsInSameSocket() = false, want true")
	}
}
