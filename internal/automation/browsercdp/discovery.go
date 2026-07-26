// Package browsercdp owns discovery and transport for explicitly installed,
// loopback Chrome DevTools Protocol page targets.
package browsercdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
)

const DefaultEndpoint = "http://127.0.0.1:9222"

var browserTargetIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)

type TargetInfo struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type Service struct {
	Endpoint   string
	HTTPClient *http.Client
}

func NewService(endpoint string) *Service {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultEndpoint
	}
	return &Service{Endpoint: endpoint}
}

func CanonicalEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultEndpoint
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse browser CDP endpoint: %w", err)
	}
	if parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("browser CDP endpoint must be an HTTP loopback origin without credentials, path, query, or fragment")
	}
	host, port, err := loopbackHostPort(parsed, "9222")
	if err != nil {
		return "", err
	}
	return "http://" + net.JoinHostPort(host, port), nil
}

func ValidateWebSocketURL(raw, endpoint, targetID string) (string, error) {
	if !browserTargetIDPattern.MatchString(targetID) {
		return "", errors.New("browser CDP page identity is invalid")
	}
	canonicalEndpoint, err := CanonicalEndpoint(endpoint)
	if err != nil {
		return "", err
	}
	expectedOrigin, _ := url.Parse(canonicalEndpoint)
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse browser CDP websocket URL: %w", err)
	}
	if parsed.Scheme != "ws" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("browser CDP websocket must use an uncredentialed loopback ws URL")
	}
	host, port, err := loopbackHostPort(parsed, "")
	if err != nil {
		return "", err
	}
	expectedHost, expectedPort, _ := net.SplitHostPort(expectedOrigin.Host)
	if host != expectedHost || port != expectedPort {
		return "", errors.New("browser CDP websocket authority drifted from the installed discovery endpoint")
	}
	if parsed.EscapedPath() != "/devtools/page/"+url.PathEscape(targetID) {
		return "", errors.New("browser CDP websocket page identity drifted from the selected target")
	}
	return "ws://" + net.JoinHostPort(host, port) + parsed.EscapedPath(), nil
}

func ValidateLoopbackWebSocketURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "ws" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("browser CDP websocket must use an uncredentialed loopback ws URL")
	}
	host, port, err := loopbackHostPort(parsed, "")
	if err != nil {
		return "", err
	}
	if port == "" || !strings.HasPrefix(parsed.EscapedPath(), "/devtools/") {
		return "", errors.New("browser CDP websocket URL is incomplete")
	}
	return "ws://" + net.JoinHostPort(host, port) + parsed.EscapedPath(), nil
}

func loopbackHostPort(parsed *url.URL, defaultPort string) (string, string, error) {
	host := parsed.Hostname()
	address := net.ParseIP(host)
	if address == nil || !address.IsLoopback() {
		return "", "", errors.New("browser CDP authority must use a literal loopback IP address")
	}
	port := parsed.Port()
	if port == "" {
		port = defaultPort
	}
	if port == "" {
		return "", "", errors.New("browser CDP authority requires an explicit port")
	}
	return address.String(), port, nil
}

func (s *Service) ListTargets(ctx context.Context, endpoint string) ([]TargetInfo, error) {
	canonical, err := CanonicalEndpoint(s.endpoint(endpoint))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonical+"/json", nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("cdp discovery %s/json returned %s", canonical, res.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	targets, err := ParseTargetsJSON(raw)
	if err != nil {
		return nil, err
	}
	for index := range targets {
		validated, err := ValidateWebSocketURL(targets[index].WebSocketDebuggerURL, canonical, targets[index].ID)
		if err != nil {
			return nil, fmt.Errorf("reject browser CDP page %q: %w", targets[index].ID, err)
		}
		targets[index].WebSocketDebuggerURL = validated
	}
	return targets, nil
}

func (s *Service) TargetByID(ctx context.Context, endpoint, id string) (TargetInfo, bool, error) {
	if !browserTargetIDPattern.MatchString(id) {
		return TargetInfo{}, false, errors.New("browser CDP page identity is invalid")
	}
	targets, err := s.ListTargets(ctx, endpoint)
	if err != nil {
		return TargetInfo{}, false, err
	}
	for _, candidate := range targets {
		if candidate.ID == id {
			return candidate, true, nil
		}
	}
	return TargetInfo{}, false, nil
}

func (s *Service) endpoint(endpoint string) string {
	if strings.TrimSpace(endpoint) != "" {
		return endpoint
	}
	if s != nil && strings.TrimSpace(s.Endpoint) != "" {
		return s.Endpoint
	}
	return DefaultEndpoint
}

func (s *Service) client() *http.Client {
	if s != nil && s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("browser CDP discovery redirects are not allowed")
		},
	}
}

func ParseTargetsJSON(raw []byte) ([]TargetInfo, error) {
	var entries []TargetInfo
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	out := make([]TargetInfo, 0, len(entries))
	for _, entry := range entries {
		if !browserTargetIDPattern.MatchString(entry.ID) || entry.WebSocketDebuggerURL == "" || entry.Type != "" && entry.Type != "page" {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func TargetFromInfo(endpoint string, info TargetInfo, width, height int, name string) target.Target {
	if name == "" {
		name = info.Title
	}
	if name == "" {
		name = info.URL
	}
	if name == "" {
		name = info.ID
	}
	return target.Target{
		ID: "browser:" + info.ID, Kind: target.KindBrowserCDP, DisplayName: name,
		Ref: target.TargetRef{BrowserID: info.ID}, Resolution: target.Size{W: width, H: height},
		Metadata: map[string]any{
			"endpoint": endpoint, "url": info.URL, "title": info.Title,
			"webSocketDebuggerUrl": info.WebSocketDebuggerURL,
		},
	}
}
