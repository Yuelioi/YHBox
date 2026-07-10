package io

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	stdio "io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&Fetch{}) }

const (
	fetchInMethod          = "Method"
	fetchInURL             = "URL"
	fetchInHeaders         = "Headers"
	fetchInCookies         = "Cookies"
	fetchInBody            = "Body"
	fetchInBodyMode        = "BodyMode"
	fetchInTimeoutMs       = "TimeoutMs"
	fetchInFollowRedirects = "FollowRedirects"
	fetchInFailOnStatus    = "FailOnStatus"
	fetchInMaxBytes        = "MaxBytes"

	fetchOutDone       = "Done"
	fetchOutFail       = "Fail"
	fetchDataStatus    = "StatusCode"
	fetchDataBody      = "Body"
	fetchDataJSON      = "JSON"
	fetchDataHeaders   = "Headers"
	fetchDataDuration  = "DurationMs"
	fetchDataError     = "Error"
	fetchDataCode      = "Code"
	defaultFetchMax    = 1 << 20
	defaultFetchTimout = 10_000
)

type Fetch struct{}

func (Fetch) Spec() node.Spec {
	return node.Spec{
		Kind:     "Fetch",
		Category: "IO",
		Inputs: []node.InputSpec{
			{Name: "In", Type: node.TypeExec},
			{Name: fetchInMethod, Type: "String", Default: "GET",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "GET"},
							{Value: "POST"},
							{Value: "PUT"},
							{Value: "PATCH"},
							{Value: "DELETE"},
							{Value: "HEAD"},
						}})}},
			{Name: fetchInURL, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: fetchInHeaders, Type: "JSON", Widget: node.WidgetSpec{Kind: "json"}},
			{Name: fetchInCookies, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: fetchInBody, Type: "*"},
			{Name: fetchInBodyMode, Type: "String", Default: "none",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "none"},
							{Value: "text"},
							{Value: "json"},
						}})}},
			{Name: fetchInTimeoutMs, Type: "Number", Default: json.Number("10000"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: fetchInFollowRedirects, Type: "Bool", Default: true, Widget: node.WidgetSpec{Kind: "checkbox"}},
			{Name: fetchInFailOnStatus, Type: "Bool", Default: true, Widget: node.WidgetSpec{Kind: "checkbox"}},
			{Name: fetchInMaxBytes, Type: "Integer", Default: json.Number("1048576"), Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: fetchOutDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: fetchDataStatus, Type: "Integer"},
				{Name: fetchDataBody, Type: "String"},
				{Name: fetchDataJSON, Type: "JSON", Optional: true},
				{Name: fetchDataHeaders, Type: "JSON"},
				{Name: fetchDataDuration, Type: "Integer"},
			}},
			{Name: fetchOutFail, Type: node.TypeExec, Semantic: "error", Data: []node.DataField{
				{Name: fetchDataError, Type: "String"},
				{Name: fetchDataCode, Type: "String"},
				{Name: fetchDataStatus, Type: "Integer", Optional: true},
				{Name: fetchDataBody, Type: "String", Optional: true},
			}},
		},
	}
}

func (Fetch) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	targetURL := strings.TrimSpace(in.String(fetchInURL))
	parsed, err := url.Parse(targetURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fetchFail(ctx, string(node.CodeError), "Fetch: URL 必须是 http/https 绝对地址", 0, ""), nil
	}

	body, contentType, err := buildFetchBody(in)
	if err != nil {
		return fetchFail(ctx, string(node.CodeError), err.Error(), 0, ""), nil
	}
	req, err := http.NewRequestWithContext(ctx.Context(), fetchMethod(in), targetURL, body)
	if err != nil {
		return fetchFail(ctx, string(node.CodeError), err.Error(), 0, ""), nil
	}
	applyFetchHeaders(req, in.JSON(fetchInHeaders))
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	if cookies := strings.TrimSpace(in.String(fetchInCookies)); cookies != "" && req.Header.Get("Cookie") == "" {
		req.Header.Set("Cookie", cookies)
	}

	timeoutMs := in.Int(fetchInTimeoutMs)
	if timeoutMs <= 0 {
		timeoutMs = defaultFetchTimout
	}
	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	if !in.Bool(fetchInFollowRedirects) {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		code := string(node.CodeError)
		if isTimeout(err) {
			code = string(node.CodeTimeout)
		}
		return fetchFail(ctx, code, err.Error(), 0, ""), nil
	}
	defer resp.Body.Close()

	maxBytes := in.Int(fetchInMaxBytes)
	if maxBytes <= 0 {
		maxBytes = defaultFetchMax
	}
	raw, err := readLimitedBody(resp.Body, maxBytes)
	if err != nil {
		return fetchFail(ctx, string(node.CodeError), err.Error(), resp.StatusCode, ""), nil
	}
	bodyText := string(raw)
	if in.Bool(fetchInFailOnStatus) && resp.StatusCode >= 400 {
		return fetchFail(ctx, "http_status", resp.Status, resp.StatusCode, bodyText), nil
	}

	var jsonValue any
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "json") && len(raw) > 0 {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&jsonValue); err == nil {
			var extra any
			if err := dec.Decode(&extra); err != stdio.EOF {
				jsonValue = nil
			}
		}
	}
	return ctx.Out(fetchOutDone).
		Set(fetchDataStatus, resp.StatusCode).
		Set(fetchDataBody, bodyText).
		Set(fetchDataJSON, jsonValue).
		Set(fetchDataHeaders, headersToJSON(resp.Header)).
		Set(fetchDataDuration, durationMs).
		Fire(), nil
}

func fetchMethod(in node.Inputs) string {
	method := strings.ToUpper(strings.TrimSpace(in.String(fetchInMethod)))
	switch method {
	case "POST", "PUT", "PATCH", "DELETE", "HEAD":
		return method
	default:
		return "GET"
	}
}

func buildFetchBody(in node.Inputs) (stdio.Reader, string, error) {
	mode := strings.ToLower(strings.TrimSpace(in.String(fetchInBodyMode)))
	switch mode {
	case "", "none":
		return nil, "", nil
	case "json":
		data, err := json.Marshal(in.Raw(fetchInBody))
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(data), "application/json", nil
	case "text":
		return strings.NewReader(node.FormatValue(in.Raw(fetchInBody))), "text/plain; charset=utf-8", nil
	default:
		return nil, "", errors.New("Fetch: BodyMode 仅支持 none/text/json")
	}
}

func applyFetchHeaders(req *http.Request, headers map[string]any) {
	for k, v := range headers {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		req.Header.Set(key, node.FormatValue(v))
	}
}

func readLimitedBody(body stdio.Reader, maxBytes int) ([]byte, error) {
	raw, err := stdio.ReadAll(stdio.LimitReader(body, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBytes {
		return nil, errors.New("Fetch: 响应超过 MaxBytes 上限")
	}
	return raw, nil
}

func headersToJSON(headers http.Header) map[string]any {
	out := make(map[string]any, len(headers))
	for k, vals := range headers {
		if len(vals) == 1 {
			out[k] = vals[0]
		} else {
			items := make([]any, len(vals))
			for i, v := range vals {
				items[i] = v
			}
			out[k] = items
		}
	}
	return out
}

func fetchFail(ctx node.Ctx, code, message string, statusCode int, body string) node.Outputs {
	b := ctx.Out(fetchOutFail).
		Set(fetchDataError, message).
		Set(fetchDataCode, code)
	if statusCode != 0 {
		b = b.Set(fetchDataStatus, statusCode)
	}
	if body != "" {
		b = b.Set(fetchDataBody, body)
	}
	return b.Fire()
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
