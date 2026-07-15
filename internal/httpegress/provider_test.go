package httpegress

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

func TestProviderGetsInstalledOriginWithoutFollowingRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/redirect":
			http.Redirect(response, request, "/secret", http.StatusFound)
		case "/secret":
			t.Error("provider followed redirect")
			http.Error(response, "unexpected redirect", http.StatusInternalServerError)
		default:
			response.Header().Set("Content-Type", "application/json")
			response.Header().Set("Set-Cookie", "secret=value")
			_, _ = response.Write([]byte(`{"ok":true}`))
		}
	}))
	defer server.Close()

	provider, object := openTestProvider(t, server.URL, true, 1024)
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
	if err != nil || redirectOutcome.StatusCode != http.StatusFound {
		t.Fatalf("redirect outcome = %+v, err = %v", redirectOutcome, err)
	}
}

func TestProviderDeniesPrivateResolutionUnlessProfileAllowsIt(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) { _, _ = response.Write([]byte("ok")) }))
	defer server.Close()
	provider, object := openTestProvider(t, server.URL, false, 1024)
	_, err := invokeGet(t, provider, object, GetRequest{Path: "/", Query: map[string][]string{}})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Code != CodeResolutionDenied {
		t.Fatalf("error = %v", err)
	}
}

func TestProviderEnforcesResponseAndRequestShapeBudgets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) { _, _ = response.Write([]byte("too large")) }))
	defer server.Close()
	provider, object := openTestProvider(t, server.URL, true, 3)
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

func TestAddressPolicyDeniesLocalAndSpecialNetworks(t *testing.T) {
	denied := []string{"127.0.0.1", "10.0.0.1", "169.254.1.1", "100.64.0.1", "198.18.0.1", "::1", "fe80::1"}
	for _, raw := range denied {
		if allowedAddress(netip.MustParseAddr(raw), false) {
			t.Fatalf("allowed %s", raw)
		}
	}
	if !allowedAddress(netip.MustParseAddr("8.8.8.8"), false) {
		t.Fatal("denied public address")
	}
	if !allowedAddress(netip.MustParseAddr("127.0.0.1"), true) {
		t.Fatal("explicit private-network profile did not allow loopback")
	}
}

func openTestProvider(t *testing.T, origin string, allowPrivate bool, limit int64) (resource.Provider, any) {
	t.Helper()
	profile, err := SealProfile(ProfileDraft{Origin: origin, AllowPrivateNetwork: allowPrivate, ResponseByteLimit: limit, TimeoutMilliseconds: 5000})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(profile)
	if err != nil {
		t.Fatal(err)
	}
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindHTTPSession, Operations: []string{OperationGet}, Config: []byte(`{}`), CapabilityScope: json.RawMessage(`{"method":"GET"}`),
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
