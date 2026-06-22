package app

import (
	"strings"
	"testing"
)

func TestRepoURLs(t *testing.T) {
	cases := []struct{ in, ssh, https string }{
		{"clzmj", "git@github.com:clzmj/dotfiles.git", "https://github.com/clzmj/dotfiles.git"},
		{"clzmj/dots", "git@github.com:clzmj/dots.git", "https://github.com/clzmj/dots.git"},
		{"git@github.com:x/y.git", "git@github.com:x/y.git", "git@github.com:x/y.git"},
		{"https://github.com/x/y.git", "https://github.com/x/y.git", "https://github.com/x/y.git"},
	}
	for _, c := range cases {
		ssh, https := repoURLs(c.in)
		if ssh != c.ssh || https != c.https {
			t.Errorf("repoURLs(%q) = %q, %q; want %q, %q", c.in, ssh, https, c.ssh, c.https)
		}
	}
}

func TestAutoMessage(t *testing.T) {
	if got := autoMessage(""); got != "casa: update dotfiles" {
		t.Errorf("empty: got %q", got)
	}
	if got := autoMessage(" M a\n M b"); got != "casa: update a, b" {
		t.Errorf("two: got %q", got)
	}
	if got := autoMessage(" M a\n M b\n M c\n M d"); !strings.Contains(got, "and 1 more") {
		t.Errorf("many: got %q", got)
	}
}

func TestLooksSensitive(t *testing.T) {
	for _, s := range []string{"/x/.env", "/x/id_ed25519", "/x/foo.pem", "/x/credentials", "/x/api.key"} {
		if !looksSensitive(s) {
			t.Errorf("%q should be sensitive", s)
		}
	}
	for _, s := range []string{"/x/.zshrc", "/x/config.toml", "/x/.gitconfig"} {
		if looksSensitive(s) {
			t.Errorf("%q should NOT be sensitive", s)
		}
	}
}
