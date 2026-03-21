package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSortsSQLFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "000002_second.sql", "-- second")
	writeTestFile(t, dir, "000001_first.sql", "-- first")
	writeTestFile(t, dir, "README.md", "ignored")

	got, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(Discover()) = %d, want %d", len(got), 2)
	}

	if got[0].Name != "000001_first.sql" {
		t.Fatalf("got[0].Name = %q, want %q", got[0].Name, "000001_first.sql")
	}
	if got[1].Name != "000002_second.sql" {
		t.Fatalf("got[1].Name = %q, want %q", got[1].Name, "000002_second.sql")
	}
}

func TestDiscoverReturnsErrorForMissingDirectory(t *testing.T) {
	if _, err := Discover(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Discover() error = nil, want non-nil")
	}
}

func writeTestFile(t *testing.T, dir string, name string, content string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
