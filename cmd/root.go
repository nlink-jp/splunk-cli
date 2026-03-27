package cmd

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/nlink-jp/splunk-cli/internal/client"
	"github.com/nlink-jp/splunk-cli/internal/config"
)

// cfg holds the runtime configuration, built by loadConfig in PersistentPreRunE.
var cfg config.Config

// persistent flag destinations
var (
	cfgFile         string
	flagHost        string
	flagToken       string
	flagUser        string
	flagPassword    string
	flagApp         string
	flagOwner       string
	flagInsecure    bool
	flagHTTPTimeout time.Duration
	flagDebug       bool
	flagLimit       int
)

var rootCmd = &cobra.Command{
	Use:          "splunk-cli",
	Short:        "CLI client for the Splunk REST API",
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentPreRunE = loadConfig

	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&cfgFile, "config", "c", "", "Config file (default: ~/.config/splunk-cli/config.toml)")
	pf.StringVar(&flagHost, "host", "", "Splunk server URL (env: SPLUNK_HOST)")
	pf.StringVar(&flagToken, "token", "", "Bearer token (env: SPLUNK_TOKEN)")
	pf.StringVar(&flagUser, "user", "", "Username for basic auth (env: SPLUNK_USER)")
	pf.StringVar(&flagPassword, "password", "", "Password (env: SPLUNK_PASSWORD)")
	pf.StringVar(&flagApp, "app", "", "App context for searches (env: SPLUNK_APP)")
	pf.StringVar(&flagOwner, "owner", "", "Knowledge object owner (default: nobody)")
	pf.BoolVar(&flagInsecure, "insecure", false, "Skip TLS certificate verification")
	pf.DurationVar(&flagHTTPTimeout, "http-timeout", 0, "Per-request HTTP timeout (e.g. 30s, 2m)")
	pf.BoolVar(&flagDebug, "debug", false, "Enable verbose debug logging")
	pf.IntVar(&flagLimit, "limit", 0, "Max results to return (0 = all)")
}

// Execute runs the root command.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// loadConfig is PersistentPreRunE: file → env vars → flags.
func loadConfig(_ *cobra.Command, _ []string) error {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		return err
	}
	config.ApplyEnvVars(&cfg)

	pf := rootCmd.PersistentFlags()
	if pf.Changed("host") {
		cfg.Host = flagHost
	}
	if pf.Changed("token") {
		cfg.Token = flagToken
	}
	if pf.Changed("user") {
		cfg.User = flagUser
	}
	if pf.Changed("password") {
		cfg.Password = flagPassword
	}
	if pf.Changed("app") {
		cfg.App = flagApp
	}
	if pf.Changed("owner") {
		cfg.Owner = flagOwner
	}
	if pf.Changed("insecure") {
		cfg.Insecure = flagInsecure
	}
	if pf.Changed("http-timeout") {
		cfg.HTTPTimeout = flagHTTPTimeout
	}
	if pf.Changed("debug") {
		cfg.Debug = flagDebug
	}
	if pf.Changed("limit") {
		cfg.Limit = flagLimit
	}
	return nil
}

// newClient creates a Splunk client from the current cfg.
func newClient(silent bool) (*client.Client, error) {
	return client.New(&cfg, silent)
}

// promptForCredentials ensures auth credentials are available.
func promptForCredentials() error {
	if cfg.Token != "" || (cfg.User != "" && cfg.Password != "") {
		return nil
	}
	if cfg.User == "" {
		fmt.Fprintln(os.Stderr, "No authentication credentials provided.")
		fmt.Fprint(os.Stderr, "Enter Splunk authentication token: ")
		b, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		cfg.Token = string(b)
		fmt.Fprintln(os.Stderr)
	} else {
		fmt.Fprintf(os.Stderr, "Enter Splunk password for %q: ", cfg.User)
		b, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		cfg.Password = string(b)
		fmt.Fprintln(os.Stderr)
	}
	return nil
}

// requireHost returns an error if no host is configured.
func requireHost() error {
	if cfg.Host == "" {
		return fmt.Errorf("Splunk host is required (--host, SPLUNK_HOST env var, or host in config file)")
	}
	return nil
}
