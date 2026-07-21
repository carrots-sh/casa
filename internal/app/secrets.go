package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/agekey"
	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/home"
	"github.com/carrots-sh/casa/internal/ui"
)

// AddSecret starts managing a file, encrypted — with the default key, or a
// picked one when several are registered.
func AddSecret(path string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if path == "" {
		var err error
		if path, err = ui.PathInput("path of the file to encrypt + manage"); err != nil || path == "" {
			return err
		}
	}
	abs := home.Expand(path)
	keys, err := ensureKeys()
	if err != nil {
		return err
	}
	k, err := pickEncryptKey(keys)
	if err != nil || k.Name == "" {
		return err
	}
	plain, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	if err := chez.AddEncrypt(abs); err != nil { // seals with the default key
		return err
	}
	if def, _ := agekey.Default(); k.Name != def.Name {
		if err := reEncryptSource(abs, plain, k); err != nil {
			return err
		}
	}
	fmt.Printf("✓ encrypted with %s and now managing %s\n", k.Name, home.Tilde(abs))
	offerSave("casa: add secret " + filepath.Base(path))
	return nil
}

// EditSecret picks an encrypted source file, decrypts it to a temp file, opens
// the editor, then re-encrypts back into the repo.
func EditSecret(name string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	enc, err := chez.EncryptedSources()
	if err != nil {
		return err
	}
	if len(enc) == 0 {
		fmt.Println("no secrets yet — add one with: casa secrets add <path>")
		return nil
	}
	disp, bySource := displayNames(enc)
	var sel string
	if name != "" {
		var filtered []string
		for _, d := range disp {
			if strings.Contains(strings.ToLower(d), strings.ToLower(name)) {
				filtered = append(filtered, d)
			}
		}
		switch len(filtered) {
		case 1:
			sel = filtered[0]
		case 0:
			return fmt.Errorf("no secret matches %q", name)
		default:
			if sel, err = ui.Select("edit which secret?", filtered); err != nil || sel == "" {
				return err
			}
		}
	} else if sel, err = ui.Select("edit which secret?", disp); err != nil || sel == "" {
		return err
	}
	return editSecretSource(bySource[sel], sel)
}

// editSecretSource decrypts a secret source, opens the editor, validates
// templates, and re-seals with the same key. Also the routing target when
// `edit` picks an encrypted file.
func editSecretSource(source, display string) error {
	plain, err := chez.Decrypt(source)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "casa-secret-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plain); err != nil {
		return err
	}
	tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	isTemplate := strings.Contains(source, ".tmpl")
	var edited []byte
	for {
		c := exec.Command(editor, tmp.Name())
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			return err
		}
		if edited, err = os.ReadFile(tmp.Name()); err != nil {
			return err
		}
		if !isTemplate {
			break
		}
		// catch template errors now, not at the next apply
		terr := chez.ExecuteTemplate(string(edited))
		if terr == nil {
			break
		}
		fmt.Printf("template error: %v\n", terr)
		again, err := ui.ConfirmDefault("edit again to fix it?", true)
		if err != nil {
			return err
		}
		if !again {
			break // save as-is; the user knows
		}
	}
	if err := sealSecret(string(edited), source); err != nil {
		return err
	}
	_ = chez.ApplyNoScripts() // re-render any targets assembled from this secret
	fmt.Printf("✓ updated secret %s\n", display)
	offerSave("casa: edit secret")
	return nil
}

// sealSecret re-encrypts an edited secret with the SAME key that sealed it
// (probed against the keys directory), so editing never silently rotates a
// file to the default key. Falls back to chezmoi's encryption otherwise.
func sealSecret(plaintext, sourceRel string) error {
	abs := filepath.Join(chez.SourceDir(), sourceRel)
	if keys, err := agekey.List(); err == nil {
		for _, k := range keys {
			if !k.CanDecrypt(abs) {
				continue
			}
			sealed, err := k.Encrypt([]byte(plaintext))
			if err != nil {
				return err
			}
			return os.WriteFile(abs, sealed, 0o644)
		}
	}
	return chez.EncryptInto(plaintext, sourceRel)
}

// targetLabels converts source paths to readable ~/ target paths, falling back
// to the sources themselves if the conversion fails or is incomplete.
func targetLabels(sources []string) []string {
	disp, err := chez.TargetPaths(sources)
	if err != nil || len(disp) != len(sources) {
		return sources
	}
	for i, d := range disp {
		disp[i] = home.Tilde(d)
	}
	return disp
}

// displayNames maps encrypted source paths to readable target-style names,
// returning the labels (in order) and a label→source lookup. Falls back to the
// source paths if the conversion fails.
func displayNames(sources []string) ([]string, map[string]string) {
	disp := targetLabels(sources)
	bySource := make(map[string]string, len(sources))
	for i, d := range disp {
		bySource[d] = sources[i]
	}
	return disp, bySource
}

// secretLines renders the encrypted files by their readable target paths.
func secretLines() ([]string, error) {
	enc, err := chez.EncryptedSources()
	if err != nil {
		return nil, err
	}
	disp, _ := displayNames(enc)
	return disp, nil
}

// ListSecrets prints the encrypted files (plain output — pipeable).
func ListSecrets() error {
	lines, err := secretLines()
	if err != nil {
		return err
	}
	for _, l := range lines {
		fmt.Println(l)
	}
	return nil
}
