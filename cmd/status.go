package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of a search job",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().String("sid", "", "Search ID (SID) to check")
	_ = statusCmd.MarkFlagRequired("sid")
}

func runStatus(cmd *cobra.Command, _ []string) error {
	sid, _ := cmd.Flags().GetString("sid")
	if err := requireHost(); err != nil {
		return err
	}
	if err := promptForCredentials(); err != nil {
		return err
	}

	c, err := newClient(false)
	if err != nil {
		return err
	}

	status, err := c.GetJobStatus(context.Background(), sid)
	if err != nil {
		return err
	}
	fmt.Printf("SID: %s\nIsDone: %t\nDispatchState: %s\n", status.SID, status.IsDone, status.DispatchState)
	return nil
}
