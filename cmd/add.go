package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/lifefinity/autopass/internal/crypto"
	"github.com/lifefinity/autopass/internal/data"
)

var (
	addCommand string
	addMatch   string
	addPrompt  string
	addTimeout string
)

var addCmd = &cobra.Command{
	Use:   "add <profile>",
	Short: "Store a secret and create a profile",
	Long: `Store an encrypted secret and create a named profile that auto-answers
prompts. Run with 'autopass <profile>' afterwards.

Examples:
  # SSH server
  autopass add -c "ssh deploy@prod-server" -m "(?i)password:" prod

  # PostgreSQL
  autopass add -c "psql -h db.example.com -U admin mydb" -m "(?i)password" mydb

  # MySQL
  autopass add -c "mysql -h db.example.com -u root -p" -m "(?i)password:" mysql-prod

  # Sudo
  autopass add -c "sudo apt upgrade -y" -m "(?i)password" apt-upgrade

  # Docker registry
  autopass add -c "docker login registry.example.com -u ci" -m "(?i)password:" docker-reg

  # Kerberos
  autopass add -c "kinit admin@EXAMPLE.COM" -m "(?i)password for" krb

  # Redis CLI
  autopass add -c "redis-cli -h cache.example.com" -m "(?i)password:" redis

  # FTP
  autopass add -c "ftp files.example.com" -m "(?i)password:" ftp-files

  # Interactive mode (prompts for command and pattern)
  autopass add myservice`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&addCommand, "command", "c", "", "command to run for this profile")
	addCmd.Flags().StringVarP(&addMatch, "match", "m", "", "prompt pattern to match (regex)")
	addCmd.Flags().StringVarP(&addPrompt, "prompt", "p", "", "shell prompt pattern for post-login steps (regex)")
	addCmd.Flags().StringVarP(&addTimeout, "timeout", "t", "30s", "timeout for pattern matching")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	// If -c not provided, prompt interactively
	command := addCommand
	if command == "" {
		fmt.Printf("Command to run (e.g. \"ssh user@host\"): ")
		command = readLine()
		if command == "" {
			return fmt.Errorf("command is required")
		}
	}

	// If -m not provided, prompt interactively
	match := addMatch
	if match == "" {
		fmt.Printf("Prompt to match (e.g. \"password:\") [default: (?i)password:]: ")
		match = readLine()
		if match == "" {
			match = "(?i)password:"
		}
	}

	key, err := deriveKey()
	if err != nil {
		return err
	}

	fmt.Printf("Enter secret (will be hidden): ")
	secret, err := term.ReadPassword(int(os.Stdin.Fd())) // #nosec G115
	fmt.Println()
	if err != nil {
		return fmt.Errorf("reading secret: %w", err)
	}

	ciphertext, err := crypto.Encrypt(key, secret)
	if err != nil {
		return fmt.Errorf("encrypting secret: %w", err)
	}
	encryptedB64 := base64.StdEncoding.EncodeToString(ciphertext)

	timeout, _ := time.ParseDuration(addTimeout)
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	path, err := dataPath()
	if err != nil {
		return err
	}

	d, err := data.Load(path)
	if err != nil {
		return fmt.Errorf("loading data: %w", err)
	}

	profile := data.Profile{
		Command: command,
		Patterns: []data.Pattern{
			{Match: match, Hidden: true},
		},
		Secret:  encryptedB64,
		Prompt:  addPrompt,
		Timeout: data.Duration{Duration: timeout},
	}

	if err := d.AddProfile(name, profile); err != nil {
		return err
	}

	if err := data.Save(path, d); err != nil {
		return fmt.Errorf("saving data: %w", err)
	}

	fmt.Println()
	fmt.Printf("Done! Run with: autopass %s\n", name)
	return nil
}

func readLine() string {
	var buf [512]byte
	n, _ := os.Stdin.Read(buf[:])
	line := string(buf[:n])
	line = strings.TrimRight(line, "\r\n")
	return line
}
