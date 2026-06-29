package services

import (
	"encoding/json"
	"fmt"
)

// SettingsService 是 wails3 binding 暴露给 JS 的设置 RPC 入口。
type SettingsService struct {
	app *App
}

func NewSettingsService(app *App) *SettingsService { return &SettingsService{app: app} }

// Get 返回当前 settings 的快照。前端启动时 hydrate Pinia store。
func (s *SettingsService) Get() Settings {
	cur := s.app.Settings()
	return *cur
}

// Update RFC7386 deep merge 流程：clone → merge → Validate → swap → save → side-effects。
// 顺序保证 malformed patch 不会污染 settings.json。
// 前端 toast 拿到 error 直接展示。
func (s *SettingsService) Update(patchJSON string) error {
	old := s.app.Settings()
	wasAutostart := old.UI.Autostart

	cur := s.app.SnapshotSettings()
	patch := json.RawMessage(patchJSON)
	if err := ApplyMergePatch(cur, patch); err != nil {
		return fmt.Errorf("apply patch: %w", err)
	}
	if err := cur.Validate(); err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	s.app.SwapSettings(cur)
	if err := s.app.SaveSettings(); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	// 注册表写入失败不回滚 settings：用户选了"开机自启=true"但 AV 拦截写注册表，
	// 我们留住用户的意图（设置仍是 true），下次启动会再尝试一次。仅 log warning。
	if cur.UI.Autostart != wasAutostart {
		if err := ApplyAutostart(cur.UI.Autostart); err != nil {
			log := s.app.RootLogger()
			log.Warn().Err(err).Str("tag", "SYSTEM").
				Bool("enabled", cur.UI.Autostart).
				Msg("自启注册表更新失败（settings 仍已保存）")
		}
	}

	// Logger 字段变更 → LogSink 重接 (无侵入 — LogSink 通过 app 取)
	if sink := s.app.GetLogSink(); sink != nil {
		ls := cur.UI.Logger
		dir := ls.FileDir
		if !ls.WriteFile {
			dir = ""
		}
		sink.SetFileWriter(dir)
	}
	// 通知所有 webview（尤其独立悬浮窗这种自带 store 的窗口）设置已变 → 各自 reload。
	s.app.Emit("settings:changed", map[string]any{})
	return nil
}
