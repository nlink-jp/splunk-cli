package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Stderr is the writer for warnings. Overridable in tests.
var Stderr io.Writer = os.Stderr

// Config holds all runtime configuration for splunk-cli.
// Fields are populated in priority order: config file → env vars → CLI flags.
type Config struct {
	Host        string
	Token       string
	User        string
	Password    string
	App         string
	Owner       string
	Insecure    bool
	HTTPTimeout time.Duration
	Limit       int
	Debug       bool
}

// tomlConfig mirrors the TOML file structure.
type tomlConfig struct {
	Splunk tomlSplunk `toml:"splunk"`
}

type tomlSplunk struct {
	Host        string `toml:"host"`
	Token       string `toml:"token"`
	User        string `toml:"user"`
	Password    string `toml:"password"`
	App         string `toml:"app"`
	Owner       string `toml:"owner"`
	Insecure    bool   `toml:"insecure"`
	HTTPTimeout string `toml:"http_timeout"`
	Limit       int    `toml:"limit"`
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "splunk-cli", "config.toml")
}

// Load reads the config file at path (or the default path if empty).
// A missing file is not an error — the returned Config will have zero values.
func Load(path string) (Config, error) {
	var cfg Config

	if path == "" {
		path = DefaultPath()
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("config: stat %s: %w", path, err)
	}

	checkPermissions(path, info)

	var raw tomlConfig
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return cfg, fmt.Errorf("config: parse %s: %w", path, err)
	}

	s := raw.Splunk
	cfg.Host = strings.TrimSpace(s.Host)
	cfg.Token = strings.TrimSpace(s.Token)
	cfg.User = strings.TrimSpace(s.User)
	cfg.Password = strings.TrimSpace(s.Password)
	cfg.App = strings.TrimSpace(s.App)
	cfg.Owner = strings.TrimSpace(s.Owner)
	cfg.Insecure = s.Insecure
	cfg.Limit = s.Limit

	if s.HTTPTimeout != "" {
		d, err := time.ParseDuration(s.HTTPTimeout)
		if err != nil {
			return cfg, fmt.Errorf("config: invalid http_timeout %q: %w", s.HTTPTimeout, err)
		}
		cfg.HTTPTimeout = d
	}

	return cfg, nil
}

// ApplyEnvVars overrides cfg fields with values from environment variables.
func ApplyEnvVars(cfg *Config) {
	if v := os.Getenv("SPLUNK_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("SPLUNK_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("SPLUNK_USER"); v != "" {
		cfg.User = v
	}
	if v := os.Getenv("SPLUNK_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := os.Getenv("SPLUNK_APP"); v != "" {
		cfg.App = v
	}
}

func checkPermissions(path string, info os.FileInfo) {
	if info.Mode().Perm()&0077 != 0 {
		fmt.Fprintf(Stderr,
			"Warning: config file %s has permissions %#o; expected 0600.\n"+
				"  The file may contain credentials. Run: chmod 600 %s\n",
			path, info.Mode().Perm(), path,
		)
	}
}
