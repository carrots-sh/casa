// Encryption keys: create, adopt, pick a default, encrypt-with, delete with
// orphan re-encryption, and optional doppler backup of private identities.
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

func keyReg() (agekey.Registry, error) {
	return agekey.Load(filepath.Join(chez.SourceDir(), agekey.RegistryRel))
}

// writeAgeBlock regenerates the encryption block of the config template from
// the registry and re-renders the machine config.
func writeAgeBlock(r agekey.Registry) error {
	path, ok := chez.ConfigTemplate()
	if !ok {
		path = filepath.Join(chez.SourceDir(), ".casa.toml.tmpl")
	}
	block := agekey.AgeBlock(r)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return os.WriteFile(path, []byte(block), 0o644)
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	// drop the old block: the encryption line, the [age] section (through the
	// next [section]), and any casa-generated template lines in between
	var out []string
	inAge := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "encryption =") {
			continue
		}
		if t == "[age]" {
			inAge = true
			continue
		}
		if inAge {
			if strings.HasPrefix(t, "[") { // next real section
				inAge = false
			} else {
				continue
			}
		}
		out = append(out, l)
	}
	// insert the fresh block before the first section ([data], [edit], …)
	for i, l := range out {
		if strings.HasPrefix(strings.TrimSpace(l), "[") {
			rest := append([]string{}, out[i:]...)
			out = append(out[:i:i], strings.Split(strings.TrimRight(block, "\n"), "\n")...)
			out = append(out, rest...)
			if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
				return err
			}
			return chez.Init()
		}
	}
	out = append(out, strings.Split(strings.TrimRight(block, "\n"), "\n")...)
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return err
	}
	return chez.Init()
}

var keyNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ensureKeys guarantees at least one usable key, adopting a legacy ~/key.txt
// or generating a first key ("main") when the registry is empty.
func ensureKeys() (agekey.Registry, error) {
	r, err := keyReg()
	if err != nil || len(r.Keys) > 0 {
		return r, err
	}
	if _, err := exec.LookPath("age-keygen"); err != nil {
		return r, fmt.Errorf("age is not installed — casa needs it for encryption (brew install age)")
	}
	// legacy single-key setup: ~/key.txt referenced by the config template
	if _, err := os.Stat(home.Path("key.txt")); err == nil {
		rec, err := agekey.RecipientOf("~/key.txt")
		if err != nil {
			return r, err
		}
		fmt.Println("found your existing age key (~/key.txt) — registering it as \"main\".")
		r.Keys = append(r.Keys, agekey.Key{Name: "main", Recipient: rec, Identity: "~/key.txt"})
		r.Default = "main"
	} else {
		fmt.Println("no encryption key yet — creating one.")
		k, err := agekey.Generate("main")
		if err != nil {
			return r, err
		}
		fmt.Printf("✓ created %s (back it up! without it your secrets are unreadable)\n", k.Identity)
		r.Keys = append(r.Keys, k)
		r.Default = "main"
	}
	if err := r.Save(); err != nil {
		return r, err
	}
	return r, writeAgeBlock(r)
}

// pickEncryptKey picks which key seals a new secret: the default when it's
// the only one, a picker (default preselected) when there are several.
func pickEncryptKey(r agekey.Registry) (agekey.Key, error) {
	if len(r.Keys) == 1 {
		return r.Keys[0], nil
	}
	labels := make([]string, len(r.Keys))
	byLabel := map[string]agekey.Key{}
	def := ""
	for i, k := range r.Keys {
		l := keyRow(r, k)
		labels[i] = l
		byLabel[l] = k
		if k.Name == r.Default {
			def = l
		}
	}
	sel, err := ui.SelectDefault("encrypt with which key?", labels, def)
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
	sealed, err := agekey.Encrypt(plaintext, k.Recipient)
	if err != nil {
		return err
	}
	return os.WriteFile(srcs[0], sealed, 0o644)
}

func keyRow(r agekey.Registry, k agekey.Key) string {
	marks := ""
	if k.Name == r.Default {
		marks += "  ★ default"
	}
	if !k.Present() {
		marks += "  (not on this machine)"
	}
	return fmt.Sprintf("%-8s %s…%s", k.Name, k.Recipient[:min(14, len(k.Recipient))], marks)
}

// Keys is the key-management screen.
func Keys() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	for {
		r, err := ensureKeys()
		if err != nil {
			return err
		}
		const newKey = "new key · generate + register"
		labels := []string{}
		byLabel := map[string]agekey.Key{}
		for _, k := range r.Keys {
			l := keyRow(r, k)
			labels = append(labels, l)
			byLabel[l] = k
		}
		labels = append(labels, newKey)
		sel, err := ui.Select("encryption keys", labels)
		if err != nil || sel == "" {
			return err
		}
		if sel == newKey {
			if err := createKey(&r); err != nil {
				return err
			}
			continue
		}
		if err := keyActions(&r, byLabel[sel]); err != nil {
			return err
		}
	}
}

func createKey(r *agekey.Registry) error {
	name, err := ui.Input("key name (e.g. work, vault)")
	if err != nil || name == "" {
		return err
	}
	if !keyNameRe.MatchString(name) {
		return fmt.Errorf("key names are lowercase letters, digits, dashes")
	}
	if _, taken := r.Get(name); taken {
		return fmt.Errorf("key %q already exists", name)
	}
	k, err := agekey.Generate(name)
	if err != nil {
		return err
	}
	r.Keys = append(r.Keys, k)
	if err := r.Save(); err != nil {
		return err
	}
	if err := writeAgeBlock(*r); err != nil {
		return err
	}
	fmt.Printf("✓ created %s → %s (back it up!)\n", k.Name, k.Identity)
	offerSave("casa: add encryption key " + k.Name)
	return nil
}

func keyActions(r *agekey.Registry, k agekey.Key) error {
	acts := []string{"make default", "delete"}
	if _, err := exec.LookPath("doppler"); err == nil {
		acts = append(acts, "push to doppler", "pull from doppler")
	}
	sel, err := ui.Select(k.Name+" · "+k.Recipient, acts)
	if err != nil || sel == "" {
		return err
	}
	switch sel {
	case "make default":
		r.Default = k.Name
		if err := r.Save(); err != nil {
			return err
		}
		if err := writeAgeBlock(*r); err != nil {
			return err
		}
		fmt.Printf("✓ new secrets encrypt with %s\n", k.Name)
		offerSave("casa: default encryption key → " + k.Name)
		return nil
	case "delete":
		return deleteKey(r, k)
	case "push to doppler":
		return dopplerKey(k, true)
	case "pull from doppler":
		return dopplerKey(k, false)
	}
	return nil
}

// deleteKey removes a key. Files only that key can open are orphans: casa
// re-encrypts them with a surviving key first (or refuses).
func deleteKey(r *agekey.Registry, k agekey.Key) error {
	if len(r.Keys) == 1 {
		return fmt.Errorf("%s is your only key — create another first", k.Name)
	}
	var survivors []agekey.Key
	for _, o := range r.Keys {
		if o.Name != k.Name && o.Present() {
			survivors = append(survivors, o)
		}
	}
	if !k.Present() {
		ok, err := ui.Confirm("this key isn't on this machine, so casa can't check for files only it can open — remove it from the registry anyway?")
		if err != nil || !ok {
			return err
		}
	} else {
		orphans, err := orphanedBy(*r, k)
		if err != nil {
			return err
		}
		if len(orphans) > 0 {
			if len(survivors) == 0 {
				return fmt.Errorf("%d file(s) are only readable by %s and no other key is on this machine — aborting", len(orphans), k.Name)
			}
			disp := targetLabels(orphans)
			fmt.Printf("%d file(s) are only readable by %s:\n", len(orphans), k.Name)
			for _, d := range disp {
				fmt.Println("  " + d)
			}
			repl, err := pickReplacement(survivors)
			if err != nil || repl.Name == "" {
				return err
			}
			for _, rel := range orphans {
				abs := filepath.Join(chez.SourceDir(), rel)
				plain, err := agekey.Decrypt(k, abs)
				if err != nil {
					return err
				}
				sealed, err := agekey.Encrypt(plain, repl.Recipient)
				if err != nil {
					return err
				}
				if err := os.WriteFile(abs, sealed, 0o644); err != nil {
					return err
				}
			}
			fmt.Printf("✓ re-encrypted %d file(s) with %s\n", len(orphans), repl.Name)
		}
		ok, err := ui.Confirm("also delete the private key file " + k.Identity + "? (unrecoverable)")
		if err != nil {
			return err
		}
		if ok {
			if err := os.Remove(home.Expand(k.Identity)); err != nil {
				return err
			}
		}
	}
	kept := r.Keys[:0]
	for _, o := range r.Keys {
		if o.Name != k.Name {
			kept = append(kept, o)
		}
	}
	r.Keys = kept
	if r.Default == k.Name {
		r.Default = r.Keys[0].Name
		fmt.Printf("default key is now %s\n", r.Default)
	}
	if err := r.Save(); err != nil {
		return err
	}
	if err := writeAgeBlock(*r); err != nil {
		return err
	}
	fmt.Printf("✓ deleted key %s\n", k.Name)
	offerSave("casa: delete encryption key " + k.Name)
	return nil
}

// orphanedBy lists encrypted sources (repo-relative) that only k can open.
func orphanedBy(r agekey.Registry, k agekey.Key) ([]string, error) {
	enc, err := chez.EncryptedSources()
	if err != nil {
		return nil, err
	}
	var orphans []string
	for _, rel := range enc {
		abs := filepath.Join(chez.SourceDir(), rel)
		if !agekey.CanDecrypt(k, abs) {
			continue
		}
		saved := false
		for _, o := range r.Keys {
			if o.Name != k.Name && agekey.CanDecrypt(o, abs) {
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
func dopplerKey(k agekey.Key, push bool) error {
	secret := "CASA_AGE_KEY_" + strings.ToUpper(strings.ReplaceAll(k.Name, "-", "_"))
	if push {
		if !k.Present() {
			return fmt.Errorf("key %s isn't on this machine", k.Name)
		}
		b, err := os.ReadFile(home.Expand(k.Identity))
		if err != nil {
			return err
		}
		c := exec.Command("doppler", "secrets", "set", secret+"="+string(b))
		c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := c.Run(); err != nil {
			return fmt.Errorf("doppler (run `doppler setup` first?): %w", err)
		}
		fmt.Printf("✓ pushed %s to doppler as %s\n", k.Name, secret)
		return nil
	}
	out, err := exec.Command("doppler", "secrets", "get", secret, "--plain").Output()
	if err != nil {
		return fmt.Errorf("doppler get %s (run `doppler setup` first?): %w", secret, err)
	}
	p := home.Expand(k.Identity)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(p, out, 0o600); err != nil {
		return err
	}
	fmt.Printf("✓ restored %s to %s\n", k.Name, k.Identity)
	return nil
}
