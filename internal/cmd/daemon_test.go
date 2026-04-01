package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gofrs/flock"
)

func TestRunDaemonRun_SucceedsQuietlyWhenDaemonAlreadyRunning(t *testing.T) {
	townRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(townRoot, "mayor"), 0o755); err != nil {
		t.Fatalf("mkdir mayor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(townRoot, "mayor", "town.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write town.json: %v", err)
	}

	daemonDir := filepath.Join(townRoot, "daemon")
	if err := os.MkdirAll(daemonDir, 0o755); err != nil {
		t.Fatalf("mkdir daemon: %v", err)
	}

	lock := flock.New(filepath.Join(daemonDir, "daemon.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		t.Fatalf("TryLock() error = %v", err)
	}
	if !locked {
		t.Fatal("expected test lock acquisition to succeed")
	}
	defer func() { _ = lock.Unlock() }()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(townRoot); err != nil {
		t.Fatalf("Chdir(%q) error = %v", townRoot, err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if err := runDaemonRun(daemonRunCmd, nil); err != nil {
		t.Fatalf("runDaemonRun() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(daemonDir, "daemon.log")); !os.IsNotExist(err) {
		t.Fatalf("expected no daemon log on idempotent duplicate start, stat err = %v", err)
	}
}
