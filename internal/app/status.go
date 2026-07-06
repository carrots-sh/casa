package app

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/pm"
)

type statusInfo struct {
	machine string
	toSave  int // uncommitted changes in the repo
	behind  int // commits behind the remote
	updates int // outdated packages (-1 = not computed yet)
}

var (
	stMu       sync.Mutex
	cheapCache *statusInfo // git-derived hints (fast); nil = stale
	updCache   = -1        // brew/npm outdated count; -1 = unknown
	updBusy    bool        // a background outdated check is running
)

// invalidateStatus clears the cached hints after an action changes state.
func invalidateStatus() {
	stMu.Lock()
	cheapCache = nil
	updCache = -1
	stMu.Unlock()
}

// computeStatus returns the menu hints. The git-derived parts are cheap and
// cached; the slow `outdated` count is computed once in the background so the
// menu paints instantly (the count appears on a later render).
func computeStatus() statusInfo {
	stMu.Lock()
	if updCache == -1 && !updBusy {
		updBusy = true
		go func() {
			n := len(pm.Outdated())
			stMu.Lock()
			updCache, updBusy = n, false
			stMu.Unlock()
		}()
	}
	upd := updCache
	cached := cheapCache
	stMu.Unlock()

	if cached != nil {
		s := *cached
		s.updates = upd
		return s
	}

	var s statusInfo
	s.machine = machineName()
	if out, err := chez.GitOut("status", "--porcelain"); err == nil {
		s.toSave = len(chez.NonEmpty(out))
	}
	if out, err := chez.GitOut("rev-list", "--count", "HEAD..@{u}"); err == nil {
		s.behind, _ = strconv.Atoi(strings.TrimSpace(out))
	}
	stMu.Lock()
	cp := s
	cheapCache = &cp
	stMu.Unlock()

	s.updates = upd
	return s
}

// driftCount is the chezmoi status (managed files needing apply). Used only by
// the full `status` command, not the menu, since it's comparatively slow.
func driftCount() int {
	st, _ := chez.Status()
	return len(st)
}

// outdatedCount returns the package-update count, computing synchronously if the
// background check hasn't finished yet (for the explicit `status` command).
func outdatedCount() int {
	stMu.Lock()
	n := updCache
	stMu.Unlock()
	if n < 0 {
		n = len(pm.Outdated())
		stMu.Lock()
		updCache = n
		stMu.Unlock()
	}
	return n
}

func machineName() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return strings.ToLower(strings.TrimSuffix(h, ".local"))
	}
	return "this machine"
}
