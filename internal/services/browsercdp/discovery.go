package browsercdp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
)

const DefaultEndpoint = "http://127.0.0.1:9222"

type TargetInfo struct {
	ID                   string
	Type                 string
	Title                string
	URL                  string
	WebSocketDebuggerURL string
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

func (s *Service) ListTargets(ctx context.Context, endpoint string) ([]TargetInfo, error) {
	endpoint = s.endpoint(endpoint)
	u, err := endpointURL(endpoint, "/json")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("cdp discovery %s returned %s", u, res.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	return ParseTargetsJSON(raw)
}

func (s *Service) TargetByID(ctx context.Context, endpoint, id string) (TargetInfo, bool, error) {
	targets, err := s.ListTargets(ctx, endpoint)
	if err != nil {
		return TargetInfo{}, false, err
	}
	for _, t := range targets {
		if t.ID == id {
			return t, true, nil
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
	return &http.Client{Timeout: 3 * time.Second}
}

func ParseTargetsJSON(raw []byte) ([]TargetInfo, error) {
	var entries []struct {
		ID                   string `json:"id"`
		Type                 string `json:"type"`
		Title                string `json:"title"`
		URL                  string `json:"url"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	out := make([]TargetInfo, 0, len(entries))
	for _, e := range entries {
		if e.ID == "" || e.WebSocketDebuggerURL == "" {
			continue
		}
		if e.Type != "" && e.Type != "page" {
			continue
		}
		out = append(out, TargetInfo{
			ID:                   e.ID,
			Type:                 e.Type,
			Title:                e.Title,
			URL:                  e.URL,
			WebSocketDebuggerURL: e.WebSocketDebuggerURL,
		})
	}
	return out, nil
}

func endpointURL(endpoint, path string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
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
		ID:          "browser:" + info.ID,
		Kind:        target.KindBrowserCDP,
		DisplayName: name,
		Ref:         target.TargetRef{BrowserID: info.ID},
		Resolution:  target.Size{W: width, H: height},
		Metadata: map[string]any{
			"endpoint":             strings.TrimSpace(endpoint),
			"url":                  info.URL,
			"title":                info.Title,
			"webSocketDebuggerUrl": info.WebSocketDebuggerURL,
		},
	}
}
