package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var resultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Fetch results of a completed search job",
	RunE:  runResults,
}

func init() {
	rootCmd.AddCommand(resultsCmd)
	resultsCmd.Flags().String("sid", "", "Search ID (SID) to fetch results for")
	resultsCmd.Flags().Bool("silent", false, "Suppress progress messages")
	_ = resultsCmd.MarkFlagRequired("sid")
}

func runResults(cmd *cobra.Command, _ []string) error {
	sid, _ := cmd.Flags().GetString("sid")
	silent, _ := cmd.Flags().GetBool("silent")

	if err := requireHost(); err != nil {
		return err
	}
	if err := promptForCredentials(); err != nil {
		return err
	}

	c, err := newClient(silent)
	if err != nil {
		return err
	}

	ctx := context.Background()
	status, err := c.GetJobStatus(ctx, sid)
	if err != nil {
		return err
	}
	if !status.IsDone {
		return fmt.Errorf("job %s is not complete yet (state: %s)", sid, status.DispatchState)
	}
	if status.DispatchState == "FAILED" {
		return fmt.Errorf("job %s failed", sid)
	}

	c.Logf("Fetching results...\n")
	results, err := c.Results(ctx, sid, cfg.Limit, status.ResultCount)
	if err != nil {
		return err
	}
	fmt.Println(results)
	return nil
}
