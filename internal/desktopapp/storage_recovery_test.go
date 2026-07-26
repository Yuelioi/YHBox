package desktopapp

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/storage"
	storagemigrate "github.com/yottaapp/yotta/internal/storage/migrate"
)

func TestStorageRecoveryHandlerResumesAndRequestsRestart(t *testing.T) {
	root := filepath.Join(t.TempDir(), "layout-1")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "root.json"),
		[]byte("{\"format\":\"yotta.storage-root\",\"version\":\"1\"}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	quit := make(chan struct{}, 1)
	controller := &storageRecoveryController{
		options: storagemigrate.Options{Root: root},
		quit:    func() { quit <- struct{}{} },
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/action",
		strings.NewReader(`{"action":"resume"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Yotta-Recovery", "1")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	response := httptest.NewRecorder()

	controller.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("resume status = %d, body = %s", response.Code, response.Body.String())
	}
	if !controller.proceed.Load() {
		t.Fatal("resume did not request process restart")
	}
	select {
	case <-quit:
	case <-time.After(time.Second):
		t.Fatal("resume did not close the recovery window")
	}
	health, err := storage.Inspect(context.Background(), storage.InspectOptions{Root: root})
	if err != nil || !health.Supported || health.LayoutVersion != storage.LayoutVersion {
		t.Fatalf("health after GUI resume = %#v, %v", health, err)
	}
}

func TestStorageRecoveryHandlerRejectsCrossSiteMutation(t *testing.T) {
	controller := &storageRecoveryController{}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/action",
		strings.NewReader(`{"action":"resume"}`),
	)
	request.Header.Set("X-Yotta-Recovery", "1")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	response := httptest.NewRecorder()

	controller.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-site status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestStorageRecoveryHandlerRejectsTrailingJSON(t *testing.T) {
	controller := &storageRecoveryController{}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/action",
		strings.NewReader(`{"action":"resume"} {}`),
	)
	request.Header.Set("X-Yotta-Recovery", "1")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	response := httptest.NewRecorder()

	controller.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestStorageRecoveryAssetsKeepOneAccessibleDarkTheme(t *testing.T) {
	err := fs.WalkDir(storageRecoveryAssets, "storage_recovery_assets", func(
		path string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		raw, err := storageRecoveryAssets.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(raw, []byte("—")) || bytes.Contains(raw, []byte("–")) {
			t.Fatalf("%s contains a forbidden visible dash", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
