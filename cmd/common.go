package cmd

import (
	"errors"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// getSPL reads the SPL query from --spl or --file (mutually exclusive).
func getSPL(cmd *cobra.Command) (string, error) {
	spl, _ := cmd.Flags().GetString("spl")
	file, _ := cmd.Flags().GetString("file")

	if spl != "" {
		return spl, nil
	}
	if file != "" {
		var b []byte
		var err error
		if file == "-" {
			b, err = io.ReadAll(os.Stdin)
		} else {
			b, err = os.ReadFile(file)
		}
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", errors.New("--spl or --file is required")
}
