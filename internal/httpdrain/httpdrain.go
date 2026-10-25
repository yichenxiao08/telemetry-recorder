// Package httpdrain rejects requests without resetting the client connection.
//
// Responding before the request body is read can make the server close a
// connection that still has unread bytes. On Windows that close is sent as a
// TCP reset, so a client still writing its body sees WinError 10054 instead of
// the status the server wrote.
package httpdrain

import (
	"io"
	"net/http"
)

// Limit bounds how much unread body is discarded before responding. Larger
// bodies get Connection: close, which is the honest signal for abandoning them.
const Limit = 1 << 20

// Error discards the request body, up to Limit, then writes http.Error.
func Error(w http.ResponseWriter, r *http.Request, message string, status int) {
	if r.Body != nil {
		if _, err := io.CopyN(io.Discard, r.Body, Limit+1); err == nil {
			w.Header().Set("Connection", "close")
		}
	}
	http.Error(w, message, status)
}
