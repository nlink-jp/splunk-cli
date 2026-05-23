// Package spl contains pure helpers for shaping SPL queries before they are
// sent to Splunk's REST API.
package spl

import (
	"fmt"
	"strings"
)

// PrependMode controls how Wrap decides whether to add a leading "search "
// command to a user-supplied SPL string.
type PrependMode string

const (
	// ModeAuto skips the "search " prefix when the input already starts with
	// the "search" command (followed by whitespace or end-of-string) or with
	// a "|" pipe leader. Everything else (bare terms, quoted phrases,
	// backtick macros) gets the prefix.
	ModeAuto PrependMode = "auto"

	// ModePipeOnly skips the prefix only when the input starts with "|".
	// This matches the historical default — passing `search index=foo`
	// produces the double-search artifact `search search index=foo`.
	ModePipeOnly PrependMode = "pipe-only"

	// ModeOff never prepends. The caller must supply a complete SPL,
	// including any leading "search" / generating command / "|" leader.
	ModeOff PrependMode = "off"
)

// DefaultMode is the prepend mode used when neither config nor flag specifies
// one. ModePipeOnly preserves historical splunk-cli behavior so existing
// workflows keep working without change.
const DefaultMode = ModePipeOnly

// Wrap returns the SPL string ready for submission, applying "search "
// prepend according to mode. The original input is preserved verbatim
// when no prefix is added (whitespace and all).
func Wrap(spl string, mode PrependMode) string {
	if mode == "" {
		mode = DefaultMode
	}
	if mode == ModeOff {
		return spl
	}
	trimmed := strings.TrimSpace(spl)
	if strings.HasPrefix(trimmed, "|") {
		return spl
	}
	if mode == ModeAuto && hasLeadingSearchCommand(trimmed) {
		return spl
	}
	return "search " + spl
}

// hasLeadingSearchCommand reports whether s begins with the SPL command
// "search" followed by whitespace or end-of-string. Tokens that merely
// start with "search" (for example searchengine=foo) return false.
func hasLeadingSearchCommand(s string) bool {
	const kw = "search"
	if !strings.HasPrefix(s, kw) {
		return false
	}
	if len(s) == len(kw) {
		return true
	}
	switch s[len(kw)] {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// ParseMode normalizes user-supplied input (CLI flag value or TOML field)
// into a PrependMode. Empty string maps to DefaultMode. Unknown values
// return an error so misconfiguration fails loudly rather than silently
// falling back.
func ParseMode(s string) (PrependMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return DefaultMode, nil
	case string(ModeAuto):
		return ModeAuto, nil
	case string(ModePipeOnly):
		return ModePipeOnly, nil
	case string(ModeOff):
		return ModeOff, nil
	}
	return "", fmt.Errorf("invalid prepend mode %q (want auto | pipe-only | off)", s)
}
