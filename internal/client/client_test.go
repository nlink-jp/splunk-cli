package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlink-jp/splunk-cli/internal/config"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)

	cfg := &config.Config{
		Host:     srv.URL,
		Token:    "testtoken",
		Insecure: true,
	}
	c, err := New(cfg, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Use the test server's TLS client to handle self-signed cert.
	c.http = srv.Client()
	// Re-add token setup by wrapping the transport.
	return c
}

func TestStartSearch(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "search/jobs") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"sid": "test_sid_123"})
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := &config.Config{Host: srv.URL, Token: "tok"}
	c, err := New(cfg, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	sid, err := c.StartSearch(context.Background(), "index=main", "-1h", "now")
	if err != nil {
		t.Fatalf("StartSearch: %v", err)
	}
	if sid != "test_sid_123" {
		t.Errorf("SID = %q, want %q", sid, "test_sid_123")
	}
}

func TestStartSearch_PipePrefix(t *testing.T) {
	var gotSearch string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotSearch = r.FormValue("search")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"sid": "sid1"})
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := &config.Config{Host: srv.URL, Token: "tok"}
	c, _ := New(cfg, true)

	// SPL starting with | should NOT get "search " prefix.
	c.StartSearch(context.Background(), "| stats count", "", "")
	if gotSearch != "| stats count" {
		t.Errorf("pipe SPL should not get prefix, got: %q", gotSearch)
	}

	// Normal SPL should get "search " prefix.
	c.StartSearch(context.Background(), "index=main", "", "")
	if gotSearch != "search index=main" {
		t.Errorf("normal SPL should get prefix, got: %q", gotSearch)
	}
}

func TestGetJobStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"entry": []map[string]any{
				{"content": map[string]any{
					"isDone":        true,
					"dispatchState": "DONE",
					"resultCount":   42,
					"messages":      []any{},
				}},
			},
		})
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := &config.Config{Host: srv.URL, Token: "tok"}
	c, _ := New(cfg, true)

	status, err := c.GetJobStatus(context.Background(), "sid1")
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}
	if !status.IsDone {
		t.Error("IsDone should be true")
	}
	if status.DispatchState != "DONE" {
		t.Errorf("DispatchState = %q", status.DispatchState)
	}
	if status.ResultCount != 42 {
		t.Errorf("ResultCount = %d", status.ResultCount)
	}
}

func TestGetJobStatus_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"entry": []any{}})
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := &config.Config{Host: srv.URL, Token: "tok"}
	c, _ := New(cfg, true)

	_, err := c.GetJobStatus(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for empty entry")
	}
}

func TestGetJobStatus_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := &config.Config{Host: srv.URL, Token: "tok"}
	c, _ := New(cfg, true)

	_, err := c.GetJobStatus(context.Background(), "sid1")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestCancelSearch(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "control") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		r.ParseForm()
		if r.FormValue("action") != "cancel" {
			t.Errorf("expected action=cancel, got %q", r.FormValue("action"))
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := &config.Config{Host: srv.URL, Token: "tok"}
	c, _ := New(cfg, true)

	if err := c.CancelSearch(context.Background(), "sid1"); err != nil {
		t.Fatalf("CancelSearch: %v", err)
	}
}

func TestHTTPWarning(t *testing.T) {
	// New should not fail for HTTP URLs — it just warns.
	cfg := &config.Config{
		Host:  "http://splunk.example.com:8089",
		Token: "secret",
	}
	c, err := New(cfg, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c == nil {
		t.Error("expected non-nil client")
	}
}
