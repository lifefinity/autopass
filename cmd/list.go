package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured profiles",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	d, err := loadData()
	if err != nil {
		return err
	}

	names := d.Profiles.ListProfiles()
	if len(names) == 0 {
		fmt.Println("No profiles configured.")
		fmt.Println()
		printExamples()
		return nil
	}

	// Calculate column widths
	maxName, maxCmd := len("NAME"), len("COMMAND")
	for name, profile := range d.Profiles.Entries {
		if len(name) > maxName {
			maxName = len(name)
		}
		if len(profile.Command) > maxCmd {
			maxCmd = len(profile.Command)
		}
	}

	header := fmt.Sprintf("  %-*s  %-*s  %s", maxName, "NAME", maxCmd, "COMMAND", "DESCRIPTION")
	fmt.Println(header)
	fmt.Println("  " + strings.Repeat("-", len(header)-2))
	for _, name := range names {
		profile := d.Profiles.Entries[name]
		desc := profile.Description
		if desc == "" {
			// Fallback: show matched patterns
			prompts := []string{}
			for _, p := range profile.Patterns {
				prompts = append(prompts, friendlyPattern(p.Match))
			}
			desc = strings.Join(prompts, ", ")
		}
		fmt.Printf("  %-*s  %-*s  %s\n", maxName, name, maxCmd, profile.Command, desc)
	}
	fmt.Println()
	fmt.Println("Run: autopass <name>")
	fmt.Println()

	return nil
}

func friendlyPattern(match string) string {
	// Strip common regex prefixes to show a readable prompt description
	s := match
	if len(s) > 4 && s[:4] == "(?i)" {
		s = s[4:]
	}
	// Remove regex escapes for display
	replacer := strings.NewReplacer(`\(`, "(", `\)`, ")", `\[`, "[", `\]`, "]")
	s = replacer.Replace(s)
	return s
}

func printExamples() {
	fmt.Println("Get started with 'autopass add <name>'. Examples:")
	fmt.Println()
	fmt.Println("  autopass add mwinit       # Midway (Amazon)")
	fmt.Println("  autopass add myserver     # SSH to a host")
	fmt.Println("  autopass add mysudo       # Sudo commands")
	fmt.Println("  autopass add docker       # Docker login")
	fmt.Println("  autopass add aws-sso      # AWS SSO login")
	fmt.Println("  autopass add mysql        # MySQL client")
	fmt.Println("  autopass add git-push     # Git HTTPS credential")
	fmt.Println()
	fmt.Println("Or with flags:")
	fmt.Println("  autopass add -c \"ssh user@host\" -m \"password:\" myserver")
	fmt.Println()
	fmt.Println("Then run: autopass <name>")
}
