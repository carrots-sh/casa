package selfupdate

import "testing"

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"0.1.0", "v0.1.0", false},                  // same
		{"0.1.0", "v0.1.1", true},                   // newer patch
		{"0.1.9", "v0.2.0", true},                   // newer minor resets patch
		{"0.9.9", "v1.0.0", true},                   // newer major
		{"0.1.9", "v0.1.10", true},                  // parts compare numerically
		{"1.0.0", "v0.9.9", false},                  // ahead of latest
		{"dev", "v0.1.0", false},                    // dev build never flags
		{"v0.0.0-20260706-abcdef", "v0.1.0", false}, // go pseudo-version
		{"v2026.06.22-7", "v0.1.0", false},          // legacy date-based build
		{"0.1.0", "", false},                        // unknown latest
	}
	for _, c := range cases {
		if got := Newer(c.cur, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.cur, c.latest, got, c.want)
		}
	}
}
