package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "autopass",
	Short:   "Automated interactive prompt responder",
	Long:    "Wraps commands in a PTY, matches output patterns, and responds with decrypted secrets.",
	Version: Version,
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		profileName, runOpts := parseProfileArgs(args)
		return runProfileWithSteps(profileName, runOpts)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type profileRunOpts struct {
	then       []string
	scriptFile string
	prompt     string
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
