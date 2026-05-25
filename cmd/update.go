package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/lifefinity/autopass/internal/crypto"
	"github.com/lifefinity/autopass/internal/data"
)

var (
	updateCommand string
	updateMatch   string
	updatePrompt  string
	updateTimeout string
	updateSecret  bool
)

var updateCmd = &cobra.Command{
	Use:   "update <profile>",
	Short: "Update an existing profile",
	Long: `Update fields of an existing profile. Only specified flags are changed;
unspecified fields remain unchanged.

Examples:
  # Update only the secret
  autopass update mwinit --secret

  # Update the command
  autopass update mwinit -c "mwinit -s -o -f"

  # Update match pattern and timeout
  autopass update mwinit -m "(?i)PIN:" -t 60s

  # Update multiple fields at once
  autopass update myserver -c "ssh newuser@host" -m "(?i)password:" --secret`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().StringVarP(&updateCommand, "command", "c", "", "update the command")
	updateCmd.Flags().StringVarP(&updateMatch, "match", "m", "", "update the prompt pattern (regex)")
	updateCmd.Flags().StringVarP(&updatePrompt, "prompt", "p", "", "update the shell prompt pattern for post-login steps")
	updateCmd.Flags().StringVarP(&updateTimeout, "timeout", "t", "", "update the timeout (e.g. 30s, 1m)")
	updateCmd.Flags().BoolVar(&updateSecret, "secret", false, "prompt for a new secret")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	path, err := dataPath()
	if err != nil {
		return err
	}

	d, err := data.Load(path)
	if err != nil {
		return fmt.Errorf("loading data: %w", err)
	}

	profile, ok := d.Profiles[profileName]
	if !ok {
		return fmt.Errorf("profile %q not found", profileName)
	}

	// Check if any flag was provided
	anyChange := cmd.Flags().Changed("command") ||
		cmd.Flags().Changed("match") ||
		cmd.Flags().Changed("prompt") ||
		cmd.Flags().Changed("timeout") ||
		updateSecret

	if !anyChange {
		return fmt.Errorf("no flags specified. Use --help to see available options")
	}

	// Update command
	if cmd.Flags().Changed("command") {
		profile.Command = updateCommand
	}

	// Update match pattern
	if cmd.Flags().Changed("match") {
		profile.Patterns = []data.Pattern{
			{Match: updateMatch, Hidden: true},
		}
	}

	// Update prompt
	if cmd.Flags().Changed("prompt") {
		profile.Prompt = updatePrompt
	}

	// Update timeout
	if cmd.Flags().Changed("timeout") {
		timeout, parseErr := time.ParseDuration(updateTimeout)
		if parseErr != nil {
			return fmt.Errorf("invalid timeout %q: %w", updateTimeout, parseErr)
		}
		profile.Timeout = data.Duration{Duration: timeout}
	}

	// Update secret
	if updateSecret {
		key, keyErr := deriveKey()
		if keyErr != nil {
			return keyErr
		}

		fmt.Printf("Enter new secret (will be hidden): ")
		secret, readErr := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if readErr != nil {
			return fmt.Errorf("reading secret: %w", readErr)
		}

		ciphertext, encErr := crypto.Encrypt(key, secret)
		if encErr != nil {
			return fmt.Errorf("encrypting secret: %w", encErr)
		}
		profile.Secret = base64.StdEncoding.EncodeToString(ciphertext)
	}

	d.Profiles[profileName] = profile

	if err := data.Save(path, d); err != nil {
		return fmt.Errorf("saving data: %w", err)
	}

	fmt.Printf("Updated profile %q\n", profileName)
	return nil
}
