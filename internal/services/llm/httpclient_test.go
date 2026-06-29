package llm

import "testing"

func TestTunedHTTPClientNoTimeout(t *testing.T) {
	c := tunedHTTPClient()
	if c.Timeout != 0 {
		t.Errorf("client.Timeout = %v, want 0 (per-call ctx deadline instead)", c.Timeout)
	}
	if c.Transport == nil {
		t.Error("transport must be set")
	}
}
