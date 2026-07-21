package pm

import (
	"slices"
	"testing"
)

func TestParseBunList(t *testing.T) {
	// real `bun pm ls -g` output: header + colored tree rows
	in := "/u/me/.bun/install/global node_modules (41)\n" +
		"\x1b[2m└──\x1b[0m cowsay\x1b[0m\x1b[2m@1.6.0\x1b[0m\n" +
		"\x1b[2m├──\x1b[0m @scope/tool\x1b[0m\x1b[2m@2.0.1\x1b[0m\n"
	got := parseBunList(in)
	want := []string{"cowsay", "@scope/tool"}
	if !slices.Equal(got, want) {
		t.Errorf("parseBunList = %v, want %v", got, want)
	}
	if got := parseBunList("error: Lockfile not found\n"); got != nil {
		t.Errorf("empty list = %v, want nil", got)
	}
}
