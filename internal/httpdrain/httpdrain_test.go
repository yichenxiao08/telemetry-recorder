package httpdrain

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type countingBody struct {
	io.Reader
	read int
}

func (c *countingBody) Read(p []byte) (int, error) {
	n, err := c.Reader.Read(p)
	c.read += n
	return n, err
}
func (c *countingBody) Close() error { return nil }

func TestErrorDrainsSmallBody(t *testing.T) {
	body := &countingBody{Reader: bytes.NewReader(make([]byte, 4096))}
	r := httptest.NewRequest("POST", "/", body)
	r.Body = body
	w := httptest.NewRecorder()
	Error(w, r, "nope", http.StatusForbidden)
	if body.read != 4096 {
		t.Fatalf("drained %d bytes, want 4096", body.read)
	}
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "nope") {
		t.Fatalf("response %d %q", w.Code, w.Body.String())
	}
	if w.Header().Get("Connection") != "" {
		t.Fatal("fully drained body should keep the connection reusable")
	}
}

func TestErrorClosesAfterLimit(t *testing.T) {
	body := &countingBody{Reader: io.LimitReader(zeros{}, Limit*2)}
	r := httptest.NewRequest("POST", "/", body)
	r.Body = body
	w := httptest.NewRecorder()
	Error(w, r, "nope", http.StatusForbidden)
	if body.read != Limit+1 {
		t.Fatalf("drained %d bytes, want %d", body.read, Limit+1)
	}
	if w.Header().Get("Connection") != "close" {
		t.Fatal("oversized body must close the connection")
	}
}

func TestErrorNilBody(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Body = nil
	w := httptest.NewRecorder()
	Error(w, r, "x", http.StatusBadRequest)
	if w.Code != http.StatusBadRequest {
		t.Fatal(w.Code)
	}
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) { clear(p); return len(p), nil }

// A real server rejecting before reading must not reset a client that is
// still uploading. This is the regression for the strict-P0 403 flake.
func TestRealServerRejectionDoesNotReset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, "denied", http.StatusForbidden)
	}))
	defer srv.Close()
	payload := bytes.Repeat([]byte("x"), 512<<10)
	for i := 0; i < 200; i++ {
		resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status %d", resp.StatusCode)
		}
	}
}
