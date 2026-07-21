// Package selfupdate upgrades the running casa binary from GitHub releases.
package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const repo = "carrots-sh/casa"

var client = &http.Client{Timeout: 60 * time.Second}

// Latest returns the newest release tag, e.g. "v2026.06.22-7".
func Latest() (string, error) {
	resp, err := client.Get("https://api.github.com/repos/" + repo + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github: %s", resp.Status)
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	if r.TagName == "" {
		return "", fmt.Errorf("github: no releases")
	}
	return r.TagName, nil
}

// Newer reports whether latest is a strictly newer release than current.
// Dev builds (anything that isn't a vYYYY.MM.DD-N version) never compare newer.
func Newer(current, latest string) bool {
	cd, cn, cok := parseVersion(current)
	ld, ln, lok := parseVersion(latest)
	if !cok || !lok {
		return false
	}
	if cd != ld {
		return ld > cd
	}
	return ln > cn
}

// parseVersion splits casa's date-based version ("v2026.06.22-7" or
// "2026.06.22-7") into its date and counter.
func parseVersion(v string) (date string, n int, ok bool) {
	v = strings.TrimPrefix(v, "v")
	date, c, found := strings.Cut(v, "-")
	if !found || len(date) != len("2026.06.22") || !strings.HasPrefix(date, "20") {
		return "", 0, false
	}
	n, err := strconv.Atoi(c)
	return date, n, err == nil
}

// Upgrade downloads the release build for this OS/arch and atomically replaces
// the running binary. Returns the path that was replaced.
func Upgrade(tag string) (string, error) {
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("self-update isn't supported on windows — grab the release from github")
	}
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/casa_%s_%s_%s.tar.gz",
		repo, tag, strings.TrimPrefix(tag, "v"), runtime.GOOS, runtime.GOARCH)
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: %s", url, resp.Status)
	}
	bin, err := extractBinary(resp.Body)
	if err != nil {
		return "", err
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, bin, 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return exe, nil
}

// extractBinary pulls the casa binary out of a release tar.gz stream.
func extractBinary(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("no casa binary in the release archive")
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(h.Name) == "casa" && h.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
}

// LatestThrottled is Latest with a 24h on-disk cache, for background checks
// that shouldn't hit the network on every run. Returns "" when unknown.
func LatestThrottled() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(dir, "casa", "latest")
	if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) < 24*time.Hour {
		b, _ := os.ReadFile(p)
		return strings.TrimSpace(string(b))
	}
	tag, err := Latest()
	if err != nil {
		return ""
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(tag+"\n"), 0o644)
	return tag
}
