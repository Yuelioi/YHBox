package desktopapp

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/yottaapp/yotta/internal/storage"
	storagemigrate "github.com/yottaapp/yotta/internal/storage/migrate"
)

//go:embed storage_recovery_assets/*
var storageRecoveryAssets embed.FS

type storageRecoveryController struct {
	options storagemigrate.Options
	assets  http.Handler
	quit    func()

	proceed atomic.Bool
	mu      sync.Mutex
	message string
}

type storageRecoveryView struct {
	Status  storagemigrate.RecoveryStatus `json:"status"`
	Message string                        `json:"message,omitempty"`
}

type storageRecoveryAction struct {
	Action string `json:"action"`
	Name   string `json:"name,omitempty"`
}

func runStorageRecovery(config Config, cause error) error {
	assets, err := fs.Sub(storageRecoveryAssets, "storage_recovery_assets")
	if err != nil {
		return err
	}
	controller := &storageRecoveryController{
		options: storagemigrate.Options{Root: config.StorageRoot, MaxRuns: 65536},
		assets:  http.FileServer(http.FS(assets)),
		message: safeStorageRecoveryError(config.StorageRoot, cause),
	}
	recoveryApp := application.New(application.Options{
		Name:        "Yotta Storage Recovery",
		Description: "Recover an interrupted Yotta storage migration",
		Assets: application.AssetOptions{
			Handler:        controller,
			DisableLogging: true,
		},
	})
	controller.quit = recoveryApp.Quit
	recoveryApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Yotta 存储恢复",
		Width:            900,
		Height:           680,
		MinWidth:         720,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(9, 9, 11),
		URL:              "/",
	})
	if err := recoveryApp.Run(); err != nil {
		return fmt.Errorf("run storage recovery window: %w", err)
	}
	if controller.proceed.Load() {
		if err := restartCurrentProcess(); err != nil {
			return fmt.Errorf("restart Yotta after storage recovery: %w", err)
		}
	}
	return nil
}

func (c *storageRecoveryController) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/api/status":
		if request.Method != http.MethodGet {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.writeStatus(response, request)
	case "/api/action":
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if request.Header.Get("X-Yotta-Recovery") != "1" ||
			!trustedRecoveryFetch(request.Header.Get("Sec-Fetch-Site")) {
			http.Error(response, "recovery action is not trusted", http.StatusForbidden)
			return
		}
		c.runAction(response, request)
	default:
		c.setAssetHeaders(response, request.URL.Path)
		c.assets.ServeHTTP(response, request)
	}
}

func (c *storageRecoveryController) writeStatus(
	response http.ResponseWriter,
	request *http.Request,
) {
	status, err := storagemigrate.InspectRecovery(request.Context(), c.options)
	if err != nil {
		c.setMessage(safeStorageRecoveryError(c.options.Root, err))
	}
	c.writeJSON(response, storageRecoveryView{Status: status, Message: c.getMessage()})
}

func (c *storageRecoveryController) runAction(
	response http.ResponseWriter,
	request *http.Request,
) {
	var action storageRecoveryAction
	decoder := json.NewDecoder(io.LimitReader(request.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&action); err != nil {
		http.Error(response, "invalid recovery action", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		http.Error(response, "recovery action must contain one JSON value", http.StatusBadRequest)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := request.Context()
	var resultMessage string
	var actionErr error
	committed := false
	switch action.Action {
	case "resume":
		result, err := resumeOrApplyStorageMigration(ctx, c.options)
		actionErr = err
		committed = result.Journal.State == storagemigrate.StateCommitted
		if committed {
			resultMessage = "迁移已验证并提交。Yotta 将重新启动。"
		}
	case "rollback":
		_, actionErr = storagemigrate.Rollback(ctx, c.options)
		if actionErr == nil {
			resultMessage = "已恢复迁移前快照。你可以重新尝试迁移，或关闭 Yotta。"
		}
	case "quarantine":
		_, actionErr = storagemigrate.QuarantineLegacyRun(ctx, c.options, action.Name)
		if actionErr == nil {
			resultMessage = "阻塞记录已隔离。原始字节保留在迁移隔离区。"
		}
	case "restore":
		_, actionErr = storagemigrate.RestoreLegacyRun(ctx, c.options, action.Name)
		if actionErr == nil {
			resultMessage = "隔离记录已恢复到旧 Run 目录。"
		}
	case "export":
		var destination string
		destination, actionErr = storagemigrate.ExportDiagnosticsToProfile(ctx, c.options)
		if actionErr == nil {
			resultMessage = "诊断已导出到 " + destination
		}
	default:
		http.Error(response, "unknown recovery action", http.StatusBadRequest)
		return
	}
	if actionErr != nil {
		resultMessage = safeStorageRecoveryError(c.options.Root, actionErr)
	}
	c.message = resultMessage
	status, statusErr := storagemigrate.InspectRecovery(ctx, c.options)
	if statusErr != nil {
		c.message = safeStorageRecoveryError(c.options.Root, errors.Join(actionErr, statusErr))
	}
	c.writeJSON(response, storageRecoveryView{Status: status, Message: c.message})
	if committed {
		c.proceed.Store(true)
		if c.quit != nil {
			go c.quit()
		}
	}
}

func resumeOrApplyStorageMigration(
	ctx context.Context,
	options storagemigrate.Options,
) (storagemigrate.Result, error) {
	status, err := storagemigrate.InspectRecovery(ctx, options)
	if err != nil {
		return storagemigrate.Result{}, err
	}
	if status.Journal == nil {
		return storagemigrate.Apply(ctx, options)
	}
	return storagemigrate.Resume(ctx, options)
}

func (c *storageRecoveryController) writeJSON(response http.ResponseWriter, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if err := json.NewEncoder(response).Encode(value); err != nil {
		return
	}
}

func (c *storageRecoveryController) setAssetHeaders(response http.ResponseWriter, path string) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("Referrer-Policy", "no-referrer")
	if path == "/" || strings.HasSuffix(path, ".html") {
		response.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; "+
				"img-src 'none'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'",
		)
	}
}

func (c *storageRecoveryController) getMessage() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.message
}

func (c *storageRecoveryController) setMessage(message string) {
	c.mu.Lock()
	c.message = message
	c.mu.Unlock()
}

func trustedRecoveryFetch(value string) bool {
	return value == "" || value == "none" || value == "same-origin"
}

func safeStorageRecoveryError(root string, err error) string {
	if err == nil {
		return ""
	}
	resolved, resolveErr := storage.Resolve(root)
	if resolveErr != nil {
		return err.Error()
	}
	value := strings.ReplaceAll(err.Error(), resolved.Root, storage.RedactedRoot)
	return strings.ReplaceAll(value, filepath.ToSlash(resolved.Root), storage.RedactedRoot)
}
