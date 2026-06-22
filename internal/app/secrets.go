package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
func EditSecret() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	enc, err := chez.EncryptedSources()
	if err != nil {
		return err
	}
	if len(enc) == 0 {
		return fmt.Errorf("no encrypted files in this repo")
	}
	disp, bySource := displayNames(enc)
	sel, err := ui.Select("edit which secret?", disp)
	if err != nil || sel == "" {
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
	c := exec.Command(editor, tmp.Name())
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return err
	}
	edited, err := os.ReadFile(tmp.Name())
	if err != nil {
		return err
	}
	if err := chez.EncryptInto(string(edited), source); err != nil {
		return err
	}
	_ = chez.ApplyNoScripts() // re-render any targets assembled from this secret
	fmt.Printf("✓ updated secret %s\n", sel)
	offerSave("casa: edit secret")
	return nil
}

// displayNames maps encrypted source paths to readable target-style names,
// returning the labels (in order) and a label→source lookup. Falls back to the
// source paths if the conversion fails.
func displayNames(sources []string) ([]string, map[string]string) {
	disp, err := chez.TargetPaths(sources)
	if err != nil || len(disp) != len(sources) {
		disp = sources
	}
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
