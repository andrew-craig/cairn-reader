package main

import (
	"net/http"
	"testing"
	"time"
)

// TestServerTimeouts verifies that the server constructed by newHTTPServer (the
// same constructor main uses) has the expected timeouts, guarding against
// accidental removal of the slow-loris mitigations (O-5).
func TestServerTimeouts(t *testing.T) {
	srv := newHTTPServer(":8082", http.NewServeMux())

	if srv.ReadHeaderTimeout == 0 {
		t.Error("ReadHeaderTimeout must be non-zero (slow-loris mitigation)")
	}
	if srv.ReadTimeout == 0 {
		t.Error("ReadTimeout must be non-zero")
	}
	if srv.WriteTimeout == 0 {
		t.Error("WriteTimeout must be non-zero")
	}
	if srv.IdleTimeout == 0 {
		t.Error("IdleTimeout must be non-zero")
	}

	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, 5*time.Second)
	}
	if srv.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout = %v, want %v", srv.ReadTimeout, 15*time.Second)
	}
	if srv.WriteTimeout != 15*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", srv.WriteTimeout, 15*time.Second)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want %v", srv.IdleTimeout, 60*time.Second)
	}
}
