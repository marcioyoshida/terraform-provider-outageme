package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to the outage.me Monitor REST API (/v1/monitor, https://outage.me/openapi.json).
type Client struct {
	BaseURL string
	APIKey  string
	Org     string
	HTTP    *http.Client
	UA      string
}

// APIError carries the HTTP status and the API's own error message.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("outage.me API %d: %s", e.Status, e.Message)
}

// IsNotFound reports a 404, which Read treats as "deleted outside Terraform".
func IsNotFound(err error) bool {
	if e, ok := err.(*APIError); ok {
		return e.Status == 404
	}
	return false
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	u := strings.TrimRight(c.BaseURL, "/") + path
	if c.Org != "" {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u += sep + "org=" + url.QueryEscape(c.Org)
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("x-tarantula-api-key", c.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UA)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	var resp *http.Response
	for attempt := 0; ; attempt++ {
		resp, err = hc.Do(req)
		if err != nil {
			return err
		}
		// the edge rate limit (WAF) answers 429: back off and retry a few times
		if resp.StatusCode != 429 || attempt >= 3 {
			break
		}
		resp.Body.Close()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(5*(attempt+1)) * time.Second):
		}
		if rdr != nil {
			b, _ := json.Marshal(body)
			req.Body = io.NopCloser(bytes.NewReader(b))
		}
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &e)
		if e.Error == "" {
			e.Error = strings.TrimSpace(string(data))
		}
		return &APIError{Status: resp.StatusCode, Message: e.Error}
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
