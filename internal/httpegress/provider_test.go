package httpegress

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

func TestProviderGetsConfiguredBaseURLAndFollowsRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/redirect":
			http.Redirect(response, request, "/secret", http.StatusFound)
		case "/secret":
			_, _ = response.Write([]byte(`{"redirected":true}`))
		default:
			response.Header().Set("Content-Type", "application/json")
			response.Header().Set("Set-Cookie", "secret=value")
			_, _ = response.Write([]byte(`{"ok":true}`))
		}
	}))
	defer server.Close()

	provider, object := openTestProvider(t, server.URL, 1024)
	raw, err := invokeGet(t, provider, object, GetRequest{Path: "/hello", Query: map[string][]string{"a": {"1"}}})
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := OpenGetResponse(raw, 1024)
	if err != nil || outcome.StatusCode != http.StatusOK || outcome.Body != `{"ok":true}` || outcome.ContentType != "application/json" {
		t.Fatalf("outcome = %+v, err = %v", outcome, err)
	}
	redirect, err := invokeGet(t, provider, object, GetRequest{Path: "/redirect", Query: map[string][]string{}})
	if err != nil {
		t.Fatal(err)
	}
	redirectOutcome, err := OpenGetResponse(redirect, 1024)
	if err != nil || redirectOutcome.StatusCode != http.StatusOK || redirectOutcome.Body != `{"redirected":true}` {
		t.Fatalf("redirect outcome = %+v, err = %v", redirectOutcome, err)
	}
}

func TestProviderAllowsConfiguredPrivateNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) { _, _ = response.Write([]byte("ok")) }))
	defer server.Close()
	provider, object := openTestProvider(t, server.URL, 1024)
	raw, err := invokeGet(t, provider, object, GetRequest{Path: "/", Query: map[string][]string{}})
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := OpenGetResponse(raw, 1024)
	if err != nil || outcome.StatusCode != http.StatusOK || outcome.Body != "ok" {
		t.Fatalf("outcome = %+v, err = %v", outcome, err)
	}
}

func TestProviderEnforcesResponseAndRequestShapeBudgets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) { _, _ = response.Write([]byte("too large")) }))
	defer server.Close()
	provider, object := openTestProvider(t, server.URL, 3)
	_, err := invokeGet(t, provider, object, GetRequest{Path: "/", Query: map[string][]string{}})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Code != CodeResponseTooLarge {
		t.Fatalf("error = %v", err)
	}
	_, err = invokeGet(t, provider, object, GetRequest{Path: "https://evil.example/", Query: map[string][]string{}})
	if !errors.As(err, &failure) || failure.Code != CodeInvalidRequest {
		t.Fatalf("error = %v", err)
	}
}

func openTestProvider(t *testing.T, origin string, limit int64) (resource.Provider, any) {
	t.Helper()
	profile, err := SealProfile(ProfileDraft{Origin: origin, ResponseByteLimit: limit, TimeoutMilliseconds: 5000})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(profile)
	if err != nil {
		t.Fatal(err)
	}
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindHTTPSession, Operations: []string{OperationGet}, Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return provider, object
}

func invokeGet(t *testing.T, provider resource.Provider, object any, request GetRequest) ([]byte, error) {
	t.Helper()
	payload, err := artifact.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	return provider.Invoke(context.Background(), object, OperationGet, payload)
}
