#!/usr/bin/env bash
# End-to-end test: exercises every casa action against a sandboxed HOME, a
# real chezmoi, a real git remote (local bare repo), and real age encryption.
# Interactive forms are driven through a pty with expect(1).
#
# Requires: go, git, chezmoi, expect. Never touches your real HOME or repos.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SB="$(mktemp -d -t casa-e2e)"
trap 'rm -rf "$SB"' EXIT

step() { printf '\n\033[1m== %s\033[0m\n' "$*"; }
fail() { printf '\033[31me2e FAILED: %s\033[0m\n' "$*"; exit 1; }

# ---- sandbox ---------------------------------------------------------------
mkdir -p "$SB/bin" "$SB/home"
go build -o "$SB/bin/casa" "$ROOT/cmd/casa"
ln -s "$(command -v chezmoi)" "$SB/bin/chezmoi"
ln -s "$(command -v git)" "$SB/bin/git"
ln -s "$(command -v age)" "$SB/bin/age"
ln -s "$(command -v age-keygen)" "$SB/bin/age-keygen"
# a non-interactive $EDITOR that appends a marker line and exits
printf '#!/bin/sh\necho "e2e-appended-line" >> "$1"\n' > "$SB/bin/ed-append"
chmod +x "$SB/bin/ed-append"

export HOME="$SB/home"
export CASA_SOURCE="$SB/clone"
export EDITOR="$SB/bin/ed-append"
unset VISUAL GIT_EDITOR 2>/dev/null || true # would outrank $EDITOR in chezmoi/git
export TERM=xterm-256color
# ponytail: brew intentionally NOT on PATH so tools/sync never touch real packages
export PATH="$SB/bin:/usr/bin:/bin"
export CASA_PLAIN_PATH=1 # keep the mask: casa must not re-add real manager dirs
export GIT_CONFIG_GLOBAL="$SB/gitconfig"
git config --file "$GIT_CONFIG_GLOBAL" user.email e2e@casa.test
git config --file "$GIT_CONFIG_GLOBAL" user.name "casa e2e"
git config --file "$GIT_CONFIG_GLOBAL" init.defaultBranch main

# ---- seed dotfiles repo (casa-named special files) -------------------------
step "seed repo + bare origin"
chezmoi age-keygen --output="$HOME/key.txt" >/dev/null 2>&1
RECIPIENT="$(grep 'public key' "$HOME/key.txt" | awk '{print $NF}')"
SRC="$SB/seed"
mkdir -p "$SRC"
cat > "$SRC/.casa.toml.tmpl" <<EOF
{{- \$machine := promptStringOnce . "machine" "Machine name" .chezmoi.hostname -}}
{{- \$email   := promptStringOnce . "email" "Email" "e2e@example.com" -}}
{{- \$work    := promptBoolOnce . "work" "Work machine" false -}}
{{- \$host    := promptChoiceOnce . "hosttype" "Host type" (list "desktop" "laptop" "server") "laptop" -}}
{{- \$feats   := promptMultichoiceOnce . "features" "Features" (list "docker" "k8s" "gpu") -}}
encryption = "age"
[age]
    identity  = "$HOME/key.txt"
    recipient = "$RECIPIENT"
[data]
    machine   = {{ \$machine | quote }}
    email     = {{ \$email | quote }}
    work      = {{ \$work }}
    hosttype  = {{ \$host | quote }}
    features  = [{{ range \$i, \$f := \$feats }}{{ if \$i }}, {{ end }}{{ \$f | quote }}{{ end }}]
EOF
printf 'README.md\n' > "$SRC/.casaignore"
printf 'hello e2e\n' > "$SRC/dot_testrc"
printf '[user]\n\temail = {{ .email }}\n' > "$SRC/dot_gitconfig.tmpl"
printf 'seed readme\n' > "$SRC/README.md"
git -C "$SRC" init -q && git -C "$SRC" add -A && git -C "$SRC" commit -qm seed
git clone -q --bare "$SRC" "$SB/origin.git"

# ---- expect driver ----------------------------------------------------------
# exp <label> <<'EOF' ... EOF — tcl body with `must <text>` / `hit <keys>` helpers
exp() {
  local label="$1" script
  script="$(cat)"
  step "$label"
  expect -c '
set timeout 30
set stty_init "rows 40 columns 120"
proc must {pat} {
  expect {
    -ex $pat {}
    timeout { puts stderr "\ne2e: timed out waiting for: $pat"; exit 1 }
    eof     { puts stderr "\ne2e: eof before: $pat"; exit 1 }
  }
}
proc hit {keys} { sleep 0.3; send -- $keys }
'"$script"'
expect eof
' || fail "$label"
}

# ---- 1. version + help ------------------------------------------------------
step "version + help"
casa version | grep "casa" >/dev/null || fail "version"
casa help | grep "interactive menu" >/dev/null || fail "help"

# ---- 2. machine setup (clone + questionnaire in casa UI) --------------------
exp "machine setup — full questionnaire" <<EOF
spawn casa machine setup file://$SB/origin.git
must "Machine name";  hit "\r"
must "Email";         hit "\r"
must "Work machine";  hit "\r"
must "Host type";     hit "\r"
must "Features";      hit " "; hit "\r"
must "Homebrew";      hit "n"
must "applying your dotfiles"
EOF
grep -q "hello e2e" "$HOME/.testrc" || fail "setup did not apply .testrc"
grep -q "email = e2e@example.com" "$HOME/.gitconfig" || fail "template did not render email"
grep -q 'hosttype  = "laptop"' "$HOME/.config/chezmoi/chezmoi.toml" || fail "choice default not stored"
grep -q '"docker"' "$HOME/.config/chezmoi/chezmoi.toml" || fail "multichoice pick not stored"
[ ! -e "$HOME/README.md" ] || fail ".casaignore not honored"

# ---- 3. casa-named files are mirrored for chezmoi ---------------------------
step "mirror: casa-named files linked + gitignored"
[ -L "$CASA_SOURCE/.chezmoi.toml.tmpl" ] || fail "questionnaire not mirrored"
[ -L "$CASA_SOURCE/.chezmoiignore" ] || fail "ignore file not mirrored"
grep -q ".chezmoi.toml.tmpl" "$CASA_SOURCE/.gitignore" || fail "mirror not gitignored"

# ---- 4. configs list (badges) -----------------------------------------------
step "configs list + badges"
casa configs list | grep "^~/.testrc$" >/dev/null || fail "list missing ~/.testrc"
casa configs list | grep "~/.gitconfig  (template)" >/dev/null || fail "missing template badge"

# ---- 5. track: plain / template / encrypted defaults ------------------------
printf 'just plain text\n' > "$HOME/.plainrc"
exp "track — plain (default)" <<EOF
spawn casa configs track $HOME/.plainrc
must "same on every machine"; hit "\r"
must "now managing"
must "saved + pushed"
EOF
[ -f "$CASA_SOURCE/dot_plainrc" ] || fail "plain source missing"

printf 'email = e2e@example.com\n' > "$HOME/.tplrc"
exp "track — template default via data-value heuristic" <<EOF
spawn casa configs track $HOME/.tplrc
must "same on every machine"; hit "\r"
must "now managing"
must "saved + pushed"
EOF
grep -q "{{ .email }}" "$CASA_SOURCE/dot_tplrc.tmpl" || fail "autotemplate did not substitute"

printf 's3cr3t\n' > "$HOME/.apitoken.key"
exp "track — encrypted default via sensitive-name heuristic" <<EOF
spawn casa configs track $HOME/.apitoken.key
must "same on every machine"; hit "\r"
must "now managing"
must "saved + pushed"
EOF
ls "$CASA_SOURCE" | grep "encrypted_dot_apitoken.key.age" >/dev/null || fail "encrypted source missing"

# ---- 6. secrets: add / list / edit (with re-encrypt) -------------------------
step "secrets add + list"
printf 'tok-123\n' > "$HOME/.extra.token"
casa secrets add "$HOME/.extra.token" | grep "encrypted with main and now managing" >/dev/null || fail "secrets add (adopts ~/key.txt as main)"
casa secrets list | grep ".apitoken.key" >/dev/null || fail "secrets list"

exp "secrets edit — decrypt, edit, re-encrypt" <<EOF
spawn casa secrets edit .apitoken.key
must "updated secret"
EOF
chezmoi --source "$CASA_SOURCE" cat "$HOME/.apitoken.key" | grep -q "e2e-appended-line" \
  || fail "secret edit did not persist"

# ---- 6b. keys: create second, encrypt with it, delete → orphan re-encrypt ----
exp "keys — create a second key" <<EOF
spawn casa secrets keys
must "generate in"
sleep 0.3; send "new"; sleep 0.3; send "\r"
must "key name"; sleep 0.3; send "vault\r"
must "created vault"
must "generate in"
sleep 0.3; send "\x1b"
EOF
[ -f "$HOME/.config/casa/keys/vault.txt" ] || fail "vault key file missing"

printf 'vault-secret\n' > "$HOME/.vault.token"
exp "secrets add — pick the non-default key" <<EOF
spawn casa secrets add $HOME/.vault.token
must "★ default"
sleep 0.3; send "vault"; sleep 0.3; send "\r"
must "encrypted with vault"
must "saved + pushed"
EOF
# main's identity must NOT open it; the registry-driven config must
age --decrypt --identity "$HOME/.config/casa/keys/main.txt" \
  "$CASA_SOURCE/encrypted_dot_vault.token.age" >/dev/null 2>&1 \
  && fail "vault file readable by main (wrong key used)"
chezmoi --source "$CASA_SOURCE" cat "$HOME/.vault.token" | grep -q vault-secret \
  || fail "vault-encrypted file not decryptable via config identities"

exp "keys — backup to repo with passphrase" <<EOF
spawn casa secrets keys
must "generate in"
sleep 0.3; send "vault"; sleep 0.3; send "\r"
must "backup to repo"
sleep 0.3; send "backup"; sleep 0.3; send "\r"
must "passphrase"
sleep 0.5; send "e2e-pass\r"
sleep 0.5; send "e2e-pass\r"
must "backed up vault"
must "saved + pushed"
must "generate in"
sleep 0.3; send "\x1b"
EOF
[ -f "$CASA_SOURCE/.casa/keys/vault.key.age" ] || fail "backup file missing"
[ -f "$CASA_SOURCE/run_once_before_00-casa-keys.sh.tmpl" ] || fail "restore script missing"
CASA_KEY_RESTORED="$SB/restored.txt"
expect -c "set timeout 20
spawn age --decrypt -o $CASA_KEY_RESTORED $CASA_SOURCE/.casa/keys/vault.key.age
expect \"passphrase\"
sleep 0.3; send \"e2e-pass\r\"
expect eof" >/dev/null 2>&1
cmp -s "$CASA_KEY_RESTORED" "$HOME/.config/casa/keys/vault.txt" || fail "backup does not round-trip"

exp "keys — delete vault, orphan re-encrypted with main" <<EOF
spawn casa secrets keys
must "generate in"
sleep 0.3; send "vault"; sleep 0.3; send "\r"
must "make default"
sleep 0.3; send "delete"; sleep 0.3; send "\r"
must "only readable by vault"
sleep 0.8; send "\r"
must "re-encrypted 1 file"
must "delete the private key file"
sleep 0.3; send "y"
must "deleted key vault"
must "saved + pushed"
must "generate in"
sleep 0.3; send "\x1b"
EOF
[ ! -f "$HOME/.config/casa/keys/vault.txt" ] || fail "vault key file still present"
[ ! -f "$CASA_SOURCE/.casa/keys/vault.key.age" ] || fail "dead key backup still in repo"
chezmoi --source "$CASA_SOURCE" cat "$HOME/.vault.token" | grep -q vault-secret \
  || fail "orphan not readable after re-encrypt with main"

# ---- 7. storage: toggle template on and off ----------------------------------
exp "storage — plain → template" <<EOF
spawn casa configs storage .plainrc
must "executable"; hit " "; hit "\r"
must "now stored: template"
must "saved + pushed"
EOF
[ -f "$CASA_SOURCE/dot_plainrc.tmpl" ] || fail "chattr +template missing"

exp "storage — template → plain" <<EOF
spawn casa configs storage .plainrc
must "executable"; hit " "; hit "\r"
must "now stored plain"
must "saved + pushed"
EOF
[ -f "$CASA_SOURCE/dot_plainrc" ] || fail "chattr -template missing"

# ---- 8. edit ------------------------------------------------------------------
exp "configs edit — exact match, apply, autosave" <<EOF
spawn casa configs edit .plainrc
must "edited ~/.plainrc"
must "saved + pushed"
EOF
grep -q "e2e-appended-line" "$HOME/.plainrc" || fail "edit did not apply"

# ---- 9. answers: change one, keep the rest -----------------------------------
exp "machine answers — change Email only" <<EOF
spawn casa machine answers Email
must "Email"; hit "2\r"
must "applying"
EOF
grep -q "email = e2e@example.com2" "$HOME/.gitconfig" || fail "answer change did not re-render"
grep -q 'hosttype  = "laptop"' "$HOME/.config/chezmoi/chezmoi.toml" || fail "other answers were lost"

# ---- 10. question: author a new setup question --------------------------------
exp "machine question — add + answer immediately" <<EOF
spawn casa machine question
must "data key";  hit "favtool\r"
must "question to ask"; hit "Favorite tool\r"
must "yes / no";  hit "\r"
must "Favorite tool"; hit "helix\r"
must "use {{ .favtool }}"
must "saved + pushed"
EOF
grep -q 'promptStringOnce . "favtool" "Favorite tool"' "$CASA_SOURCE/.casa.toml.tmpl" \
  || fail "question not written to questionnaire"
chezmoi --source "$CASA_SOURCE" data --format json | grep -q '"favtool": "helix"' \
  || fail "question answer not in data"

# ---- 11. save / undo / sync ----------------------------------------------------
step "save"
printf 'drift\n' >> "$CASA_SOURCE/README.md"
casa save | grep "saved + pushed" >/dev/null || fail "save"
casa save | grep "nothing to save" >/dev/null || fail "save clean"

exp "undo — confirm and revert" <<EOF
spawn casa machine undo
must "undo last change?"; hit "\x1b\[D"; hit "\r"
must "applying the revert"
EOF
grep -q "drift" "$CASA_SOURCE/README.md" && fail "undo did not revert"

step "sync"
casa sync | grep "up to date" >/dev/null || fail "sync"

step "sync — pushes unsaved changes first"
printf '# sync-push-line\n' >> "$CASA_SOURCE/README.md"
exp "sync with dirty repo offers push" <<EOF
spawn casa sync
must "unsaved local changes"
must "push these as part of the sync?"; hit "\r"
must "up to date"
EOF
cd "$CASA_SOURCE" && git status --porcelain | grep -q . && fail "sync did not push"; cd - >/dev/null

# ---- 12. status / info / doctor / tools ----------------------------------------
step "status + info + doctor + tools"
casa status | grep "unsaved changes:" >/dev/null || fail "status"
casa machine info | grep "managed:" >/dev/null || fail "info"
casa machine doctor >/dev/null 2>&1 || true # informational; exit code varies by env
casa tools list >/dev/null || fail "tools list"
casa tools update | grep "nothing outdated" >/dev/null || fail "tools update (brew masked)"

# ---- 13. menu opens and quits ---------------------------------------------------
# drift: change a managed target outside casa, review the diff, keep local
printf 'locally changed\n' >> "$HOME/.plainrc"
exp "files drift — diff in pager, keep local records it" <<EOF
spawn casa files drift
must "plainrc"
sleep 0.3; send "\r"
must "locally changed"
must "keep my local version"
sleep 0.4; send "\r"
must "recorded your local"
must "saved + pushed"
must "nothing drifted"
EOF
grep -q "locally changed" "$CASA_SOURCE/dot_plainrc" || fail "drift keep did not record local version"

exp "menu — noun clusters, unified verbs, list pager, esc backs out" <<EOF
spawn casa
must "pick + edit a file"; must "install a tool"
sleep 0.3; send "managed files"; sleep 0.4; send "\r"
must "~/.testrc"
sleep 0.3; send "\x1b"
must "install a tool"
sleep 0.3; send "\x1b"
EOF

# smart edit: picking an encrypted file from the unified edit routes through
# the secret flow (decrypt → edit → same-key re-seal)
exp "edit — encrypted file routes to secret flow" <<EOF
spawn casa configs edit .apitoken.key
must "updated secret"
EOF

# ---- 14. untrack -----------------------------------------------------------------
step "untrack"
casa configs untrack "$HOME/.plainrc" | grep "no longer managing" >/dev/null || fail "untrack"
casa configs list | grep -q "^~/.plainrc$" && fail "untrack left file managed"

printf '\n\033[32m✓ e2e: all actions passed\033[0m\n'
