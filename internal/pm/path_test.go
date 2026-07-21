package pm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsurePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOBIN", "")
	t.Setenv("BUN_INSTALL", "")
	for _, d := range []string{"go/bin", ".cargo/bin"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", "/usr/bin:"+filepath.Join(home, ".cargo", "bin"))

	EnsurePath()

	path := os.Getenv("PATH")
	parts := strings.Split(path, string(os.PathListSeparator))
	if parts[len(parts)-2] != "/usr/bin" { // original tail preserved
		t.Fatalf("original PATH not preserved: %q", path)
	}
	if !strings.Contains(path, filepath.Join(home, "go", "bin")) {
		t.Errorf("go/bin not added: %q", path)
	}
	if strings.Count(path, filepath.Join(home, ".cargo", "bin")) != 1 {
		t.Errorf(".cargo/bin duplicated: %q", path)
	}
	if strings.Contains(path, filepath.Join(home, ".bun", "bin")) {
		t.Errorf("nonexistent .bun/bin added: %q", path)
	}
}
