// Package client provides a Splunk REST API client.
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/nlink-jp/splunk-cli/internal/config"
)

const (
	defaultHTTPTimeout  = 30 * time.Second
	defaultOwner        = "nobody"
	pollInterval        = 2 * time.Second
	maxResultsPerPage   = 50_000
)

// Client is a Splunk REST API client.
type Client struct {
	http   *http.Client
	cfg    *config.Config
	stderr io.Writer
	silent bool
}

// New creates a new Client. If cfg.HTTPTimeout is zero, defaultHTTPTimeout is used.
// If cfg.Owner is empty, defaultOwner is used.
func New(cfg *config.Config, silent bool) (*Client, error) {
	if strings.HasPrefix(cfg.Host, "http://") && cfg.Token != "" {
		fmt.Fprintf(os.Stderr,
			"Warning: sending Splunk token over unencrypted HTTP to %s.\n"+
				"  Use an https:// endpoint to protect your credentials.\n",
			cfg.Host,
		)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("client: create cookie jar: %w", err)
	}

	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = defaultHTTPTimeout
	}
	if cfg.Owner == "" {
		cfg.Owner = defaultOwner
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.Insecure} //nolint:gosec

	return &Client{
		http: &http.Client{
			Transport: transport,
			Timeout:   timeout,
			Jar:       jar,
		},
		cfg:    cfg,
		stderr: os.Stderr,
		silent: silent && !cfg.Debug,
	}, nil
}

// Logf writes to stderr unless silent.
func (c *Client) Logf(format string, a ...any) {
	if !c.silent {
		fmt.Fprintf(c.stderr, format, a...)
	}
}

// debugf writes to stderr only when debug is enabled.
func (c *Client) debugf(format string, a ...any) {
	if c.cfg.Debug {
		fmt.Fprintf(c.stderr, "DEBUG: "+format, a...)
	}
}

func (c *Client) apiURL(segments ...string) (string, error) {
	base, err := url.Parse(c.cfg.Host)
	if err != nil {
		return "", fmt.Errorf("invalid host URL: %w", err)
	}
	var parts []string
	if c.cfg.App != "" {
		parts = append([]string{"servicesNS", c.cfg.Owner, c.cfg.App}, segments...)
	} else {
		parts = append([]string{"services"}, segments...)
	}
	return base.JoinPath(parts...).String(), nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	} else if c.cfg.User != "" && c.cfg.Password != "" {
		req.SetBasicAuth(c.cfg.User, c.cfg.Password)
	}

	if c.cfg.Debug {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			s := string(dump)
			if c.cfg.Token != "" {
				s = strings.Replace(s, c.cfg.Token, "<TOKEN>", 1)
			}
			c.debugf("\n--- REQUEST ---\n%s\n--- END ---\n", s)
		}
	}

	return c.http.Do(req)
}

func checkStatus(resp *http.Response, want int) error {
	if resp.StatusCode == want {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API error %s: %s", resp.Status, strings.TrimSpace(string(body)))
}

// SplunkMessage is a message returned by the Splunk API in a job status response.
type SplunkMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// JobStatus holds the result of a job status check.
type JobStatus struct {
	SID           string
	IsDone        bool
	DispatchState string
	Messages      []SplunkMessage
	ResultCount   int
}

// StartSearch initiates an asynchronous Splunk search and returns the SID.
func (c *Client) StartSearch(ctx context.Context, spl, earliest, latest string) (string, error) {
	endpoint, err := c.apiURL("search", "jobs")
	if err != nil {
		return "", err
	}
	c.debugf("POST %s\n", endpoint)

	form := url.Values{}
	if !strings.HasPrefix(strings.TrimSpace(spl), "|") {
		form.Set("search", "search "+spl)
	} else {
		form.Set("search", spl)
	}
	if earliest != "" {
		form.Set("earliest_time", earliest)
	}
	if latest != "" {
		form.Set("latest_time", latest)
	}
	form.Set("output_mode", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, http.StatusCreated); err != nil {
		return "", err
	}

	var job struct {
		SID string `json:"sid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return "", fmt.Errorf("decode start-search response: %w", err)
	}
	return job.SID, nil
}

// GetJobStatus returns the current status of a search job.
func (c *Client) GetJobStatus(ctx context.Context, sid string) (JobStatus, error) {
	endpoint, err := c.apiURL("search", "jobs", sid)
	if err != nil {
		return JobStatus{}, err
	}
	c.debugf("GET %s\n", endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return JobStatus{}, err
	}
	q := req.URL.Query()
	q.Set("output_mode", "json")
	req.URL.RawQuery = q.Encode()

	resp, err := c.do(req)
	if err != nil {
		return JobStatus{}, err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, http.StatusOK); err != nil {
		return JobStatus{}, err
	}

	var raw struct {
		Entry []struct {
			Content struct {
				IsDone        bool            `json:"isDone"`
				DispatchState string          `json:"dispatchState"`
				Messages      []SplunkMessage `json:"messages"`
				ResultCount   int             `json:"resultCount"`
			} `json:"content"`
		} `json:"entry"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return JobStatus{}, fmt.Errorf("read job status response: %w", err)
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return JobStatus{}, fmt.Errorf("decode job status: %w (body: %s)", err, body)
	}
	if len(raw.Entry) == 0 {
		return JobStatus{}, errors.New("job not found in status response")
	}
	content := raw.Entry[0].Content
	return JobStatus{
		SID:           sid,
		IsDone:        content.IsDone,
		DispatchState: content.DispatchState,
		Messages:      content.Messages,
		ResultCount:   content.ResultCount,
	}, nil
}

// WaitForJob polls until the job is done or ctx is cancelled.
func (c *Client) WaitForJob(ctx context.Context, sid string) error {
	c.Logf("Waiting for job to complete...\n")
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := c.GetJobStatus(ctx, sid)
			if err != nil {
				return err
			}
			if !status.IsDone {
				continue
			}
			if status.DispatchState == "FAILED" {
				var msgs strings.Builder
				for _, m := range status.Messages {
					if strings.EqualFold(m.Type, "FATAL") || strings.EqualFold(m.Type, "ERROR") {
						fmt.Fprintf(&msgs, "\n  - %s", m.Text)
					}
				}
				if msgs.Len() > 0 {
					return fmt.Errorf("job %s failed:%s", sid, msgs.String())
				}
				return fmt.Errorf("job %s failed", sid)
			}
			c.Logf("Job complete.\n")
			return nil
		}
	}
}

// Results fetches all results of a completed job, handling pagination.
// totalResults is the result count from a prior GetJobStatus call; pass it to
// avoid a redundant status fetch. If limit is 0, all results are returned.
func (c *Client) Results(ctx context.Context, sid string, limit, totalResults int) (string, error) {
	fetchCount := limit
	if limit == 0 || limit > totalResults {
		fetchCount = totalResults
	}

	all := make([]json.RawMessage, 0, fetchCount)
	for offset := 0; offset < fetchCount; offset += maxResultsPerPage {
		count := maxResultsPerPage
		if offset+count > fetchCount {
			count = fetchCount - offset
		}

		page, err := c.fetchResultsPage(ctx, sid, offset, count)
		if err != nil {
			return "", err
		}
		all = append(all, page...)
	}

	out, err := json.MarshalIndent(map[string]any{"results": all}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal results: %w", err)
	}
	return string(out), nil
}

// fetchResultsPage fetches one page of results. The response body is closed
// before returning so callers do not need to manage it.
func (c *Client) fetchResultsPage(ctx context.Context, sid string, offset, count int) ([]json.RawMessage, error) {
	endpoint, err := c.apiURL("search", "jobs", sid, "results")
	if err != nil {
		return nil, err
	}
	c.debugf("GET %s (offset=%d count=%d)\n", endpoint, offset, count)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("output_mode", "json")
	q.Set("offset", fmt.Sprintf("%d", offset))
	q.Set("count", fmt.Sprintf("%d", count))
	req.URL.RawQuery = q.Encode()

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var page struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("decode results page: %w", err)
	}
	return page.Results, nil
}

// CancelSearch cancels a running job.
func (c *Client) CancelSearch(ctx context.Context, sid string) error {
	c.Logf("\nCancelling job %s...\n", sid)
	endpoint, err := c.apiURL("search", "jobs", sid, "control")
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader("action=cancel"))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cancel job %s: %s: %s", sid, resp.Status, body)
	}
	c.Logf("Job cancelled.\n")
	return nil
}
