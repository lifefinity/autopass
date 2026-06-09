package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/term"

	"github.com/lifefinity/autopass/internal/crypto"
	"github.com/lifefinity/autopass/internal/data"
)

func dataPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".autopass", "data.json"), nil
}

func loadData() (*data.Data, error) {
	path, err := dataPath()
	if err != nil {
		return nil, err
	}

	d, err := data.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading data: %w", err)
	}

	// If data.json doesn't exist on disk, auto-initialize
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		fmt.Println("autopass is not initialized. Running setup now...")
		if initErr := runInit(nil, nil); initErr != nil {
			return nil, fmt.Errorf("auto-initialization failed: %w", initErr)
		}
		d, err = data.Load(path)
		if err != nil {
			return nil, fmt.Errorf("loading data after init: %w", err)
		}
	}

	return d, nil
}

func deriveKey() ([]byte, error) {
	d, err := loadData()
	if err != nil {
		return nil, err
	}

	// Priority: key_command > key_file
	if d.Config.KeyCommand != "" {
		return deriveKeyFromCommand(d.Config.KeyCommand)
	}

	home, _ := os.UserHomeDir()
	keyFilePath := d.Config.KeyFile
	if len(keyFilePath) > 2 && keyFilePath[:2] == "~/" {
		keyFilePath = filepath.Join(home, keyFilePath[2:])
	}

	key, err := crypto.DeriveKey(keyFilePath, nil)
	if err != nil {
		fmt.Print("Enter SSH key passphrase: ")
		passphrase, readErr := term.ReadPassword(int(os.Stdin.Fd())) // #nosec G115
		fmt.Println()
		if readErr != nil {
			return nil, fmt.Errorf("reading passphrase: %w", readErr)
		}
		key, err = crypto.DeriveKey(keyFilePath, passphrase)
		if err != nil {
			return nil, fmt.Errorf("deriving key: %w", err)
		}
	}

	return key, nil
}

func deriveKeyFromCommand(command string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", command) // #nosec G204 -- user-configured key command
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("key_command failed: %w", err)
	}

	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 {
		return nil, fmt.Errorf("key_command returned empty output")
	}

	key, err := crypto.DeriveKeyFromBytes(raw)
	// Zero the raw material
	for i := range raw {
		raw[i] = 0
	}
	if err != nil {
		return nil, err
	}

	return key, nil
}
