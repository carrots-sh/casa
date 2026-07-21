package app

import "testing"

func TestParseInstallCommand(t *testing.T) {
	cases := []struct{ in, mgr, pkg string }{
		{"go install golang.org/x/tools/gopls@latest", "go", "golang.org/x/tools/gopls"},
		{"go get github.com/x/y", "go", "github.com/x/y"},
		{"cargo install eza", "cargo", "eza"},
		{"cargo install --locked eza", "cargo", "eza"},
		{"npm install -g typescript", "npm", "typescript"},
		{"npm i -g prettier", "npm", "prettier"},
		{"sudo npm install --global tsx", "npm", "tsx"},
		{"npm install typescript", "", ""}, // not global — not casa's business
		{"uv tool install ruff", "uv", "ruff"},
		{"bun add -g @anthropic-ai/claude-code", "bun", "@anthropic-ai/claude-code"},
		{"bun install -g eslint", "bun", "eslint"},
		{"bun add typescript", "", ""}, // not global
		{"brew install jq", "brew", "jq"},
		{"brew install --cask ghostty", "cask", "ghostty"},
		{"brew tap carrots-sh/tap", "tap", "carrots-sh/tap"},
		{"curl -fsSL https://herdr.dev/install.sh | sh", "sh", ""},
		{"wget -qO- https://x.dev/i.sh | bash", "sh", ""},
		{`sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"`, "sh", ""},
		{`NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://brew.sh/install.sh)"`, "sh", ""},
		{"pip install requests", "", ""},
		{"make install", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		mgr, pkg := parseInstallCommand(c.in)
		if mgr != c.mgr || pkg != c.pkg {
			t.Errorf("parseInstallCommand(%q) = (%q, %q), want (%q, %q)", c.in, mgr, pkg, c.mgr, c.pkg)
		}
	}
}
