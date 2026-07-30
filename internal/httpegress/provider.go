package httpegress

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
	"github.com/yottaapp/yotta/pkg/version"
)

const (
	ProviderABI     = "https://schemas.yotta.dev/provider-abi/resource/v1"
	KindHTTPSession = "network/http-session"
	TargetKind      = "http-target"
	OperationGet    = "get"
	providerImpl    = "origin-bound-http-get/v1"

	CodeInvalidRequest    = "network.invalid_request"
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
	client := &http.Client{Timeout: timeout}
	return &provider{profile: profile, client: client}, nil
}

func (p *provider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.Kind != KindHTTPSession || len(request.Operations) != 1 || request.Operations[0] != OperationGet {
		return nil, failure(CodeContractViolation, errors.New("invalid HTTP egress session request"))
	}
	var config map[string]any
	if err := decodeExact(request.Config, &config, 1024); err != nil || len(config) != 0 {
		return nil, failure(CodeContractViolation, errors.New("HTTP egress session config must be empty"))
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
	if err := decodeExact(payload, &request, 0); err != nil {
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
	httpRequest.Header.Set("User-Agent", "Yotta/"+version.Version)
	response, err := p.client.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
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
	if strings.ContainsAny(contentType, "\r\n") {
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
	if request.Path == "" || strings.ContainsAny(request.Path, "\r\n") {
		return "", errors.New("HTTP request path is invalid")
	}
	reference, err := url.Parse(request.Path)
	if err != nil || reference.IsAbs() || reference.Host != "" {
		return "", errors.New("HTTP request path must be relative to the configured base URL")
	}
	query := url.Values{}
	keys := make([]string, 0, len(request.Query))
	for key := range request.Query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "" || strings.ContainsAny(key, "\x00\r\n") {
			return "", errors.New("HTTP query is invalid")
		}
		for _, value := range request.Query[key] {
			if strings.ContainsAny(value, "\x00\r\n") {
				return "", errors.New("HTTP query is invalid")
			}
			query.Add(key, value)
		}
	}
	baseURL, _ := url.Parse(p.profile.Machine().Origin)
	requestURL := baseURL.ResolveReference(reference)
	merged := requestURL.Query()
	for key, values := range query {
		for _, value := range values {
			merged.Add(key, value)
		}
	}
	requestURL.RawQuery = merged.Encode()
	return requestURL.String(), nil
}

func OpenGetResponse(raw []byte, maximum int64) (GetResponse, error) {
	var response GetResponse
	if decodeExact(raw, &response, 0) != nil || response.StatusCode < 100 || response.StatusCode > 599 ||
		maximum > 0 && int64(len(response.Body)) > maximum || !utf8.ValidString(response.Body) ||
		strings.ContainsAny(response.ContentType, "\r\n") {
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
	if len(raw) == 0 || maximum > 0 && len(raw) > maximum {
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
