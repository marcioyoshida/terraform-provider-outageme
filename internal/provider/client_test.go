package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientSendsKeyAndOrgAndParsesErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-tarantula-api-key") != "k" || r.URL.Query().Get("org") != "o1" {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/v1/monitor/checks/missing" {
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"No such check."}`))
			return
		}
		w.Write([]byte(`{"check":{"check_id":"abc"}}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, APIKey: "k", Org: "o1"}
	var out checkEnvelope
	if err := c.do(context.Background(), "GET", "/v1/monitor/checks/abc", nil, &out); err != nil || out.Check["check_id"] != "abc" {
		t.Fatalf("got %v %v", out, err)
	}
	err := c.do(context.Background(), "GET", "/v1/monitor/checks/missing", nil, nil)
	if !IsNotFound(err) || err.Error() != "outage.me API 404: No such check." {
		t.Fatalf("want a 404 with the API's message, got %v", err)
	}
}

func TestClientRetriesTheEdgeRateLimit(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.WriteHeader(429)
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, APIKey: "k"}
	if err := c.do(context.Background(), "POST", "/v1/monitor/windows", map[string]any{"a": 1}, nil); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want one retry, got %d calls", n)
	}
}
