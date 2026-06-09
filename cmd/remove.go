package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lifefinity/autopass/internal/data"
)

var removeCmd = &cobra.Command{
	Use:   "remove <profile>",
	Short: "Delete a profile and its secret",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeProfileNames(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	path, err := dataPath()
	if err != nil {
		return err
	}

	d, err := data.Load(path)
	if err != nil {
		return fmt.Errorf("loading data: %w", err)
	}

	if err := d.Profiles.RemoveProfile(name); err != nil {
		return err
	}

	if err := data.Save(path, d); err != nil {
		return fmt.Errorf("saving data: %w", err)
	}

	fmt.Printf("Removed %q\n", name)
	return nil
}
