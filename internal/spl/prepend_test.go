package spl

import "testing"

func TestWrap(t *testing.T) {
	cases := []struct {
		name string
		mode PrependMode
		in   string
		want string
	}{
		// --- ModePipeOnly (historical default) --------------------------
		{"pipe-only/bare term", ModePipeOnly, "index=foo", "search index=foo"},
		{"pipe-only/leading search", ModePipeOnly, "search index=foo", "search search index=foo"}, // historical double-search
		{"pipe-only/pipe leader", ModePipeOnly, "| stats count", "| stats count"},
		{"pipe-only/leading whitespace before pipe", ModePipeOnly, "  | stats count", "  | stats count"},
		{"pipe-only/quoted phrase", ModePipeOnly, `"foo bar"`, `search "foo bar"`},
		{"pipe-only/backtick macro", ModePipeOnly, "`mymacro`", "search `mymacro`"},

		// --- ModeAuto --------------------------------------------------
		{"auto/bare term", ModeAuto, "index=foo", "search index=foo"},
		{"auto/leading search command", ModeAuto, "search index=foo", "search index=foo"},
		{"auto/lone search keyword", ModeAuto, "search", "search"},
		{"auto/leading search with tab", ModeAuto, "search\tindex=foo", "search\tindex=foo"},
		{"auto/searchengine token (not the command)", ModeAuto, "searchengine=foo", "search searchengine=foo"},
		{"auto/pipe leader", ModeAuto, "| tstats count BY host", "| tstats count BY host"},
		{"auto/leading whitespace + search", ModeAuto, "  search index=foo", "  search index=foo"},
		{"auto/quoted phrase", ModeAuto, `"foo bar" earliest=-1h`, `search "foo bar" earliest=-1h`},
		{"auto/backtick macro", ModeAuto, "`mymacro`", "search `mymacro`"},

		// --- empty mode falls back to DefaultMode (pipe-only) ----------
		{"empty/bare term", "", "index=foo", "search index=foo"},
		{"empty/pipe leader", "", "| stats count", "| stats count"},

		// --- ModeOff --------------------------------------------------
		{"off/bare term passes through", ModeOff, "index=foo", "index=foo"},
		{"off/leading search passes through", ModeOff, "search index=foo", "search index=foo"},
		{"off/pipe leader passes through", ModeOff, "| stats count", "| stats count"},
		{"off/empty passes through", ModeOff, "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Wrap(tc.in, tc.mode)
			if got != tc.want {
				t.Errorf("Wrap(%q, %q) = %q; want %q", tc.in, tc.mode, got, tc.want)
			}
		})
	}
}

func TestParseMode(t *testing.T) {
	ok := []struct {
		in   string
		want PrependMode
	}{
		{"", DefaultMode},
		{"auto", ModeAuto},
		{"AUTO", ModeAuto},
		{"  pipe-only  ", ModePipeOnly},
		{"off", ModeOff},
	}
	for _, tc := range ok {
		got, err := ParseMode(tc.in)
		if err != nil {
			t.Errorf("ParseMode(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseMode(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}

	bad := []string{"smart", "raw", "yes", "no", "true"}
	for _, in := range bad {
		if _, err := ParseMode(in); err == nil {
			t.Errorf("ParseMode(%q) expected error, got nil", in)
		}
	}
}

// Regression: Issue #4 — pasting `search index=foo` should NOT produce
// `search search index=foo` when the user opted into ModeAuto.
func TestWrap_Issue4_AutoPreventsDoubleSearch(t *testing.T) {
	got := Wrap("search index=foo", ModeAuto)
	if got != "search index=foo" {
		t.Errorf("ModeAuto must not double-prepend; got %q", got)
	}
}
