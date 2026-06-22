package llm

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestStatusToKind(t *testing.T) {
	cases := map[int]ErrKind{
		http.StatusUnauthorized:        KindAuth,
		http.StatusForbidden:           KindAuth,
		http.StatusTooManyRequests:     KindRateLimit,
		http.StatusNotFound:            KindNotFound,
		http.StatusBadRequest:          KindBadRequest,
		http.StatusInternalServerError: KindUpstream,
		http.StatusBadGateway:          KindUpstream,
		418:                            KindUnknown,
	}
	for status, want := range cases {
		if got := statusToKind(status); got != want {
			t.Errorf("statusToKind(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestKindOf(t *testing.T) {
	if got := KindOf(&CodedError{Kind: KindRateLimit, Err: errors.New("x")}); got != KindRateLimit {
		t.Errorf("KindOf(CodedError) = %q, want rateLimit", got)
	}
	if got := KindOf(context.DeadlineExceeded); got != KindTimeout {
		t.Errorf("KindOf(deadline) = %q, want timeout", got)
	}
	if got := KindOf(errors.New("plain")); got != KindUnknown {
		t.Errorf("KindOf(plain) = %q, want unknown", got)
	}
}

func TestIsOfficialEndpoint(t *testing.T) {
	if !isOfficialEndpoint("openai", "") {
		t.Error("empty baseURL must be official")
	}
	if !isOfficialEndpoint("openai", "https://api.openai.com/v1") {
		t.Error("api.openai.com must be official")
	}
	if isOfficialEndpoint("openai", "http://localhost:11434/v1") {
		t.Error("localhost must NOT be official")
	}
	if !isOfficialEndpoint("anthropic", "https://api.anthropic.com") {
		t.Error("api.anthropic.com must be official")
	}
}
