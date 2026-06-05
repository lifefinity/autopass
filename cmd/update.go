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
	updateCommand       string
	updateMatch         []string
	updatePrompt        string
	updateTimeout       string
	updateSecret        bool
	updateCaseSensitive bool
	updateSteps         []string
	updateAfter         []string
	updateDesc          string
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
  autopass update mwinit -m "PIN:" -t 60s

  # Update multiple fields at once
  autopass update myserver -c "ssh newuser@host" -m "password:" --secret`,
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeProfileNames(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().StringVarP(&updateCommand, "command", "c", "", "update the command")
	updateCmd.Flags().StringVarP(&updateDesc, "desc", "d", "", "update the description")
	updateCmd.Flags().StringArrayVarP(&updateMatch, "match", "m", nil, "update the prompt pattern (regex, can specify multiple)")
	updateCmd.Flags().StringVarP(&updatePrompt, "prompt", "p", "", "update the shell prompt pattern for post-login steps")
	updateCmd.Flags().StringVarP(&updateTimeout, "timeout", "t", "", "update the timeout (e.g. 30s, 1m)")
	updateCmd.Flags().BoolVar(&updateSecret, "secret", false, "prompt for a new secret")
	updateCmd.Flags().BoolVar(&updateCaseSensitive, "case-sensitive", false, "match pattern with exact case (default: case-insensitive)")
	updateCmd.Flags().StringArrayVar(&updateSteps, "then", nil, "update post-login commands (replaces existing)")
	updateCmd.Flags().StringArrayVar(&updateAfter, "after", nil, "update post-exit commands (replaces existing)")
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
		cmd.Flags().Changed("desc") ||
		cmd.Flags().Changed("match") ||
		cmd.Flags().Changed("prompt") ||
		cmd.Flags().Changed("timeout") ||
		cmd.Flags().Changed("case-sensitive") ||
		cmd.Flags().Changed("then") ||
		cmd.Flags().Changed("after") ||
		updateSecret

	if !anyChange {
		return fmt.Errorf("no flags specified. Use --help to see available options")
	}

	// Update command
	if cmd.Flags().Changed("command") {
		profile.Command = updateCommand
	}

	// Update description
	if cmd.Flags().Changed("desc") {
		profile.Description = updateDesc
	}

	// Update match pattern
	if cmd.Flags().Changed("match") {
		cs := updateCaseSensitive
		if !cmd.Flags().Changed("case-sensitive") && len(profile.Patterns) > 0 {
			cs = profile.Patterns[0].CaseSensitive
		}
		patterns := make([]data.Pattern, len(updateMatch))
		for i, m := range updateMatch {
			patterns[i] = data.Pattern{Match: m, Hidden: true, CaseSensitive: cs}
		}
		profile.Patterns = patterns
	} else if cmd.Flags().Changed("case-sensitive") && len(profile.Patterns) > 0 {
		for i := range profile.Patterns {
			profile.Patterns[i].CaseSensitive = updateCaseSensitive
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

	// Update steps
	if cmd.Flags().Changed("then") {
		profile.Steps = updateSteps
	}

	// Update after
	if cmd.Flags().Changed("after") {
		profile.After = updateAfter
	}

	// Update secret
	if updateSecret {
		key, keyErr := deriveKey()
		if keyErr != nil {
			return keyErr
		}

		fmt.Printf("Enter new secret (will be hidden): ")
		secret, readErr := term.ReadPassword(int(os.Stdin.Fd())) // #nosec G115
		fmt.Println()
		if readErr != nil {
			return fmt.Errorf("reading secret: %w", readErr)
		}
		defer func() {
			for i := range secret {
				secret[i] = 0
			}
		}()

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
