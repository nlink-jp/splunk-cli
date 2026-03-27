//go:build integration

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nlink-jp/splunk-cli/internal/config"
)

// integrationClient builds a Client from SPLUNK_HOST / SPLUNK_TOKEN env vars.
// The test is skipped if either variable is unset.
func integrationClient(t *testing.T) *Client {
	t.Helper()
	host := os.Getenv("SPLUNK_HOST")
	token := os.Getenv("SPLUNK_TOKEN")
	if host == "" || token == "" {
		t.Skip("SPLUNK_HOST and SPLUNK_TOKEN must be set for integration tests")
	}
	cfg := &config.Config{
		Host:     host,
		Token:    token,
		Insecure: true, // container uses a self-signed cert
	}
	c, err := New(cfg, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// TestIntegration_SearchAndResults runs a full search lifecycle:
//
//	StartSearch → WaitForJob → GetJobStatus → Results
func TestIntegration_SearchAndResults(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// A simple generating SPL that always produces exactly 3 rows.
	spl := `| makeresults count=3 | eval msg="hello"`

	sid, err := c.StartSearch(ctx, spl, "", "")
	if err != nil {
		t.Fatalf("StartSearch: %v", err)
	}
	t.Logf("SID: %s", sid)

	if err := c.WaitForJob(ctx, sid); err != nil {
		t.Fatalf("WaitForJob: %v", err)
	}

	status, err := c.GetJobStatus(ctx, sid)
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}
	if !status.IsDone {
		t.Errorf("IsDone = false, want true")
	}
	if status.DispatchState != "DONE" {
		t.Errorf("DispatchState = %q, want DONE", status.DispatchState)
	}
	if status.ResultCount != 3 {
		t.Errorf("ResultCount = %d, want 3", status.ResultCount)
	}

	results, err := c.Results(ctx, sid, 0, status.ResultCount)
	if err != nil {
		t.Fatalf("Results: %v", err)
	}

	// Validate JSON shape: {"results": [...]}
	var parsed struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal([]byte(results), &parsed); err != nil {
		t.Fatalf("Results JSON invalid: %v\nbody: %s", err, results)
	}
	if len(parsed.Results) != 3 {
		t.Errorf("len(results) = %d, want 3", len(parsed.Results))
	}
	t.Logf("Results: %s", results)
}

// TestIntegration_Limit checks that the limit parameter is respected.
func TestIntegration_Limit(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	spl := `| makeresults count=10 | eval n=_serial`
	sid, err := c.StartSearch(ctx, spl, "", "")
	if err != nil {
		t.Fatalf("StartSearch: %v", err)
	}
	if err := c.WaitForJob(ctx, sid); err != nil {
		t.Fatalf("WaitForJob: %v", err)
	}
	status, err := c.GetJobStatus(ctx, sid)
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}

	results, err := c.Results(ctx, sid, 3, status.ResultCount)
	if err != nil {
		t.Fatalf("Results(limit=3): %v", err)
	}

	var parsed struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal([]byte(results), &parsed); err != nil {
		t.Fatalf("Results JSON invalid: %v", err)
	}
	if len(parsed.Results) != 3 {
		t.Errorf("len(results) = %d, want 3 (limit respected)", len(parsed.Results))
	}
}

// TestIntegration_EmptyResults checks that a zero-result job returns [].
func TestIntegration_EmptyResults(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	spl := `| makeresults count=0`
	sid, err := c.StartSearch(ctx, spl, "", "")
	if err != nil {
		t.Fatalf("StartSearch: %v", err)
	}
	if err := c.WaitForJob(ctx, sid); err != nil {
		t.Fatalf("WaitForJob: %v", err)
	}
	status, err := c.GetJobStatus(ctx, sid)
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}

	results, err := c.Results(ctx, sid, 0, status.ResultCount)
	if err != nil {
		t.Fatalf("Results: %v", err)
	}

	if !strings.Contains(results, `"results": []`) {
		t.Errorf("expected empty array, got: %s", results)
	}
}

// TestIntegration_CancelSearch starts a long-running job and cancels it.
func TestIntegration_CancelSearch(t *testing.T) {
	c := integrationClient(t)
	ctx := context.Background()

	// A search over all time that won't finish quickly.
	spl := `search index=* | head 1`
	sid, err := c.StartSearch(ctx, spl, "0", "now")
	if err != nil {
		t.Fatalf("StartSearch: %v", err)
	}
	t.Logf("Started job %s", sid)

	if err := c.CancelSearch(ctx, sid); err != nil {
		t.Fatalf("CancelSearch: %v", err)
	}

	// After cancel, job should not be in a running state.
	status, err := c.GetJobStatus(ctx, sid)
	if err != nil {
		// 404 after cancel is also acceptable behaviour.
		t.Logf("GetJobStatus after cancel: %v (may be expected)", err)
		return
	}
	if status.DispatchState == "RUNNING" {
		t.Errorf("job still RUNNING after cancel")
	}
	t.Logf("Post-cancel state: %s", status.DispatchState)
}

// TestIntegration_InvalidSPL checks that a syntax error returns an error.
func TestIntegration_InvalidSPL(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	spl := `| thisisnotarealcommand`
	sid, err := c.StartSearch(ctx, spl, "", "")
	if err != nil {
		// Some Splunk versions reject bad SPL at submit time — that's fine.
		t.Logf("StartSearch rejected invalid SPL at submit: %v", err)
		return
	}

	err = c.WaitForJob(ctx, sid)
	if err == nil {
		t.Error("expected WaitForJob to fail for invalid SPL, got nil")
		return
	}
	if !strings.Contains(err.Error(), "failed") && !strings.Contains(err.Error(), "FATAL") {
		t.Errorf("unexpected error: %v", err)
	}
	t.Logf("WaitForJob correctly returned error: %v", err)
}

// TestIntegration_SearchPrefix verifies that bare SPL gets "search " prepended.
func TestIntegration_SearchPrefix(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// This SPL is valid only if the "search " prefix is added.
	spl := `index=_internal | head 1`
	sid, err := c.StartSearch(ctx, spl, "-1m", "now")
	if err != nil {
		t.Fatalf("StartSearch: %v", err)
	}
	if err := c.WaitForJob(ctx, sid); err != nil {
		t.Fatalf("WaitForJob: %v", err)
	}
	status, err := c.GetJobStatus(ctx, sid)
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}
	if status.DispatchState == "FAILED" {
		t.Errorf("search with bare SPL failed — prefix may not have been added")
	}
	t.Logf("DispatchState=%s ResultCount=%d", status.DispatchState, status.ResultCount)
}

// Compile-time check: fmt is used via t.Logf / Sprintf.
var _ = fmt.Sprintf
