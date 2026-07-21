package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fresh(t *testing.T) Manifest {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, DefaultRel)
	if _, err := Bootstrap(dir, p); err != nil {
		t.Fatal(err)
	}
	return Manifest{Path: p}
}

func TestBootstrapCreatesFilesOnce(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, DefaultRel)
	created, err := Bootstrap(dir, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 3 {
		t.Fatalf("want 3 created files, got %v", created)
	}
	for _, s := range []string{ScriptPackages, ScriptShTools} {
		if _, err := os.Stat(filepath.Join(dir, s)); err != nil {
			t.Fatalf("missing %s: %v", s, err)
		}
	}
	again, err := Bootstrap(dir, p)
	if err != nil || len(again) != 0 {
		t.Fatalf("second bootstrap should create nothing, got %v (%v)", again, err)
	}
}

func TestAddListRemoveRoundTrip(t *testing.T) {
	m := fresh(t)
	for _, n := range []string{"jq", "ripgrep"} {
		if err := m.Add("brew", n); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Add("brew", "jq"); err != nil { // idempotent
		t.Fatal(err)
	}
	got, err := m.List("brew")
	if err != nil || strings.Join(got, ",") != "jq,ripgrep" {
		t.Fatalf("list = %v (%v)", got, err)
	}
	if err := m.Remove("brew", "jq"); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("brew", "never-there"); err != nil { // idempotent
		t.Fatal(err)
	}
	got, _ = m.List("brew")
	if strings.Join(got, ",") != "ripgrep" {
		t.Fatalf("after remove: %v", got)
	}
}

func TestCommentsAndHandEditsSurvive(t *testing.T) {
	m := fresh(t)
	data, _ := os.ReadFile(m.Path)
	// simulate a hand edit: entry with a trailing comment + inline array
	s := strings.Replace(string(data), "brew = [\n]", "brew = [\n  \"fd\", # fast find\n]", 1)
	s = strings.Replace(s, "npm = [\n]", "npm = [\"typescript\"]", 1)
	os.WriteFile(m.Path, []byte(s), 0o644)

	if err := m.Add("brew", "jq"); err != nil {
		t.Fatal(err)
	}
	if err := m.Add("npm", "prettier"); err != nil { // inline → multiline rewrite
		t.Fatal(err)
	}
	if err := m.Remove("brew", "fd"); err != nil { // entry with trailing comment
		t.Fatal(err)
	}
	out, _ := os.ReadFile(m.Path)
	text := string(out)
	if !strings.Contains(text, "# CLI tools — cross-platform") {
		t.Fatal("section comment lost")
	}
	brew, _ := m.List("brew")
	npm, _ := m.List("npm")
	if strings.Join(brew, ",") != "jq" || strings.Join(npm, ",") != "typescript,prettier" {
		t.Fatalf("brew=%v npm=%v", brew, npm)
	}
}

func TestShToolBlocks(t *testing.T) {
	m := fresh(t)
	a := ShTool{Bin: "herdr", Install: `curl -fsSL https://herdr.dev/install.sh | sh`, Update: "herdr self-update"}
	b := ShTool{Bin: "zed", Install: `curl -f https://zed.dev/install.sh | sh`, OS: "darwin"}
	for _, tool := range []ShTool{a, b, a} { // third add is a dup → no-op
		if err := m.AddSh(tool); err != nil {
			t.Fatal(err)
		}
	}
	tools, err := m.ShTools()
	if err != nil || len(tools) != 2 {
		t.Fatalf("tools = %+v (%v)", tools, err)
	}
	if tools[0] != a || tools[1] != b {
		t.Fatalf("round-trip mismatch: %+v", tools)
	}
	if err := m.RemoveSh("herdr"); err != nil {
		t.Fatal(err)
	}
	tools, _ = m.ShTools()
	if len(tools) != 1 || tools[0].Bin != "zed" {
		t.Fatalf("after remove: %+v", tools)
	}
	// pm sections still intact after block surgery
	if err := m.Add("brew", "jq"); err != nil {
		t.Fatal(err)
	}
	brew, _ := m.List("brew")
	if len(brew) != 1 {
		t.Fatalf("brew = %v", brew)
	}
}

func TestMissingSectionIsCreated(t *testing.T) {
	m := fresh(t)
	data, _ := os.ReadFile(m.Path)
	s := strings.Replace(string(data), "cargo = [\n]", "", 1)
	os.WriteFile(m.Path, []byte(s), 0o644)
	if err := m.Add("cargo", "eza"); err != nil {
		t.Fatal(err)
	}
	got, _ := m.List("cargo")
	if strings.Join(got, ",") != "eza" {
		t.Fatalf("cargo = %v", got)
	}
}
