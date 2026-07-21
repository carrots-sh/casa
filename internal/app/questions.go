// The setup-questions screens: change answers, ask the questionnaire during
// setup, and author new questions into the repo's config template.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

// Answers is the setup-questions screen: change one answer (or all of them)
// and re-render this machine's config through chezmoi's own init.
func Answers(name string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	qs, found, err := loadQuestions()
	if err != nil {
		return err
	}
	if !found || len(qs) == 0 {
		if found {
			return rerunPrompts() // questionnaire casa can't parse — chezmoi asks directly
		}
		fmt.Println("no setup questions yet — add one with: casa machine question")
		return nil
	}
	data, _ := chez.Data()

	const allLabel = "everything · ask all questions again"
	labels := []string{}
	byLabel := map[string]question{}
	for _, q := range qs {
		l := q.text
		if cur, ok := currentValue(data, q); ok {
			l += "   (" + cur + ")"
		}
		labels = append(labels, l)
		byLabel[l] = q
	}
	labels = append(labels, allLabel)

	var sel string
	if name != "" {
		var hits []string
		for _, l := range labels[:len(labels)-1] {
			q := byLabel[l]
			if strings.Contains(strings.ToLower(q.text+" "+q.key), strings.ToLower(name)) {
				hits = append(hits, l)
			}
		}
		switch len(hits) {
		case 1:
			sel = hits[0]
		case 0:
			return fmt.Errorf("no setup question matches %q", name)
		default:
			if sel, err = ui.Select("change which answer?", hits); err != nil || sel == "" {
				return err
			}
		}
	} else if sel, err = ui.Select("change which answer?", labels); err != nil || sel == "" {
		return err
	}

	ans := map[string]string{}
	for _, q := range qs {
		cur, has := currentValue(data, q)
		if sel == allLabel || byLabel[sel].text == q.text {
			pre, hasPre := cur, has
			if !hasPre {
				pre, hasPre = resolveDef(q, data)
			}
			v, err := askQuestion(q, pre, hasPre)
			if err != nil {
				return err
			}
			if v == "" && q.kind != "multichoice" && q.kind != "string" {
				return nil // backed out
			}
			ans[q.text] = v
		} else if has {
			ans[q.text] = cur // pass through unchanged so chezmoi won't re-ask
		}
	}
	fmt.Println("updating this machine's answers...")
	if err := chez.Init(initFlags(qs, ans, true)...); err != nil {
		return err
	}
	invalidateStatus()
	fmt.Println("applying...")
	return chez.Apply()
}

// askSetupQuestions runs the repo questionnaire in casa's UI and renders the
// machine config. Unparsed prompts fall through to chezmoi's own prompting.
func askSetupQuestions() error {
	qs, found, err := loadQuestions()
	if err != nil {
		return err
	}
	if !found {
		return chez.Init() // nothing to ask; still make sure a config exists
	}
	data, _ := chez.Data()
	ans := map[string]string{}
	for _, q := range qs {
		cur, has := currentValue(data, q)
		if !has {
			cur, has = resolveDef(q, data)
		}
		v, err := askQuestion(q, cur, has)
		if err != nil {
			return err
		}
		ans[q.text] = v
	}
	return chez.Init(initFlags(qs, ans, false)...)
}

var identRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// question kinds offered by AddQuestion, mapped to prompt functions.
var kindByLabel = map[string]string{
	"text":              "string",
	"yes / no":          "bool",
	"one of a list":     "choice",
	"several of a list": "multichoice",
	"number":            "int",
}

// AddQuestion appends a new setup question to the repo questionnaire and
// answers it for this machine right away.
func AddQuestion() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	key, err := ui.Input("data key (used as {{ .key }} in templates)")
	if err != nil || key == "" {
		return err
	}
	if !identRe.MatchString(key) {
		return fmt.Errorf("key must be letters, digits, or underscores, starting with a letter")
	}
	text, err := ui.Input("question to ask when setting up a machine")
	if err != nil || text == "" {
		return err
	}
	kindLabel, err := ui.Select("what kind of answer?",
		[]string{"text", "yes / no", "one of a list", "several of a list", "number"})
	if err != nil || kindLabel == "" {
		return err
	}
	q := question{kind: kindByLabel[kindLabel], once: true, key: key, text: text}
	if q.kind == "choice" || q.kind == "multichoice" {
		raw, err := ui.Input("choices (comma-separated)")
		if err != nil || raw == "" {
			return err
		}
		for c := range strings.SplitSeq(raw, ",") {
			if c = strings.TrimSpace(c); c != "" {
				q.choices = append(q.choices, c)
			}
		}
		if len(q.choices) < 2 {
			return fmt.Errorf("need at least two choices")
		}
	}

	path, ok := chez.ConfigTemplate()
	if !ok {
		path = filepath.Join(chez.SourceDir(), ".casa.toml.tmpl")
	}
	if err := insertQuestion(path, q); err != nil {
		return err
	}

	v, err := askQuestion(q, "", false)
	if err != nil {
		return err
	}
	if err := chez.Init(initFlags([]question{q}, map[string]string{q.text: v}, false)...); err != nil {
		return err
	}
	invalidateStatus()
	fmt.Printf("✓ added — use {{ .%s }} in any template\n", key)
	offerSave("casa: add setup question " + key)
	return nil
}

// insertQuestion writes the prompt line above [data] and the assignment below
// it, creating the questionnaire if needed.
func insertQuestion(path string, q question) error {
	fn := map[string]string{
		"string": "promptStringOnce", "bool": "promptBoolOnce", "int": "promptIntOnce",
		"choice": "promptChoiceOnce", "multichoice": "promptMultichoiceOnce",
	}[q.kind]
	call := fmt.Sprintf("%s . %q %q", fn, q.key, q.text)
	if len(q.choices) > 0 {
		quoted := make([]string, len(q.choices))
		for i, c := range q.choices {
			quoted[i] = strconv.Quote(c)
		}
		call += " (list " + strings.Join(quoted, " ") + ")"
	}
	// left-trim only: a right trim (-}}) would glue the previous rendered
	// line to whatever follows the insertion point and corrupt the TOML
	promptLine := fmt.Sprintf("{{- $%s := %s }}", q.key, call)

	var dataLine string
	switch q.kind {
	case "bool", "int":
		dataLine = fmt.Sprintf("    %s = {{ $%s }}", q.key, q.key)
	case "multichoice":
		dataLine = fmt.Sprintf(
			"    %s = [{{ range $i, $v := $%s }}{{ if $i }}, {{ end }}{{ $v | quote }}{{ end }}]",
			q.key, q.key)
	default:
		dataLine = fmt.Sprintf("    %s = {{ $%s | quote }}", q.key, q.key)
	}

	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return os.WriteFile(path,
			[]byte(promptLine+"\n\n[data]\n"+dataLine+"\n"), 0o644)
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == "[data]" {
			out := make([]string, 0, len(lines)+2)
			out = append(out, lines[:i]...)
			out = append(out, promptLine, lines[i], dataLine)
			out = append(out, lines[i+1:]...)
			return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
		}
	}
	out := strings.TrimRight(string(b), "\n") +
		"\n" + promptLine + "\n\n[data]\n" + dataLine + "\n"
	return os.WriteFile(path, []byte(out), 0o644)
}
