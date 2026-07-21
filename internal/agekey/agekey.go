// Package agekey manages casa's age keys: a committed registry of PUBLIC
// recipients (.casadata/keys.toml) and per-machine PRIVATE identity files.
// Encryption needs only a recipient, so any machine can encrypt to any key;
// decryption needs the identity file, which never enters the repo.
package agekey

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/carrots-sh/casa/internal/home"
)

// RegistryRel is the registry's path relative to the source dir. It lives in
// .casadata so the config template can render [age] from it as chezmoi data.
const RegistryRel = ".casadata/keys.toml"

// Key is one age key: a public recipient plus where its identity lives.
type Key struct {
	Name      string
	Recipient string `toml:"recipient"`
	Identity  string `toml:"identity"` // path with ~; private file, never committed
}

// Present reports whether the private identity exists on this machine.
func (k Key) Present() bool {
	_, err := os.Stat(home.Expand(k.Identity))
	return err == nil
}

// Registry is the committed key set.
type Registry struct {
	Path    string
	Default string
	Keys    []Key // sorted by name
}

type regDoc struct {
	Keys struct {
		Default string         `toml:"default"`
		Named   map[string]Key `toml:"named"`
	} `toml:"keys"`
}

// Load reads the registry (empty registry if the file doesn't exist).
func Load(path string) (Registry, error) {
	r := Registry{Path: path}
	var d regDoc
	if _, err := toml.DecodeFile(path, &d); err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return r, err
	}
	r.Default = d.Keys.Default
	for name, k := range d.Keys.Named {
		k.Name = name
		r.Keys = append(r.Keys, k)
	}
	sort.Slice(r.Keys, func(i, j int) bool { return r.Keys[i].Name < r.Keys[j].Name })
	return r, nil
}

// Save writes the registry (recipients are public — safe to commit).
func (r Registry) Save() error {
	var b strings.Builder
	b.WriteString("# casa's age keys — public recipients only; private identities stay on\n")
	b.WriteString("# each machine at the listed paths. Managed by `casa secrets keys`.\n\n")
	b.WriteString("[keys]\n")
	fmt.Fprintf(&b, "default = %q\n", r.Default)
	for _, k := range r.Keys {
		fmt.Fprintf(&b, "\n[keys.named.%s]\n", k.Name)
		fmt.Fprintf(&b, "recipient = %q\n", k.Recipient)
		fmt.Fprintf(&b, "identity = %q\n", k.Identity)
	}
	if err := os.MkdirAll(filepath.Dir(r.Path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.Path, []byte(b.String()), 0o644)
}

// Get returns the named key.
func (r Registry) Get(name string) (Key, bool) {
	for _, k := range r.Keys {
		if k.Name == name {
			return k, true
		}
	}
	return Key{}, false
}

// DefaultKey returns the default key when set and known.
func (r Registry) DefaultKey() (Key, bool) { return r.Get(r.Default) }

// IdentityDir is where casa creates new private keys.
func IdentityDir() string { return home.Path(".config/casa/keys") }

// Generate creates a new identity at ~/.config/casa/keys/<name>.txt and
// returns the key. Fails if the name is taken or the file exists.
func Generate(name string) (Key, error) {
	p := filepath.Join(IdentityDir(), name+".txt")
	if _, err := os.Stat(p); err == nil {
		return Key{}, fmt.Errorf("key file already exists: %s", home.Tilde(p))
	}
	if err := os.MkdirAll(IdentityDir(), 0o700); err != nil {
		return Key{}, err
	}
	out, err := exec.Command("age-keygen", "-o", p).CombinedOutput()
	if err != nil {
		return Key{}, fmt.Errorf("age-keygen: %s", strings.TrimSpace(string(out)))
	}
	_ = os.Chmod(p, 0o600)
	rec, err := RecipientOf(p)
	if err != nil {
		return Key{}, err
	}
	return Key{Name: name, Recipient: rec, Identity: home.Tilde(p)}, nil
}

// RecipientOf derives the public recipient from an identity file.
func RecipientOf(identityPath string) (string, error) {
	out, err := exec.Command("age-keygen", "-y", home.Expand(identityPath)).Output()
	if err != nil {
		return "", fmt.Errorf("age-keygen -y: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Encrypt seals plaintext for recipient (armored, like chezmoi's age files).
func Encrypt(plaintext []byte, recipient string) ([]byte, error) {
	c := exec.Command("age", "--encrypt", "--armor", "--recipient", recipient)
	c.Stdin = strings.NewReader(string(plaintext))
	out, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("age encrypt: %w", err)
	}
	return out, nil
}

// CanDecrypt reports whether this key's identity opens the encrypted file.
func CanDecrypt(k Key, encryptedPath string) bool {
	if !k.Present() {
		return false
	}
	c := exec.Command("age", "--decrypt", "--identity", home.Expand(k.Identity), encryptedPath)
	c.Stdout, c.Stderr = nil, nil
	return c.Run() == nil
}

// Decrypt opens the encrypted file with the key's identity.
func Decrypt(k Key, encryptedPath string) ([]byte, error) {
	out, err := exec.Command("age", "--decrypt", "--identity", home.Expand(k.Identity), encryptedPath).Output()
	if err != nil {
		return nil, fmt.Errorf("age decrypt with %s: %w", k.Name, err)
	}
	return out, nil
}

// AgeBlock renders the config template's encryption block from the registry:
// identities are stat-filtered at init time, so a machine that only holds
// some of the keys still decrypts everything those keys cover; encryption
// (chezmoi add --encrypt) targets the default key's recipient.
func AgeBlock(r Registry) string {
	var quoted []string
	for _, k := range r.Keys {
		quoted = append(quoted, fmt.Sprintf("%q", k.Identity))
	}
	def, _ := r.DefaultKey()
	var b strings.Builder
	b.WriteString("encryption = \"age\"\n")
	b.WriteString("[age]\n")
	b.WriteString("{{- $ids := list }}\n")
	fmt.Fprintf(&b, "{{- range list %s }}\n", strings.Join(quoted, " "))
	b.WriteString("{{- $p := replace \"~\" $.chezmoi.homeDir . }}\n")
	b.WriteString("{{- if stat $p }}{{ $ids = append $ids $p }}{{ end }}\n")
	b.WriteString("{{- end }}\n")
	b.WriteString("    identities = [{{ range $i, $p := $ids }}{{ if $i }}, {{ end }}{{ $p | quote }}{{ end }}]\n")
	fmt.Fprintf(&b, "    recipient  = %q\n", def.Recipient)
	return b.String()
}
