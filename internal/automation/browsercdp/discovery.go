// Package browsercdp owns discovery and transport for configured Chrome
// DevTools Protocol page targets.
package browsercdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

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
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", errors.New("browser CDP endpoint must be an absolute HTTP or HTTPS URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func ValidateWebSocketURL(raw, endpoint, targetID string) (string, error) {
	_ = endpoint
	_ = targetID
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse browser CDP websocket URL: %w", err)
	}
	if (parsed.Scheme != "ws" && parsed.Scheme != "wss") || parsed.Host == "" {
		return "", errors.New("browser CDP websocket must be an absolute ws or wss URL")
	}
	return parsed.String(), nil
}

func ValidateLoopbackWebSocketURL(raw string) (string, error) {
	return ValidateWebSocketURL(raw, "", "")
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
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	targets, err := ParseTargetsJSON(raw)
	if err != nil {
		return nil, err
	}
	for index := range targets {
		validated, err := ValidateWebSocketURL(targets[index].WebSocketDebuggerURL, "", "")
		if err != nil {
			return nil, fmt.Errorf("browser CDP page %q returned an invalid websocket URL: %w", targets[index].ID, err)
		}
		targets[index].WebSocketDebuggerURL = validated
	}
	return targets, nil
}

func (s *Service) TargetByID(ctx context.Context, endpoint, id string) (TargetInfo, bool, error) {
	if !browserTargetIDPattern.MatchString(id) {
		return TargetInfo{}, false, errors.New("browser CDP page descriptor is invalid")
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
	return &http.Client{}
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
