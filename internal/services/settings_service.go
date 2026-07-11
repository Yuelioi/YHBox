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
	patch := json.RawMessage(patchJSON)
	_, cur, err := s.app.MutateSettings(func(settings *Settings) error {
		if err := ApplyMergePatch(settings, patch); err != nil {
			return fmt.Errorf("apply patch: %w", err)
		}
		return nil
	}, func(old, cur *Settings) {
		// Side effects run inside the serialized settings writer boundary, so
		// an older commit cannot overwrite external state from a newer commit.
		if cur.UI.Autostart != old.UI.Autostart {
			if err := ApplyAutostart(cur.UI.Autostart); err != nil {
				log := s.app.RootLogger()
				log.Warn().Err(err).Str("tag", "SYSTEM").
					Bool("enabled", cur.UI.Autostart).
					Msg("自启注册表更新失败（settings 仍已保存）")
			}
		}
		if sink := s.app.GetLogSink(); sink != nil {
			ls := cur.UI.Logger
			dir := ls.FileDir
			if !ls.WriteFile {
				dir = ""
			}
			sink.SetFileWriter(dir)
		}
	})
	if err != nil && cur == nil {
		return err
	}
	commitErr := err

	// 通知所有 webview（尤其独立悬浮窗这种自带 store 的窗口）设置已变 → 各自 reload。
	s.app.Emit("settings:changed", map[string]any{})
	return commitErr
}
