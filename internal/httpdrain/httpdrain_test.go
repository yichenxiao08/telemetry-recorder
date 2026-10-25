package httpdrain

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (w *deadlineRecorder) SetReadDeadline(d time.Time) error {
	w.deadlines = append(w.deadlines, d)
	return nil
}

func TestErrorDrainsSmallBody(t *testing.T) {
	body := &countingBody{Reader: bytes.NewReader(make([]byte, 4096))}
	r := httptest.NewRequest("POST", "/", body)
	r.Body = body
	w := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
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
	if len(w.deadlines) != 2 || w.deadlines[0].IsZero() || !w.deadlines[1].IsZero() {
		t.Fatalf("deadline was not set and cleared: %v", w.deadlines)
	}
}

func TestErrorClosesAfterLimit(t *testing.T) {
	body := &countingBody{Reader: io.LimitReader(zeros{}, Limit*2)}
	r := httptest.NewRequest("POST", "/", body)
	r.Body = body
	w := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	Error(w, r, "nope", http.StatusForbidden)
	if body.read != Limit+1 {
		t.Fatalf("drained %d bytes, want %d", body.read, Limit+1)
	}
	if w.Header().Get("Connection") != "close" {
		t.Fatal("oversized body must close the connection")
	}
}

func TestErrorDoesNotDrainWithoutDeadlineSupport(t *testing.T) {
	body := &countingBody{Reader: bytes.NewReader(make([]byte, 4096))}
	r := httptest.NewRequest("POST", "/", body)
	r.Body = body
	w := httptest.NewRecorder()
	Error(w, r, "denied", http.StatusForbidden)
	if body.read != 0 || w.Header().Get("Connection") != "close" || w.Code != http.StatusForbidden {
		t.Fatalf("read=%d headers=%v status=%d", body.read, w.Header(), w.Code)
	}
}

func TestErrorDoesNotDrainCanceledRequest(t *testing.T) {
	body := &countingBody{Reader: bytes.NewReader(make([]byte, 4096))}
	r := httptest.NewRequest("POST", "/", body)
	ctx, cancel := context.WithCancel(r.Context())
	cancel()
	r = r.WithContext(ctx)
	r.Body = body
	w := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	Error(w, r, "denied", http.StatusForbidden)
	if body.read != 0 || w.Header().Get("Connection") != "close" {
		t.Fatalf("read=%d headers=%v", body.read, w.Header())
	}
}

func TestRealServerStalledBodyIsBounded(t *testing.T) {
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, "denied", http.StatusForbidden)
		close(done)
	}))
	defer srv.Close()
	conn, err := net.DialTimeout("tcp", srv.Listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err = conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err = fmt.Fprint(conn, "POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 4096\r\n\r\nx"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden || !resp.Close {
		t.Fatalf("status=%d close=%t", resp.StatusCode, resp.Close)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not return after stalled body")
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
