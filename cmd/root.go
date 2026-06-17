package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "autopass",
	Short: "Automated interactive prompt responder",
	Long: `Wraps commands in a PTY, matches output patterns, and responds with decrypted secrets.

Run a profile:
  autopass <profile>              Run with auto-answering
  autopass <profile> -s <service> Run specific service profile
  autopass <profile> --then "cmd" Execute command after login
  autopass <profile> --script f   Execute commands from file after login
  autopass <profile> --prompt "x" Override shell prompt pattern
  autopass <profile> -e K=V       Inject environment variable
  autopass <profile> --after cmd  Run command in new shell after profile exits
  autopass <profile> --quiet      Suppress terminal output

Examples:
  autopass mwinit                            # Auto-fill PIN for mwinit
  autopass mydb --then "SELECT now();"       # Run SQL after connecting
  autopass prod --script deploy.sh           # Run script after login
  autopass mydb --then "\timing" --then "\q" # Chain multiple commands
  autopass deploy -e HOST=prod.example.com   # Inject env var
  autopass mydb -s staging                   # Run the 'staging' service variant`,
	Version: Version,
	Args:    cobra.ArbitraryArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeProfileNames(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		profileName, runOpts := parseProfileArgs(args)
		return runProfileWithSteps(profileName, runOpts)
	},
}

var serviceFlag string

func Execute() {
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "bypass keychain cache for key derivation")
	rootCmd.PersistentFlags().StringVarP(&serviceFlag, "service", "s", "", "service name for profile disambiguation")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type profileRunOpts struct {
	then       []string
	scriptFile string
	prompt     string
	quiet      bool
	dryRun     bool
	env        []string
	after      []string
}

func parseProfileArgs(args []string) (string, profileRunOpts) {
	profileName := args[0]
	var opts profileRunOpts

	i := 1
	for i < len(args) {
		switch args[i] {
		case "--then":
			if i+1 < len(args) {
				opts.then = append(opts.then, args[i+1])
				i += 2
			} else {
				i++
			}
		case "--after":
			if i+1 < len(args) {
				opts.after = append(opts.after, args[i+1])
				i += 2
			} else {
				i++
			}
		case "--script":
			if i+1 < len(args) {
				opts.scriptFile = args[i+1]
				i += 2
			} else {
				i++
			}
		case "--prompt":
			if i+1 < len(args) {
				opts.prompt = args[i+1]
				i += 2
			} else {
				i++
			}
		case "--env", "-e":
			if i+1 < len(args) {
				opts.env = append(opts.env, args[i+1])
				i += 2
			} else {
				i++
			}
		case "--quiet", "-q":
			opts.quiet = true
			i++
		case "--dry-run":
			opts.dryRun = true
			i++
		default:
			i++
		}
	}

	return profileName, opts
}

func loadSteps(opts profileRunOpts) ([]string, error) {
	var steps []string

	steps = append(steps, opts.then...)

	if opts.scriptFile != "" {
		lines, err := readScriptFile(opts.scriptFile)
		if err != nil {
			return nil, err
		}
		steps = append(steps, lines...)
	}

	return steps, nil
}

func readScriptFile(path string) ([]string, error) {
	f, err := os.Open(path) // #nosec G304 -- path is user-provided script file
	if err != nil {
		return nil, fmt.Errorf("opening script file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading script file: %w", err)
	}
	return lines, nil
}

func completeProfileNames(prefix string) []string {
	d, err := loadData()
	if err != nil {
		return nil
	}
	names := d.Profiles.ListProfiles()
	if prefix == "" {
		return names
	}
	var filtered []string
	for _, name := range names {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			filtered = append(filtered, name)
		}
	}
	return filtered
}
