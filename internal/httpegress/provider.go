package httpegress

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

const (
	ProviderABI     = "https://schemas.yotta.dev/provider-abi/resource/v1"
	KindHTTPSession = "network/http-session"
	TargetKind      = "http-origin"
	OperationGet    = "get"
	providerImpl    = "origin-bound-http-get/v1"

	CodeInvalidRequest    = "network.invalid_request"
	CodeResolutionDenied  = "network.resolution_denied"
	CodeRequestFailed     = "network.request_failed"
	CodeResponseTooLarge  = "network.response_too_large"
	CodeInvalidResponse   = "network.invalid_response"
	CodeContractViolation = "network.contract_violation"
)

type Failure struct {
	Code  string
	Cause error
}

func (e *Failure) Error() string {
	if e == nil {
		return "HTTP egress failure"
	}
	if e.Cause == nil {
		return e.Code
	}
	return e.Code + ": " + e.Cause.Error()
}
func (e *Failure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type CapabilityScope struct {
	Method string `json:"method"`
}
type GetRequest struct {
	Path  string              `json:"path"`
	Query map[string][]string `json:"query"`
}
type GetResponse struct {
	StatusCode  int    `json:"statusCode"`
	Body        string `json:"body"`
	ContentType string `json:"contentType"`
}

type provider struct {
	profile Profile
	client  *http.Client
}
type session struct {
	mu     sync.Mutex
	closed bool
}

func NewProvider(profile Profile) (resource.Provider, error) {
	if !profile.Valid() {
		return nil, errors.New("HTTP egress provider requires a profile")
	}
	draft := profile.Machine()
	timeout := time.Duration(draft.TimeoutMilliseconds) * time.Millisecond
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: nil, ForceAttemptHTTP2: true, DisableCompression: false,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout: timeout, ResponseHeaderTimeout: timeout,
		DialContext: restrictedDialer(dialer, draft.AllowPrivateNetwork),
	}
	client := &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	return &provider{profile: profile, client: client}, nil
}

func (p *provider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.Kind != KindHTTPSession || request.CredentialBindingID != "" || len(request.Operations) != 1 || request.Operations[0] != OperationGet {
		return nil, failure(CodeContractViolation, errors.New("invalid HTTP egress session request"))
	}
	var config map[string]any
	if err := decodeExact(request.Config, &config, 1024); err != nil || len(config) != 0 {
		return nil, failure(CodeContractViolation, errors.New("HTTP egress session config must be empty"))
	}
	var scope CapabilityScope
	if err := decodeExact(request.CapabilityScope, &scope, 1024); err != nil || scope.Method != "GET" {
		return nil, failure(CodeContractViolation, errors.New("HTTP egress capability scope is invalid"))
	}
	return &session{}, nil
}

func (p *provider) Invoke(ctx context.Context, object any, operation string, payload []byte) ([]byte, error) {
	opened, ok := object.(*session)
	if !ok {
		return nil, failure(CodeContractViolation, errors.New("HTTP egress object has the wrong type"))
	}
	opened.mu.Lock()
	defer opened.mu.Unlock()
	if opened.closed || operation != OperationGet {
		return nil, failure(CodeContractViolation, errors.New("HTTP egress session is closed or operation is unsupported"))
	}
	var request GetRequest
	if err := decodeExact(payload, &request, 128<<10); err != nil {
		return nil, failure(CodeInvalidRequest, err)
	}
	requestURL, err := p.requestURL(request)
	if err != nil {
		return nil, failure(CodeInvalidRequest, err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, failure(CodeInvalidRequest, err)
	}
	httpRequest.Header.Set("Accept", "text/plain, application/json;q=0.9, */*;q=0.1")
	httpRequest.Header.Set("User-Agent", "Yotta-Workflow/3.1")
	response, err := p.client.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		var denied *resolutionDenied
		if errors.As(err, &denied) {
			return nil, failure(CodeResolutionDenied, err)
		}
		return nil, failure(CodeRequestFailed, err)
	}
	defer response.Body.Close()
	limit := p.profile.Machine().ResponseByteLimit
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, failure(CodeRequestFailed, err)
	}
	if int64(len(body)) > limit {
		return nil, failure(CodeResponseTooLarge, errors.New("HTTP response exceeded the installed byte limit"))
	}
	if !utf8.Valid(body) {
		return nil, failure(CodeInvalidResponse, errors.New("HTTP response body is not UTF-8 text"))
	}
	contentType := response.Header.Get("Content-Type")
	if len(contentType) > 255 || strings.ContainsAny(contentType, "\r\n") {
		return nil, failure(CodeInvalidResponse, errors.New("HTTP response content type is invalid"))
	}
	return artifact.Marshal(GetResponse{StatusCode: response.StatusCode, Body: string(body), ContentType: contentType})
}

func (p *provider) Close(_ context.Context, object any) error {
	opened, ok := object.(*session)
	if !ok {
		return failure(CodeContractViolation, errors.New("HTTP egress object has the wrong type"))
	}
	opened.mu.Lock()
	opened.closed = true
	opened.mu.Unlock()
	return nil
}

func (p *provider) CloseIdleConnections() { p.client.CloseIdleConnections() }

func (p *provider) requestURL(request GetRequest) (string, error) {
	if request.Path == "" || len(request.Path) > 4096 || !strings.HasPrefix(request.Path, "/") || strings.HasPrefix(request.Path, "//") || strings.ContainsAny(request.Path, "\\\r\n?#") {
		return "", errors.New("HTTP path must be an absolute path without query or fragment")
	}
	path, err := url.ParseRequestURI(request.Path)
	if err != nil || path.IsAbs() || path.Host != "" || path.RawQuery != "" {
		return "", errors.New("HTTP path is invalid")
	}
	query := url.Values{}
	keys := make([]string, 0, len(request.Query))
	for key := range request.Query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 128 {
		return "", errors.New("HTTP query has too many keys")
	}
	for _, key := range keys {
		if key == "" || len(key) > 256 || strings.ContainsAny(key, "\x00\r\n") || len(request.Query[key]) > 128 {
			return "", errors.New("HTTP query is invalid")
		}
		for _, value := range request.Query[key] {
			if len(value) > 4096 || strings.ContainsAny(value, "\x00\r\n") {
				return "", errors.New("HTTP query is invalid")
			}
			query.Add(key, value)
		}
	}
	origin, _ := url.Parse(p.profile.Machine().Origin)
	origin.Path, origin.RawPath, origin.RawQuery = path.Path, path.RawPath, query.Encode()
	if len(origin.String()) > 16<<10 {
		return "", errors.New("HTTP request URL exceeds byte budget")
	}
	return origin.String(), nil
}

type resolutionDenied struct{ host string }

func (e *resolutionDenied) Error() string { return "HTTP destination address is denied" }

func restrictedDialer(dialer *net.Dialer, allowPrivate bool) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil || len(addresses) == 0 {
			return nil, errors.Join(err, errors.New("HTTP destination did not resolve"))
		}
		for _, ip := range addresses {
			if !allowedAddress(ip, allowPrivate) {
				return nil, &resolutionDenied{host: host}
			}
		}
		var dialErrors []error
		for _, ip := range addresses {
			connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return connection, nil
			}
			dialErrors = append(dialErrors, err)
		}
		return nil, errors.Join(dialErrors...)
	}
}

var deniedPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("198.18.0.0/15"),
}

func allowedAddress(address netip.Addr, allowPrivate bool) bool {
	address = address.Unmap()
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || !address.IsGlobalUnicast() {
		return allowPrivate && address.IsLoopback()
	}
	if allowPrivate {
		return true
	}
	if address.IsLoopback() || address.IsPrivate() {
		return false
	}
	for _, prefix := range deniedPublicPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func OpenGetResponse(raw []byte, maximum int64) (GetResponse, error) {
	var response GetResponse
	if maximum <= 0 || maximum > MaxResponseBytes || decodeExact(raw, &response, int(maximum)*6+4096) != nil || response.StatusCode < 100 || response.StatusCode > 599 || int64(len(response.Body)) > maximum || !utf8.ValidString(response.Body) || len(response.ContentType) > 255 || strings.ContainsAny(response.ContentType, "\r\n") {
		return GetResponse{}, failure(CodeContractViolation, errors.New("invalid HTTP egress response"))
	}
	canonical, err := artifact.Marshal(response)
	if err != nil || !bytes.Equal(canonical, raw) {
		return GetResponse{}, failure(CodeContractViolation, errors.New("HTTP egress response is not canonical"))
	}
	return response, nil
}

func ProviderArtifactDigest(profile Profile) (artifact.Digest, error) {
	if !profile.Valid() {
		return "", errors.New("HTTP provider artifact requires a profile")
	}
	manifest, err := artifact.Marshal(map[string]any{"providerAbi": ProviderABI, "implementation": providerImpl, "profileDigest": profile.Digest(), "profile": json.RawMessage(profile.Bytes())})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/provider-implementation-manifest/v1", manifest)
}

func decodeExact(raw []byte, target any, maximum int) error {
	if len(raw) == 0 || len(raw) > maximum {
		return errors.New("JSON payload exceeds byte budget")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSON payload contains trailing values")
	}
	return nil
}

func failure(code string, cause error) error { return &Failure{Code: code, Cause: cause} }

var _ resource.Provider = (*provider)(nil)
