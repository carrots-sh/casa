// Encryption keys: create, adopt, pick a default, encrypt-with, delete with
// orphan re-encryption, and optional doppler backup of private identities.
// Registry-free: keys are the files in ~/.config/casa/keys — no names, paths,
// or recipients ever land in a repo.
package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/carrots-sh/casa/internal/agekey"
	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/home"
	"github.com/carrots-sh/casa/internal/ui"
)

// writeAgeBlock puts the generic encryption block into the config template
// (idempotent — the block embeds no machine or key specifics) and re-renders
// the machine config so identities/recipient pick up the keys directory.
func writeAgeBlock() error {
	path, ok := chez.ConfigTemplate()
	if !ok {
		path = filepath.Join(chez.SourceDir(), ".casa.toml.tmpl")
	}
	block := strings.TrimRight(agekey.AgeBlock, "\n")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if werr := os.WriteFile(path, []byte(block+"\n"), 0o644); werr != nil {
			return werr
		}
		return chez.Init()
	}
	if err != nil {
		return err
	}
	if strings.Contains(string(data), block) {
		return chez.Init() // block already current — just re-render the config
	}
	// drop any old encryption/[age] block, then insert before the first section
	var out []string
	inAge := false
	for l := range strings.SplitSeq(string(data), "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "encryption =") {
			continue
		}
		if t == "[age]" {
			inAge = true
			continue
		}
		if inAge {
			if strings.HasPrefix(t, "[") {
				inAge = false
			} else {
				continue
			}
		}
		out = append(out, l)
	}
	blockLines := strings.Split(block, "\n")
	inserted := false
	for i, l := range out {
		if strings.HasPrefix(strings.TrimSpace(l), "[") {
			rest := append([]string{}, out[i:]...)
			out = append(out[:i:i], blockLines...)
			out = append(out, rest...)
			inserted = true
			break
		}
	}
	if !inserted {
		out = append(out, blockLines...)
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return err
	}
	return chez.Init()
}

var keyNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ensureKeys guarantees at least one key, migrating a legacy ~/key.txt into
// the keys directory or generating a first key ("main").
func ensureKeys() ([]agekey.Key, error) {
	keys, err := agekey.List()
	if err != nil || len(keys) > 0 {
		return keys, err
	}
	if _, err := exec.LookPath("age-keygen"); err != nil {
		return nil, fmt.Errorf("age is not installed — casa needs it for encryption (brew install age)")
	}
	if _, err := os.Stat(home.Path("key.txt")); err == nil {
		fmt.Println("moving your age key into casa's keys dir: ~/key.txt → " + home.Tilde(agekey.Path("main")))
		if _, err := agekey.Adopt(home.Path("key.txt"), "main"); err != nil {
			return nil, err
		}
	} else {
		fmt.Println("no encryption key yet — creating one.")
		k, err := agekey.Generate("main")
		if err != nil {
			return nil, err
		}
		fmt.Printf("✓ created %s (back it up! without it your secrets are unreadable)\n", home.Tilde(k.Identity))
	}
	if err := agekey.SetDefault("main"); err != nil {
		return nil, err
	}
	if err := writeAgeBlock(); err != nil {
		return nil, err
	}
	return agekey.List()
}

// pickEncryptKey picks which key seals a new secret: the default when it's
// the only one, a picker (default preselected) when there are several.
func pickEncryptKey(keys []agekey.Key) (agekey.Key, error) {
	if len(keys) == 1 {
		return keys[0], nil
	}
	def, _ := agekey.Default()
	labels := make([]string, len(keys))
	byLabel := map[string]agekey.Key{}
	defLabel := ""
	for i, k := range keys {
		l := keyRow(k, def.Name)
		labels[i] = l
		byLabel[l] = k
		if k.Name == def.Name {
			defLabel = l
		}
	}
	sel, err := ui.SelectDefault("encrypt with which key?", labels, defLabel)
	if err != nil || sel == "" {
		return agekey.Key{}, err
	}
	return byLabel[sel], nil
}

// reEncryptSource swaps a chezmoi-encrypted source file's content to be
// sealed for k instead (the plaintext is supplied by the caller).
func reEncryptSource(homePath string, plaintext []byte, k agekey.Key) error {
	srcs, err := chez.SourcePaths([]string{homePath})
	if err != nil || len(srcs) != 1 {
		return fmt.Errorf("couldn't find the encrypted source of %s", home.Tilde(homePath))
	}
	sealed, err := k.Encrypt(plaintext)
	if err != nil {
		return err
	}
	return os.WriteFile(srcs[0], sealed, 0o644)
}

func keyRow(k agekey.Key, defName string) string {
	rec, err := k.Recipient()
	if err != nil {
		rec = "(unreadable)"
	} else if len(rec) > 15 {
		rec = rec[:15] + "…"
	}
	mark := ""
	if k.Name == defName {
		mark = "  ★ default"
	}
	return fmt.Sprintf("%-8s %s%s", k.Name, rec, mark)
}

// Keys is the key-management screen.
func Keys() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	for {
		keys, err := ensureKeys()
		if err != nil {
			return err
		}
		def, _ := agekey.Default()
		const newKey = "new key · generate in ~/.config/casa/keys"
		labels := []string{}
		byLabel := map[string]agekey.Key{}
		for _, k := range keys {
			l := keyRow(k, def.Name)
			labels = append(labels, l)
			byLabel[l] = k
		}
		labels = append(labels, newKey)
		sel, err := ui.Select("encryption keys", labels)
		if err != nil || sel == "" {
			return err
		}
		if sel == newKey {
			if err := createKey(); err != nil {
				return err
			}
			continue
		}
		if err := keyActions(byLabel[sel]); err != nil {
			return err
		}
	}
}

func createKey() error {
	name, err := ui.Input("key name (e.g. work, vault)")
	if err != nil || name == "" {
		return err
	}
	if !keyNameRe.MatchString(name) {
		return fmt.Errorf("key names are lowercase letters, digits, dashes")
	}
	k, err := agekey.Generate(name)
	if err != nil {
		return err
	}
	if err := writeAgeBlock(); err != nil {
		return err
	}
	fmt.Printf("✓ created %s → %s (back it up!)\n", k.Name, home.Tilde(k.Identity))
	return nil
}

func keyActions(k agekey.Key) error {
	acts := []string{"make default", "backup to repo (passphrase)", "delete"}
	if _, err := exec.LookPath("doppler"); err == nil {
		acts = append(acts, "push to doppler", "pull from doppler")
	}
	rec, _ := k.Recipient()
	sel, err := ui.Select(k.Name+" · "+rec, acts)
	if err != nil || sel == "" {
		return err
	}
	switch sel {
	case "backup to repo (passphrase)":
		return backupKey(k)
	case "make default":
		if err := agekey.SetDefault(k.Name); err != nil {
			return err
		}
		if err := chez.Init(); err != nil { // re-derive the config's recipient
			return err
		}
		fmt.Printf("✓ new secrets on this machine encrypt with %s\n", k.Name)
		return nil
	case "delete":
		return deleteKey(k)
	case "push to doppler":
		return dopplerKey(k.Name, true)
	case "pull from doppler":
		return dopplerKey(k.Name, false)
	}
	return nil
}

// backupKey passphrase-encrypts the private identity into the repo
// (.casa/keys/<name>.key.age — safe to commit) and generates the restore
// script that decrypts backups on a new machine before anything needs them.
func backupKey(k agekey.Key) error {
	fmt.Println("choose a passphrase — a new machine needs it (plus the repo) to restore this key.")
	out, err := k.Backup(filepath.Join(chez.SourceDir(), agekey.BackupRel))
	if err != nil {
		return err
	}
	script := filepath.Join(chez.SourceDir(), agekey.RestoreScript)
	if _, err := os.Stat(script); os.IsNotExist(err) {
		if err := os.WriteFile(script, []byte(agekey.RestoreScriptBody), 0o644); err != nil {
			return err
		}
		fmt.Println("  + " + agekey.RestoreScript + " (restores backups on new machines)")
	}
	fmt.Printf("✓ backed up %s → %s\n", k.Name, home.Tilde(out))
	offerSave("casa: backup encryption key " + k.Name)
	return nil
}

// deleteKey removes a key (= its identity file). Files only that key can open
// are orphans: casa re-encrypts them with a surviving key first, or refuses.
func deleteKey(k agekey.Key) error {
	keys, err := agekey.List()
	if err != nil {
		return err
	}
	if len(keys) == 1 {
		return fmt.Errorf("%s is your only key — create another first", k.Name)
	}
	var survivors []agekey.Key
	for _, o := range keys {
		if o.Name != k.Name {
			survivors = append(survivors, o)
		}
	}
	orphans, err := orphanedBy(k, survivors)
	if err != nil {
		return err
	}
	if len(orphans) > 0 {
		fmt.Printf("%d file(s) are only readable by %s:\n", len(orphans), k.Name)
		for _, d := range targetLabels(orphans) {
			fmt.Println("  " + d)
		}
		repl, err := pickReplacement(survivors)
		if err != nil || repl.Name == "" {
			return err
		}
		for _, rel := range orphans {
			abs := filepath.Join(chez.SourceDir(), rel)
			plain, err := k.Decrypt(abs)
			if err != nil {
				return err
			}
			sealed, err := repl.Encrypt(plain)
			if err != nil {
				return err
			}
			if err := os.WriteFile(abs, sealed, 0o644); err != nil {
				return err
			}
		}
		fmt.Printf("✓ re-encrypted %d file(s) with %s\n", len(orphans), repl.Name)
	}
	ok, err := ui.Confirm("delete the private key file " + home.Tilde(k.Identity) + "? (unrecoverable)")
	if err != nil || !ok {
		return err
	}
	if err := os.Remove(k.Identity); err != nil {
		return err
	}
	backup := filepath.Join(chez.SourceDir(), agekey.BackupRel, k.Name+".key.age")
	if err := os.Remove(backup); err == nil {
		fmt.Println("  - removed its repo backup (a new machine must never try to restore a dead key)")
	}
	if def, _ := agekey.Default(); def.Name != k.Name {
		_ = agekey.SetDefault(def.Name) // refresh marker if it pointed at the deleted key
	}
	if err := chez.Init(); err != nil {
		return err
	}
	fmt.Printf("✓ deleted key %s\n", k.Name)
	offerSave("casa: re-encrypt after deleting key " + k.Name)
	return nil
}

// orphanedBy lists encrypted sources (repo-relative) that only k can open.
func orphanedBy(k agekey.Key, survivors []agekey.Key) ([]string, error) {
	enc, err := chez.EncryptedSources()
	if err != nil {
		return nil, err
	}
	var orphans []string
	for _, rel := range enc {
		abs := filepath.Join(chez.SourceDir(), rel)
		if !k.CanDecrypt(abs) {
			continue
		}
		saved := false
		for _, o := range survivors {
			if o.CanDecrypt(abs) {
				saved = true
				break
			}
		}
		if !saved {
			orphans = append(orphans, rel)
		}
	}
	return orphans, nil
}

func pickReplacement(survivors []agekey.Key) (agekey.Key, error) {
	labels := make([]string, len(survivors))
	byLabel := map[string]agekey.Key{}
	for i, k := range survivors {
		labels[i] = k.Name
		byLabel[labels[i]] = k
	}
	sel, err := ui.Select("re-encrypt those files with which key?", labels)
	return byLabel[sel], err
}

// dopplerKey backs a private identity up to (or restores from) doppler.
// Requires a doppler project already set up (doppler setup).
func dopplerKey(name string, push bool) error {
	secret := "CASA_AGE_KEY_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	p := agekey.Path(name)
	if push {
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		c := exec.Command("doppler", "secrets", "set", secret+"="+string(b))
		c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := c.Run(); err != nil {
			return fmt.Errorf("doppler (run `doppler setup` first?): %w", err)
		}
		fmt.Printf("✓ pushed %s to doppler as %s\n", name, secret)
		return nil
	}
	out, err := exec.Command("doppler", "secrets", "get", secret, "--plain").Output()
	if err != nil {
		return fmt.Errorf("doppler get %s (run `doppler setup` first?): %w", secret, err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(p, out, 0o600); err != nil {
		return err
	}
	fmt.Printf("✓ restored %s to %s\n", name, home.Tilde(p))
	return chez.Init()
}
