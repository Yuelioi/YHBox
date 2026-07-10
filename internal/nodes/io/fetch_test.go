package io

import (
	"context"
	"encoding/json"
	stdio "io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func runFetch(t *testing.T, cfg map[string]any) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&Fetch{})
	rn, _ := node.Get("Fetch")
	return node.RunNode(context.Background(), rn, nil, cfg, nil, node.StubServices(), false)
}

func TestFetch_GETJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Reply", "ok")
		_, _ = w.Write([]byte(`{"ok":true,"items":[1,2]}`))
	}))
	defer srv.Close()

	res := runFetch(t, map[string]any{"URL": srv.URL})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v data=%#v", res.ExitName, res.Error, res.OutputData)
	}
	if got := res.OutputData["StatusCode"]; got != 200 {
		t.Fatalf("StatusCode = %#v, want 200", got)
	}
	if got := res.OutputData["Body"]; got != `{"ok":true,"items":[1,2]}` {
		t.Fatalf("Body = %#v", got)
	}
	bodyJSON := res.OutputData["JSON"].(map[string]any)
	if bodyJSON["ok"] != true {
		t.Fatalf("JSON = %#v", bodyJSON)
	}
	headers := res.OutputData["Headers"].(map[string]any)
	if headers["X-Reply"] != "ok" {
		t.Fatalf("Headers = %#v", headers)
	}
	if got := res.OutputData["DurationMs"].(int64); got < 0 {
		t.Fatalf("DurationMs = %d, want >=0", got)
	}
}

func TestFetch_InvalidJSONResponseDoesNotExposePartialJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true} trailing`))
	}))
	defer srv.Close()

	res := runFetch(t, map[string]any{"URL": srv.URL})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	if got := res.OutputData["JSON"]; got != nil {
		t.Fatalf("invalid JSON response exposed partial JSON = %#v, want nil", got)
	}
}

func TestFetch_HeadersCookiesAndJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Token"); got != "abc" {
			t.Fatalf("X-Token = %q, want abc", got)
		}
		if got := r.Header.Get("Cookie"); got != "sid=1" {
			t.Fatalf("Cookie = %q, want sid=1", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		raw, _ := stdio.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["name"] != "yotta" {
			t.Fatalf("payload = %#v", payload)
		}
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	res := runFetch(t, map[string]any{
		"Method":   "POST",
		"URL":      srv.URL,
		"Headers":  map[string]any{"X-Token": "abc"},
		"Cookies":  "sid=1",
		"Body":     map[string]any{"name": "yotta"},
		"BodyMode": "json",
	})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
}

func TestFetch_FailOnStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer srv.Close()

	res := runFetch(t, map[string]any{"URL": srv.URL, "FailOnStatus": true})
	if res.Error != nil {
		t.Fatalf("Fetch should route HTTP status to Fail output, got error %v", res.Error)
	}
	if res.ExitName != "Fail" {
		t.Fatalf("exit=%q, want Fail", res.ExitName)
	}
	if got := res.OutputData["StatusCode"]; got != http.StatusTeapot {
		t.Fatalf("StatusCode = %#v, want 418", got)
	}
	if got := res.OutputData["Body"]; got == "" {
		t.Fatal("Fail body should include response body")
	}
}

func TestFetch_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer srv.Close()

	res := runFetch(t, map[string]any{"URL": srv.URL, "TimeoutMs": 10})
	if res.Error != nil {
		t.Fatalf("Fetch should route timeout to Fail output, got error %v", res.Error)
	}
	if res.ExitName != "Fail" || res.OutputData["Code"] != "timeout" {
		t.Fatalf("exit=%q data=%#v, want Fail timeout", res.ExitName, res.OutputData)
	}
}
