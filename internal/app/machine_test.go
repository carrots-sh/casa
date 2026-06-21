package app

import "testing"

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
