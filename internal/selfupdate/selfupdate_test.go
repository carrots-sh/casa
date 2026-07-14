package selfupdate

import "testing"

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"2026.06.22-7", "v2026.06.22-7", false},           // same
		{"2026.06.22-7", "v2026.06.23-0", true},            // newer day
		{"2026.06.22-7", "v2026.06.22-8", true},            // newer counter
		{"2026.06.22-9", "v2026.06.22-10", true},           // counters compare numerically
		{"2026.06.23-0", "v2026.06.22-7", false},           // ahead of latest
		{"dev", "v2026.06.22-7", false},                    // dev build never flags
		{"v0.0.0-20260706-abcdef", "v2026.06.22-7", false}, // pseudo-version
		{"2026.06.22-7", "", false},                        // unknown latest
	}
	for _, c := range cases {
		if got := Newer(c.cur, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.cur, c.latest, got, c.want)
		}
	}
}
