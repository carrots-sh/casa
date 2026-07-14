package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

// AddSecret starts managing a file, encrypted.
func AddSecret(path string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if path == "" {
		var err error
		if path, err = ui.Input("path of the file to encrypt + manage"); err != nil || path == "" {
			return err
		}
	}
	if err := chez.AddEncrypt(expand(path)); err != nil {
		return err
	}
	fmt.Printf("✓ encrypted and now managing %s\n", path)
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
	source := bySource[sel]
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
	if err := chez.EncryptInto(string(edited), source); err != nil {
		return err
	}
	_ = chez.ApplyNoScripts() // re-render any targets assembled from this secret
	fmt.Printf("✓ updated secret %s\n", sel)
	offerSave("casa: edit secret")
	return nil
}

// targetLabels converts source paths to readable target paths, falling back to
// the sources themselves if the conversion fails or is incomplete.
func targetLabels(sources []string) []string {
	disp, err := chez.TargetPaths(sources)
	if err != nil || len(disp) != len(sources) {
		return sources
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

// ListSecrets prints the encrypted files by their readable target paths.
func ListSecrets() error {
	enc, err := chez.EncryptedSources()
	if err != nil {
		return err
	}
	disp, _ := displayNames(enc)
	for _, d := range disp {
		fmt.Println(d)
	}
	return nil
}
