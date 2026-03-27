package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a SPL search asynchronously and print the SID",
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	f := startCmd.Flags()
	f.String("spl", "", "SPL query to execute")
	f.StringP("file", "f", "", "Read SPL from a file (use '-' for stdin)")
	f.String("earliest", "", "Earliest time filter")
	f.String("latest", "", "Latest time filter")
	startCmd.MarkFlagsMutuallyExclusive("spl", "file")
}

func runStart(cmd *cobra.Command, _ []string) error {
	spl, err := getSPL(cmd)
	if err != nil {
		return err
	}
	if err := requireHost(); err != nil {
		return err
	}
	if err := promptForCredentials(); err != nil {
		return err
	}

	earliest, _ := cmd.Flags().GetString("earliest")
	latest, _ := cmd.Flags().GetString("latest")

	c, err := newClient(true)
	if err != nil {
		return err
	}

	sid, err := c.StartSearch(context.Background(), spl, earliest, latest)
	if err != nil {
		return err
	}
	fmt.Println(sid)
	return nil
}
