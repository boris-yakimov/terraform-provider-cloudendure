// Package client is a minimal HTTP client for the CloudEndure REST API.
//
// It handles session authentication (login + cookie/XSRF handling) and
// exposes typed helpers for the endpoints used by the Terraform provider.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultBaseURL is the public CloudEndure API endpoint.
	DefaultBaseURL = "https://console.cloudendure.com/api/latest"

	// sessionTTL is how long a login session is reused before re-authenticating.
	// CloudEndure sessions are valid for about an hour.
	sessionTTL = 50 * time.Minute

	xsrfCookieName = "XSRF-TOKEN"
	xsrfHeaderName = "X-XSRF-TOKEN"

	// listPageSize is the page size used when listing collections.
	listPageSize = 1500

	maxErrorBodyBytes = 4096
)

// Config configures a Client.
type Config struct {
	// BaseURL of the CloudEndure API. Defaults to DefaultBaseURL.
	BaseURL string
	// APIToken is the CloudEndure user API token used to log in.
	APIToken string
	// HTTPClient is the underlying HTTP client. Defaults to a client with a 30s timeout.
	HTTPClient *http.Client
	// UserAgent sent with every request.
	UserAgent string
}

// Client is a CloudEndure API client. It is safe for concurrent use.
type Client struct {
	baseURL    *url.URL
	apiToken   string
	httpClient *http.Client
	userAgent  string

	mu         sync.Mutex
	cookies    map[string]string
	xsrfToken  string
	loggedInAt time.Time
	now        func() time.Time
}

// New creates a new Client. It does not contact the API; authentication
// happens lazily on the first request (or explicitly via Login).
func New(cfg Config) (*Client, error) {
	if cfg.APIToken == "" {
		return nil, errors.New("cloudendure: API token must not be empty")
	}

	rawURL := cfg.BaseURL
	if rawURL == "" {
		rawURL = DefaultBaseURL
	}
	u, err := url.Parse(strings.TrimRight(rawURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("cloudendure: invalid base URL %q: %w", rawURL, err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("cloudendure: base URL %q must use http or https", rawURL)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "terraform-provider-cloudendure"
	}

	return &Client{
		baseURL:    u,
		apiToken:   cfg.APIToken,
		httpClient: httpClient,
		userAgent:  userAgent,
		cookies:    map[string]string{},
		now:        time.Now,
	}, nil
}

// APIError is returned when the API responds with a non-2xx status code.
type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("cloudendure: %s %s returned HTTP %d", e.Method, e.Path, e.StatusCode)
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}

// IsNotFound reports whether err is an APIError with HTTP status 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// Login authenticates against the API and stores the session cookies and
// XSRF token for subsequent requests.
func (c *Client) Login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loginLocked(ctx)
}

func (c *Client) loginLocked(ctx context.Context) error {
	body, err := json.Marshal(map[string]string{"userApiToken": c.apiToken})
	if err != nil {
		return err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/login", bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloudendure: login request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return newAPIError(req, resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)

	cookies := parseSetCookies(resp.Header.Values("Set-Cookie"))
	xsrf, ok := cookies[xsrfCookieName]
	if !ok || xsrf == "" {
		return errors.New("cloudendure: login succeeded but no XSRF-TOKEN cookie was returned")
	}

	c.cookies = cookies
	c.xsrfToken = strings.Trim(xsrf, `"`)
	c.loggedInAt = c.now()
	return nil
}

// ensureSession logs in when there is no session or it is about to expire.
func (c *Client) ensureSession(ctx context.Context, force bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !force && c.xsrfToken != "" && c.now().Sub(c.loggedInAt) < sessionTTL {
		return nil
	}
	return c.loginLocked(ctx)
}

// do performs an authenticated JSON request. in (if non-nil) is marshalled as
// the request body; out (if non-nil) receives the decoded response body.
// On HTTP 401 the client re-authenticates once and retries.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, in, out any) error {
	var payload []byte
	if in != nil {
		var err error
		if payload, err = json.Marshal(in); err != nil {
			return fmt.Errorf("cloudendure: encoding request body: %w", err)
		}
	}

	for attempt := 0; ; attempt++ {
		if err := c.ensureSession(ctx, attempt > 0); err != nil {
			return err
		}

		fullPath := path
		if len(query) > 0 {
			fullPath += "?" + query.Encode()
		}

		var body io.Reader
		if payload != nil {
			body = bytes.NewReader(payload)
		}
		req, err := c.newRequest(ctx, method, fullPath, body)
		if err != nil {
			return err
		}
		c.authorize(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("cloudendure: %s %s: %w", method, path, err)
		}

		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			continue
		}

		err = decodeResponse(req, resp, out)
		_ = resp.Body.Close()
		return err
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL.String()+path, body)
	if err != nil {
		return nil, fmt.Errorf("cloudendure: building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// authorize attaches the session cookies and XSRF header to req.
//
// Cookies are set manually rather than through net/http/cookiejar because
// CloudEndure's XSRF-TOKEN cookie value contains a backslash escape
// (e.g. "abc==\012"), which Go's cookie parser rejects.
func (c *Client) authorize(req *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pairs := make([]string, 0, len(c.cookies))
	for name, value := range c.cookies {
		pairs = append(pairs, name+"="+value)
	}
	if len(pairs) > 0 {
		req.Header.Set("Cookie", strings.Join(pairs, "; "))
	}
	req.Header.Set(xsrfHeaderName, c.xsrfToken)
}

func decodeResponse(req *http.Request, resp *http.Response, out any) error {
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return newAPIError(req, resp)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("cloudendure: decoding %s %s response: %w", req.Method, req.URL.Path, err)
	}
	return nil
}

func newAPIError(req *http.Request, resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	return &APIError{
		Method:     req.Method,
		Path:       req.URL.Path,
		StatusCode: resp.StatusCode,
		Body:       strings.TrimSpace(string(b)),
	}
}

// parseSetCookies extracts name/value pairs from raw Set-Cookie header values,
// keeping the values exactly as sent by the server.
func parseSetCookies(headers []string) map[string]string {
	cookies := make(map[string]string, len(headers))
	for _, h := range headers {
		pair, _, _ := strings.Cut(h, ";")
		name, value, ok := strings.Cut(pair, "=")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			continue
		}
		cookies[name] = strings.TrimSpace(value)
	}
	return cookies
}

func projectPath(projectID string, parts ...string) string {
	var b strings.Builder
	b.WriteString("/projects/")
	b.WriteString(url.PathEscape(projectID))
	for _, p := range parts {
		b.WriteByte('/')
		b.WriteString(url.PathEscape(p))
	}
	return b.String()
}

// listAll pages through a CloudEndure collection endpoint.
func listAll[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	var all []T
	for offset := 0; ; offset += listPageSize {
		q := url.Values{}
		q.Set("offset", fmt.Sprint(offset))
		q.Set("limit", fmt.Sprint(listPageSize))

		var page struct {
			Items []T `json:"items"`
		}
		if err := c.do(ctx, http.MethodGet, path, q, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		if len(page.Items) < listPageSize {
			return all, nil
		}
	}
}
