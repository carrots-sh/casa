// Package agekey manages casa's age keys, registry-free: a key IS a private
// identity file in ~/.config/casa/keys/<name>.txt. Names are filenames,
// recipients are derived from the files (age-keygen -y), and the default is a
// local .default marker — so neither repo ever stores key names, paths, or
// recipients; every machine just reads the keys directory by convention.
package agekey

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carrots-sh/casa/internal/home"
)

// Dir is where keys live on every machine.
func Dir() string { return home.Path(".config/casa/keys") }

// Key is one age key: name = filename without .txt.
type Key struct {
	Name     string
	Identity string // absolute path of the private identity file
}

// Path returns the identity path a key with this name would have.
func Path(name string) string { return filepath.Join(Dir(), name+".txt") }

// List returns the keys present on this machine, sorted by name.
func List() ([]Key, error) {
	ents, err := os.ReadDir(Dir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var keys []Key
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".txt")
		keys = append(keys, Key{Name: name, Identity: Path(name)})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Name < keys[j].Name })
	return keys, nil
}

// Get returns the named key if its identity file exists.
func Get(name string) (Key, bool) {
	if _, err := os.Stat(Path(name)); err != nil {
		return Key{}, false
	}
	return Key{Name: name, Identity: Path(name)}, true
}

// Default returns the default key: the .default marker's choice when valid,
// else the first key — the same rule the generated [age] block applies, so
// casa and chezmoi always agree.
func Default() (Key, bool) {
	if b, err := os.ReadFile(filepath.Join(Dir(), ".default")); err == nil {
		if k, ok := Get(strings.TrimSpace(string(b))); ok {
			return k, true
		}
	}
	keys, _ := List()
	if len(keys) > 0 {
		return keys[0], true
	}
	return Key{}, false
}

// SetDefault records the default key in the local marker.
func SetDefault(name string) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(Dir(), ".default"), []byte(name+"\n"), 0o644)
}

// Generate creates a new identity at ~/.config/casa/keys/<name>.txt.
func Generate(name string) (Key, error) {
	p := Path(name)
	if _, err := os.Stat(p); err == nil {
		return Key{}, fmt.Errorf("key file already exists: %s", home.Tilde(p))
	}
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return Key{}, err
	}
	out, err := exec.Command("age-keygen", "-o", p).CombinedOutput()
	if err != nil {
		return Key{}, fmt.Errorf("age-keygen: %s", strings.TrimSpace(string(out)))
	}
	_ = os.Chmod(p, 0o600)
	return Key{Name: name, Identity: p}, nil
}

// Adopt moves an identity file from elsewhere (e.g. a legacy ~/key.txt) into
// the keys directory under name.
func Adopt(from, name string) (Key, error) {
	p := Path(name)
	if _, err := os.Stat(p); err == nil {
		return Key{}, fmt.Errorf("key %q already exists", name)
	}
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return Key{}, err
	}
	if err := os.Rename(from, p); err != nil {
		// cross-device fallback
		b, rerr := os.ReadFile(from)
		if rerr != nil {
			return Key{}, err
		}
		if err := os.WriteFile(p, b, 0o600); err != nil {
			return Key{}, err
		}
		_ = os.Remove(from)
	}
	_ = os.Chmod(p, 0o600)
	return Key{Name: name, Identity: p}, nil
}

// Recipient derives the key's public recipient from its identity file.
func (k Key) Recipient() (string, error) {
	out, err := exec.Command("age-keygen", "-y", k.Identity).Output()
	if err != nil {
		return "", fmt.Errorf("age-keygen -y %s: %w", home.Tilde(k.Identity), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Encrypt seals plaintext for the key (armored, like chezmoi's age files).
func (k Key) Encrypt(plaintext []byte) ([]byte, error) {
	rec, err := k.Recipient()
	if err != nil {
		return nil, err
	}
	c := exec.Command("age", "--encrypt", "--armor", "--recipient", rec)
	c.Stdin = strings.NewReader(string(plaintext))
	out, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("age encrypt: %w", err)
	}
	return out, nil
}

// CanDecrypt reports whether this key opens the encrypted file.
func (k Key) CanDecrypt(encryptedPath string) bool {
	c := exec.Command("age", "--decrypt", "--identity", k.Identity, encryptedPath)
	c.Stdout, c.Stderr = nil, nil
	return c.Run() == nil
}

// Decrypt opens the encrypted file with the key's identity.
func (k Key) Decrypt(encryptedPath string) ([]byte, error) {
	out, err := exec.Command("age", "--decrypt", "--identity", k.Identity, encryptedPath).Output()
	if err != nil {
		return nil, fmt.Errorf("age decrypt with %s: %w", k.Name, err)
	}
	return out, nil
}

// AgeBlock is the config template's encryption block. It is fully generic —
// no key names, paths, or recipients ever enter the repo: identities glob the
// keys directory at init time and the default recipient is derived from the
// default key's file on the spot.
const AgeBlock = `encryption = "age"
[age]
{{- $keydir := joinPath .chezmoi.homeDir ".config/casa/keys" }}
{{- $keys := glob (joinPath $keydir "*.txt") }}
    identities = [{{ range $i, $p := $keys }}{{ if $i }}, {{ end }}{{ $p | quote }}{{ end }}]
{{- $def := "" }}
{{- $marker := joinPath $keydir ".default" }}
{{- if stat $marker }}{{ $def = joinPath $keydir (printf "%s.txt" (trim (output "cat" $marker))) }}{{ end }}
{{- if and (or (not $def) (not (stat $def))) $keys }}{{ $def = index $keys 0 }}{{ end }}
{{- if and $def (stat $def) (lookPath "age-keygen") }}
    recipient  = {{ output "age-keygen" "-y" $def | trim | quote }}
{{- end }}
`
