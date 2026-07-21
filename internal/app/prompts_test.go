package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const canonicalTmpl = `
{{- writeToStdout "Configuring...\n" -}}
{{- $name  := promptStringOnce . "name" "Full name" -}}
{{- $email := promptString "Email address" "me@example.com" -}}
{{- $work  := promptBoolOnce . "work" "Work machine" false -}}
{{- $n     := promptIntOnce . "monitors" "How many monitors" 1 -}}
{{- $host  := promptChoiceOnce . "hosttype" "Host type" (list "desktop" "laptop" "server") "laptop" -}}
{{- $feats := promptMultichoiceOnce . "features" "Features" (list "docker" "k8s" "gpu") -}}

[data]
    name  = {{ $name | quote }}
    email = {{ $email | quote }}
`

func TestParseQuestions(t *testing.T) {
	qs := parseQuestions(canonicalTmpl)
	if len(qs) != 6 {
		t.Fatalf("got %d questions, want 6: %+v", len(qs), qs)
	}
	want := []question{
		{kind: "string", once: true, key: "name", text: "Full name"},
		{kind: "string", text: "Email address", def: "me@example.com", hasDef: true},
		{kind: "bool", once: true, key: "work", text: "Work machine", def: "false", hasDef: true},
		{kind: "int", once: true, key: "monitors", text: "How many monitors", def: "1", hasDef: true},
		{kind: "choice", once: true, key: "hosttype", text: "Host type",
			choices: []string{"desktop", "laptop", "server"}, def: "laptop", hasDef: true},
		{kind: "multichoice", once: true, key: "features", text: "Features",
			choices: []string{"docker", "k8s", "gpu"}},
	}
	for i := range want {
		if !reflect.DeepEqual(qs[i], want[i]) {
			t.Errorf("q[%d] = %+v, want %+v", i, qs[i], want[i])
		}
	}
}

func TestParseQuestionsIgnoresJunk(t *testing.T) {
	if qs := parseQuestions(`{{ .email }} plain text, no prompts`); len(qs) != 0 {
		t.Errorf("expected none, got %+v", qs)
	}
	// duplicate prompt text is asked once
	qs := parseQuestions(`{{ promptString "X" }} {{ promptString "X" }}`)
	if len(qs) != 1 {
		t.Errorf("dupes should collapse, got %+v", qs)
	}
}

func TestInitFlags(t *testing.T) {
	qs := parseQuestions(canonicalTmpl)
	ans := map[string]string{
		"Full name": "Ada", "Work machine": "true", "Host type": "server", "Features": "docker/gpu",
	}
	flags := initFlags(qs, ans, true)
	joined := strings.Join(flags, " ")
	for _, want := range []string{
		"--prompt",
		"--promptString Full name=Ada",
		"--promptBool Work machine=true",
		"--promptChoice Host type=server",
		"--promptMultichoice Features=docker/gpu",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("flags missing %q: %v", want, flags)
		}
	}
	if strings.Contains(joined, "monitors") || strings.Contains(joined, "Email") {
		t.Errorf("unanswered questions must not produce flags: %v", flags)
	}
}

func TestExprDefault(t *testing.T) {
	// the real-world case: default is a template expression, not a literal
	qs := parseQuestions(`{{- $m := promptStringOnce . "machine" "Machine name" .chezmoi.hostname -}}`)
	if len(qs) != 1 || !qs[0].defIsExpr || qs[0].def != ".chezmoi.hostname" {
		t.Fatalf("got %+v", qs)
	}
	data := map[string]any{"chezmoi": map[string]any{"hostname": "mbp"}}
	if v, ok := resolveDef(qs[0], data); !ok || v != "mbp" {
		t.Errorf("resolveDef = %q,%v want mbp,true", v, ok)
	}
	if v, ok := resolveDef(qs[0], map[string]any{}); ok || v != "" {
		t.Errorf("unresolvable expr must return ,false — got %q,%v", v, ok)
	}
	// literal defaults are never expressions
	qs = parseQuestions(`{{- $w := promptBoolOnce . "work" "Work?" false -}}`)
	if qs[0].defIsExpr {
		t.Errorf("false is a literal: %+v", qs[0])
	}
}

func TestCurrentValue(t *testing.T) {
	data := map[string]any{
		"work": true, "hosttype": "server", "features": []any{"docker", "gpu"}, "monitors": int64(2),
	}
	cases := []struct {
		q    question
		want string
		ok   bool
	}{
		{question{key: "work"}, "true", true},
		{question{key: "hosttype"}, "server", true},
		{question{key: "features"}, "docker/gpu", true},
		{question{key: "monitors"}, "2", true},
		{question{key: "missing"}, "", false},
		{question{key: ""}, "", false},
	}
	for _, c := range cases {
		got, ok := currentValue(data, c.q)
		if got != c.want || ok != c.ok {
			t.Errorf("currentValue(%q) = %q,%v want %q,%v", c.q.key, got, ok, c.want, c.ok)
		}
	}
}

func TestAttrsFromSourceName(t *testing.T) {
	cases := []struct {
		base string
		want []string
	}{
		{"dot_zshrc", nil},
		{"dot_gitconfig.tmpl", []string{"template"}},
		{"encrypted_private_github.age", []string{"encrypted", "private"}},
		{"encrypted_private_dot_netrc.tmpl.age", []string{"template", "encrypted", "private"}},
		{"executable_deploy.sh", []string{"executable"}},
	}
	for _, c := range cases {
		got := attrsFromSourceName(c.base)
		for _, a := range attrOpts {
			want := false
			for _, w := range c.want {
				if w == a {
					want = true
				}
			}
			if got[a] != want {
				t.Errorf("attrs(%q)[%s] = %v, want %v", c.base, a, got[a], want)
			}
		}
	}
}

func TestInsertQuestion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".casa.toml.tmpl")

	// creates the questionnaire from scratch
	q := question{kind: "string", once: true, key: "email", text: "Email address"}
	if err := insertQuestion(path, q); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	s := string(b)
	if !strings.Contains(s, `promptStringOnce . "email" "Email address"`) ||
		!strings.Contains(s, "[data]") || !strings.Contains(s, "email = {{ $email | quote }}") {
		t.Fatalf("scaffold wrong:\n%s", s)
	}

	// inserts into an existing [data] section
	q2 := question{kind: "choice", once: true, key: "host", text: "Host type",
		choices: []string{"a", "b"}}
	if err := insertQuestion(path, q2); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	s = string(b)
	if !strings.Contains(s, `promptChoiceOnce . "host" "Host type" (list "a" "b")`) {
		t.Fatalf("choice call wrong:\n%s", s)
	}
	if strings.Index(s, `"host"`) > strings.Index(s, "[data]") {
		t.Errorf("prompt call must sit above [data]:\n%s", s)
	}
	// what we generate must be parseable by our own parser
	qs := parseQuestions(s)
	if len(qs) != 2 {
		t.Errorf("round-trip parse got %d questions:\n%s", len(qs), s)
	}

	// bool + multichoice data lines
	if err := insertQuestion(path, question{kind: "bool", once: true, key: "work", text: "Work?"}); err != nil {
		t.Fatal(err)
	}
	if err := insertQuestion(path, question{kind: "multichoice", once: true, key: "fx", text: "Fx?",
		choices: []string{"x", "y"}}); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	s = string(b)
	if !strings.Contains(s, "work = {{ $work }}") {
		t.Errorf("bool data line wrong:\n%s", s)
	}
	if !strings.Contains(s, "fx = [{{ range $i, $v := $fx }}") {
		t.Errorf("multichoice data line wrong:\n%s", s)
	}
}
