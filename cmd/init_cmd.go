package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/lifefinity/autopass/internal/crypto"
	"github.com/lifefinity/autopass/internal/data"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize autopass (first-time setup)",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	autopassDir := filepath.Join(home, ".autopass")
	dataFilePath := filepath.Join(autopassDir, "data.json")

	if _, err := os.Stat(dataFilePath); err == nil {
		fmt.Println("autopass is already initialized. Data at:", dataFilePath)
		return nil
	}

	sshKey := findSSHKey(home)
	if sshKey == "" {
		return fmt.Errorf("no SSH key found in ~/.ssh/. Generate one with: ssh-keygen -t ed25519")
	}

	fmt.Printf("Using SSH key: %s\n", sshKey)

	// Verify we can read the key (may need passphrase)
	_, err = crypto.DeriveKey(sshKey, nil)
	if err != nil {
		fmt.Print("Enter SSH key passphrase: ")
		passphrase, readErr := term.ReadPassword(int(os.Stdin.Fd())) // #nosec G115
		fmt.Println()
		if readErr != nil {
			return fmt.Errorf("reading passphrase: %w", readErr)
		}

		_, err = crypto.DeriveKey(sshKey, passphrase)
		if err != nil {
			return fmt.Errorf("cannot derive key from SSH key: %w", err)
		}
	}

	d := &data.Data{
		SSHKey:   sshKey,
		Profiles: make(map[string]data.Profile),
	}

	if err := data.Save(dataFilePath, d); err != nil {
		return fmt.Errorf("writing data file: %w", err)
	}

	fmt.Println("Initialized. Data at:", dataFilePath)
	fmt.Println("Next: run 'autopass add <name>' to store a secret.")
	return nil
}

func findSSHKey(home string) string {
	sshDir := filepath.Join(home, ".ssh")
	candidates := []string{"id_ed25519", "id_rsa", "id_ecdsa"}

	for _, name := range candidates {
		path := filepath.Join(sshDir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
