package services

import (
	"encoding/json"
	"fmt"
)

// SettingsService 是 wails3 binding 暴露给 JS 的设置 RPC 入口。
type SettingsService struct {
	app     *App
	secrets *AISecrets
}

func NewSettingsService(app *App, secrets ...*AISecrets) *SettingsService {
	service := &SettingsService{app: app}
	if len(secrets) > 0 {
		service.secrets = secrets[0]
	}
	return service
}

// Get 返回当前 settings 的快照。前端启动时 hydrate Pinia store。
func (s *SettingsService) Get() Settings {
	cur := s.app.Settings()
	for index := range cur.AI.Connections {
		cur.AI.Connections[index].APIKey = ""
	}
	return *cur
}

// Update RFC7386 deep merge 流程：clone → merge → Validate → swap → save → side-effects。
// 顺序保证 malformed patch 不会污染 settings.json。
// 前端 toast 拿到 error 直接展示。
func (s *SettingsService) Update(patchJSON string) error {
	patch := json.RawMessage(patchJSON)
	_, cur, err := s.app.MutateSettings(func(settings *Settings) error {
		legacyKeys := make(map[string]string, len(settings.AI.Connections))
		for _, connection := range settings.AI.Connections {
			if connection.APIKey != "" {
				legacyKeys[connection.ID] = connection.APIKey
			}
		}
		if err := ApplyMergePatch(settings, patch); err != nil {
			return fmt.Errorf("apply patch: %w", err)
		}
		// Metadata-only connection arrays from the modern UI omit apiKey. If a
		// startup migration failed, retain the legacy plaintext until a later
		// migration succeeds instead of silently losing the only credential.
		for index := range settings.AI.Connections {
			connection := &settings.AI.Connections[index]
			if connection.APIKey == "" {
				connection.APIKey = legacyKeys[connection.ID]
			}
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
		if s.app.logs != nil {
			s.app.logs.Configure(cur.UI.Logger)
		}
		s.deleteRemovedAISecrets(old, cur)
	})
	if err != nil && cur == nil {
		return err
	}
	commitErr := err

	// 通知所有 webview（尤其独立悬浮窗这种自带 store 的窗口）设置已变 → 各自 reload。
	s.app.Emit("settings:changed", map[string]any{})
	return commitErr
}

func (s *SettingsService) deleteRemovedAISecrets(before, after *Settings) {
	if s.secrets == nil {
		return
	}
	remaining := make(map[string]struct{}, len(after.AI.Connections))
	for _, connection := range after.AI.Connections {
		remaining[connection.ID] = struct{}{}
	}
	for _, connection := range before.AI.Connections {
		if _, ok := remaining[connection.ID]; ok {
			continue
		}
		if err := s.secrets.Delete(connection.ID); err != nil {
			logger := s.app.RootLogger()
			logger.Warn().Err(err).Str("tag", "AI_SECRET").
				Str("connectionId", connection.ID).Msg("delete orphaned AI credential")
		}
	}
}
