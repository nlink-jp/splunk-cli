package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a SPL search and print results (synchronous)",
	Long: `Start a Splunk search, wait for it to complete, and print results as JSON.

Press Ctrl+C to cancel the job or detach and let it run in the background.`,
	RunE: runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
	f := runCmd.Flags()
	f.String("spl", "", "SPL query to execute")
	f.StringP("file", "f", "", "Read SPL from a file (use '-' for stdin)")
	f.String("earliest", "", "Earliest time filter (e.g. -1h, @d)")
	f.String("latest", "", "Latest time filter (e.g. now, @d)")
	f.Duration("timeout", 10*time.Minute, "Total timeout for the run command")
	f.Bool("silent", false, "Suppress progress messages")
	runCmd.MarkFlagsMutuallyExclusive("spl", "file")
}

func runRun(cmd *cobra.Command, _ []string) error {
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
	timeout, _ := cmd.Flags().GetDuration("timeout")
	silent, _ := cmd.Flags().GetBool("silent")

	c, err := newClient(silent)
	if err != nil {
		return err
	}

	c.Logf("Connecting to Splunk and starting search...\n")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sid, err := c.StartSearch(ctx, spl, earliest, latest)
	if err != nil {
		return err
	}
	c.Logf("Job started: %s\n", sid)

	// Wait for completion, handling Ctrl+C.
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	doneCh := make(chan error, 1)
	go func() { doneCh <- c.WaitForJob(ctx, sid) }()

	select {
	case waitErr := <-doneCh:
		if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
			if errors.Is(waitErr, context.DeadlineExceeded) {
				return fmt.Errorf("timed out after %v", timeout)
			}
			return waitErr
		}
	case <-sigCh:
		signal.Stop(sigCh)
		fmt.Fprintln(os.Stderr, "\n^C detected. What would you like to do?")
		fmt.Fprintln(os.Stderr, "  (c)ancel the job on Splunk")
		fmt.Fprintln(os.Stderr, "  (d)etach and let it run in the background")
		fmt.Fprint(os.Stderr, "Choice [c/d]: ")

		choiceCh := make(chan string, 1)
		go func() { choiceCh <- readFromTTY() }()

		secondSig := make(chan os.Signal, 1)
		signal.Notify(secondSig, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(secondSig)

		select {
		case choice := <-choiceCh:
			if strings.ToLower(choice) == "d" {
				fmt.Fprintf(os.Stderr, "Detached from %s. Use 'results --sid %s' to fetch results.\n", sid, sid)
				return nil
			}
		case <-secondSig:
		}
		return c.CancelSearch(context.Background(), sid)
	}

	c.Logf("Fetching results...\n")
	status, err := c.GetJobStatus(ctx, sid)
	if err != nil {
		return err
	}
	results, err := c.Results(ctx, sid, cfg.Limit, status.ResultCount)
	if err != nil {
		return err
	}
	fmt.Println(results)
	return nil
}

// readFromTTY reads a line from /dev/tty (Unix) or stdin (Windows).
func readFromTTY() string {
	var r io.Reader
	if runtime.GOOS != "windows" {
		tty, err := os.Open("/dev/tty")
		if err == nil {
			defer func() { _ = tty.Close() }()
			r = tty
		}
	}
	if r == nil {
		r = os.Stdin
	}
	line, _ := bufio.NewReader(r).ReadString('\n')
	return strings.TrimSpace(line)
}
