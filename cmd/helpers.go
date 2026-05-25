package cmd

import (
	"fmt"
	"os"
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

	home, _ := os.UserHomeDir()
	sshKeyPath := d.SSHKey
	if len(sshKeyPath) > 2 && sshKeyPath[:2] == "~/" {
		sshKeyPath = filepath.Join(home, sshKeyPath[2:])
	}

	key, err := crypto.DeriveKey(sshKeyPath, nil)
	if err != nil {
		fmt.Print("Enter SSH key passphrase: ")
		passphrase, readErr := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if readErr != nil {
			return nil, fmt.Errorf("reading passphrase: %w", readErr)
		}
		key, err = crypto.DeriveKey(sshKeyPath, passphrase)
		if err != nil {
			return nil, fmt.Errorf("deriving key: %w", err)
		}
	}

	return key, nil
}
