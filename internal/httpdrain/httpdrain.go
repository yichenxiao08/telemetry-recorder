// Package httpdrain bounds best-effort draining before rejecting requests.
//
// Responding before the request body is read can make the server close a
// connection that still has unread bytes. On Windows that close is sent as a
// TCP reset, so a client still writing its body sees WinError 10054 instead of
// the status the server wrote.
package httpdrain

import (
	"io"
	"net/http"
	"time"
)

// Limit bounds how much unread body is discarded before responding. Larger
// bodies get Connection: close, which is the honest signal for abandoning them.
const Limit = 1 << 20

// Timeout bounds network body reads, independently of the serving timeout.
const Timeout = time.Second

// Error discards at most Limit+1 unread bytes within Timeout, then writes
// http.Error. Only a fully drained body can reuse the connection. Oversized,
// broken, canceled or stalled uploads may still observe a transport error.
// Writers without read-deadline support are rejected without draining.
func Error(w http.ResponseWriter, r *http.Request, message string, status int) {
	if r.Body != nil && r.Body != http.NoBody {
		controller := http.NewResponseController(w)
		if r.Context().Err() != nil || controller.SetReadDeadline(time.Now().Add(Timeout)) != nil {
			w.Header().Set("Connection", "close")
		} else {
			_, err := io.CopyN(io.Discard, r.Body, Limit+1)
			if err != io.EOF {
				w.Header().Set("Connection", "close")
			}
			if controller.SetReadDeadline(time.Time{}) != nil {
				w.Header().Set("Connection", "close")
			}
		}
	}
	http.Error(w, message, status)
}
