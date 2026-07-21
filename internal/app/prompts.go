// The setup-questionnaire parser: casa reads the repo's config template
// (.casa.toml.tmpl) for chezmoi prompt* calls, asks them in casa's own UI, and
// answers chezmoi non-interactively via init --promptString/Bool/… flags.
// chezmoi still does all the rendering, so template semantics stay native, and
// any prompt casa fails to parse simply falls through to chezmoi's own
// terminal prompting.
package app

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

// question is one prompt* call found in the config template.
type question struct {
	kind      string // string | bool | int | choice | multichoice
	once      bool
	key       string // [data] key (*Once only)
	text      string // prompt text — chezmoi matches --promptX answers by this
	choices   []string
	def       string
	hasDef    bool
	defIsExpr bool // def is a template expression like .chezmoi.hostname, not a literal
}

var promptCall = regexp.MustCompile(`prompt(String|Bool|Int|Choice|Multichoice)(Once)?\b`)

// parseQuestions extracts every prompt* call from a config template.
func parseQuestions(tmpl string) []question {
	var qs []question
	seen := map[string]bool{}
	for _, loc := range promptCall.FindAllStringSubmatchIndex(tmpl, -1) {
		kind := strings.ToLower(tmpl[loc[2]:loc[3]])
		q := question{kind: kind, once: loc[4] >= 0}
		args := parseArgs(tmpl[loc[1]:])
		i := 0
		if q.once { // MAP then PATH come first
			if i < len(args) && args[i].typ == argDot {
				i++
			}
			if i < len(args) && args[i].typ == argStr {
				q.key = args[i].val
				i++
			}
		}
		if i >= len(args) || args[i].typ != argStr {
			continue
		}
		q.text = args[i].val
		i++
		if kind == "choice" || kind == "multichoice" {
			if i >= len(args) || args[i].typ != argList {
				continue
			}
			q.choices = args[i].list
			i++
		}
		if i < len(args) {
			q.def, q.hasDef = args[i].val, true
			q.defIsExpr = args[i].typ != argStr && !isLiteral(args[i].val)
		}
		if q.text == "" || seen[q.text] {
			continue
		}
		seen[q.text] = true
		qs = append(qs, q)
	}
	return qs
}

// template-argument tokens
const (
	argDot  = iota // . or .path
	argStr         // "quoted"
	argList        // (list "a" "b")
	argLit         // true, 42, $var, other parenthesized exprs
)

type tmplArg struct {
	typ  int
	val  string
	list []string
}

var quoted = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)

// parseArgs tokenizes the space-separated args after a prompt call, stopping
// at the end of the template action or a pipe.
func parseArgs(s string) []tmplArg {
	var out []tmplArg
	i := 0
	for i < len(s) {
		switch {
		case s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r':
			i++
		case strings.HasPrefix(s[i:], "}}") || strings.HasPrefix(s[i:], "-}}") ||
			s[i] == '|' || s[i] == ')':
			return out
		case s[i] == '"':
			if m := quoted.FindStringSubmatch(s[i:]); m != nil {
				v, err := strconv.Unquote(m[0])
				if err != nil {
					v = m[1]
				}
				out = append(out, tmplArg{typ: argStr, val: v})
				i += len(m[0])
			} else {
				return out
			}
		case s[i] == '(':
			body, n := readParen(s[i:])
			i += n
			if strings.HasPrefix(strings.TrimSpace(body), "list") {
				var items []string
				for _, m := range quoted.FindAllStringSubmatch(body, -1) {
					v, err := strconv.Unquote(m[0])
					if err != nil {
						v = m[1]
					}
					items = append(items, v)
				}
				out = append(out, tmplArg{typ: argList, list: items})
			} else {
				out = append(out, tmplArg{typ: argLit, val: body})
			}
		case s[i] == '.':
			j := bareEnd(s, i)
			out = append(out, tmplArg{typ: argDot, val: s[i:j]})
			i = j
		default:
			j := bareEnd(s, i)
			out = append(out, tmplArg{typ: argLit, val: s[i:j]})
			i = j
		}
	}
	return out
}

func bareEnd(s string, i int) int {
	j := i
	for j < len(s) && !strings.ContainsRune(" \t\n\r)}|", rune(s[j])) {
		j++
	}
	return j
}

// readParen returns the body inside a balanced ( … ) (quotes respected) and
// how many bytes were consumed.
func readParen(s string) (string, int) {
	depth, inStr := 0, false
	for i := 0; i < len(s); i++ {
		switch {
		case inStr:
			if s[i] == '\\' {
				i++
			} else if s[i] == '"' {
				inStr = false
			}
		case s[i] == '"':
			inStr = true
		case s[i] == '(':
			depth++
		case s[i] == ')':
			depth--
			if depth == 0 {
				return s[1:i], i + 1
			}
		}
	}
	return s[1:], len(s)
}

// promptFlag maps a question kind to its chezmoi init flag.
var promptFlag = map[string]string{
	"string":      "--promptString",
	"bool":        "--promptBool",
	"int":         "--promptInt",
	"choice":      "--promptChoice",
	"multichoice": "--promptMultichoice",
}

// initFlags renders collected answers as chezmoi init flags. force adds
// --prompt so *Once questions re-evaluate instead of reusing stored values.
func initFlags(qs []question, ans map[string]string, force bool) []string {
	var flags []string
	if force {
		flags = append(flags, "--prompt")
	}
	for _, q := range qs {
		if v, ok := ans[q.text]; ok {
			flags = append(flags, promptFlag[q.kind], q.text+"="+v)
		}
	}
	return flags
}

// isLiteral reports whether a bare template arg is a plain value (true, 42)
// rather than an expression ($var, .chezmoi.hostname, printf …).
func isLiteral(s string) bool {
	if s == "true" || s == "false" {
		return true
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

// resolveDef resolves a question's default: literals pass through, dotted
// expressions (.chezmoi.hostname) are looked up in the template data.
func resolveDef(q question, data map[string]any) (string, bool) {
	if !q.hasDef {
		return "", false
	}
	if !q.defIsExpr {
		return q.def, true
	}
	if !strings.HasPrefix(q.def, ".") {
		return "", false
	}
	cur := any(data)
	for part := range strings.SplitSeq(strings.TrimPrefix(q.def, "."), ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		if cur, ok = m[part]; !ok {
			return "", false
		}
	}
	return fmt.Sprint(cur), true
}

// askQuestion prompts one question in casa's UI. cur (when set) wins over the
// template default as the prefill. Empty string means the user backed out.
func askQuestion(q question, cur string, hasCur bool) (string, error) {
	def := ""
	if q.hasDef && !q.defIsExpr {
		def = q.def
	}
	if hasCur {
		def = cur
	}
	switch q.kind {
	case "bool":
		b, err := ui.ConfirmDefault(q.text, def == "true")
		if err != nil {
			return "", err
		}
		return strconv.FormatBool(b), nil
	case "choice":
		fmt.Println(q.text) // Select hides its title while filtering — show the question
		return ui.SelectDefault(q.text, q.choices, def)
	case "multichoice":
		var preset []string
		if def != "" {
			preset = strings.Split(def, "/")
		}
		sel, err := ui.MultiSelect(q.text, q.choices, preset...)
		return strings.Join(sel, "/"), err
	case "int":
		for {
			v, err := ui.InputDefault(q.text, def)
			if err != nil || v == "" {
				return v, err
			}
			if _, e := strconv.Atoi(v); e == nil {
				return v, nil
			}
			fmt.Println("please enter a whole number")
		}
	default:
		return ui.InputDefault(q.text, def)
	}
}

// currentValue looks a question's stored answer up in the template data.
func currentValue(data map[string]any, q question) (string, bool) {
	if q.key == "" || data == nil {
		return "", false
	}
	v, ok := data[q.key]
	if !ok {
		return "", false
	}
	switch t := v.(type) {
	case []any:
		var ss []string
		for _, e := range t {
			ss = append(ss, fmt.Sprint(e))
		}
		return strings.Join(ss, "/"), true
	default:
		return fmt.Sprint(v), true
	}
}

// loadQuestions parses the repo's setup questionnaire, if there is one.
func loadQuestions() ([]question, bool, error) {
	path, ok := chez.ConfigTemplate()
	if !ok {
		return nil, false, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, true, err
	}
	return parseQuestions(string(b)), true, nil
}
