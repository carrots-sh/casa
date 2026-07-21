package home

import (
	"path/filepath"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	t.Setenv("HOME", "/u/test")
	cases := []struct{ in, expand, tilde string }{
		{"~/.zshrc", "/u/test/.zshrc", "~/.zshrc"},
		{"~/x/", "/u/test/x/", "~/x/"}, // trailing slash survives
		{"~", "/u/test", "~"},
		{"/u/test/.aws/config", "/u/test/.aws/config", "~/.aws/config"},
		{".aws/config", ".aws/config", "~/.aws/config"},
		{"/etc/hosts", "/etc/hosts", "/etc/hosts"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := Expand(c.in); got != c.expand {
			t.Errorf("Expand(%q) = %q, want %q", c.in, got, c.expand)
		}
		if got := Tilde(c.in); got != c.tilde {
			t.Errorf("Tilde(%q) = %q, want %q", c.in, got, c.tilde)
		}
	}
	if got := Path(".zshrc"); got != filepath.Join("/u/test", ".zshrc") {
		t.Errorf("Path = %q", got)
	}
}
