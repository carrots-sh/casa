package app

import (
	"os"
	"strconv"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/pm"
)

type statusInfo struct {
	machine string
	toSave  int // uncommitted changes in the repo
	behind  int // commits behind the remote
	drift   int // managed files differing from source (need apply)
	updates int // outdated packages
}

func computeStatus() statusInfo {
	var s statusInfo
	s.machine = machineName()
	if out, err := chez.GitOut("status", "--porcelain"); err == nil {
		s.toSave = countLines(out)
	}
	if out, err := chez.GitOut("rev-list", "--count", "HEAD..@{u}"); err == nil {
		s.behind = atoi(strings.TrimSpace(out))
	}
	if st, err := chez.Status(); err == nil {
		s.drift = len(st)
	}
	s.updates = len(pm.Outdated())
	return s
}

func machineName() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return strings.ToLower(strings.TrimSuffix(h, ".local"))
	}
	return "this machine"
}

func countLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
