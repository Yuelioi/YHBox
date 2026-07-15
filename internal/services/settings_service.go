package services

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
)

// SettingsService 是 wails3 binding 暴露给 JS 的设置 RPC 入口。
type SettingsService struct {
	app     *App
	secrets *AISecrets
}

func NewSettingsService(app *App, secrets *AISecrets) *SettingsService {
	return &SettingsService{app: app, secrets: secrets}
}

// Get 返回当前 settings 的快照。前端启动时 hydrate Pinia store。
func (s *SettingsService) Get() Settings {
	return *s.app.Settings()
}

// Update RFC7386 deep merge 流程：clone → merge → Validate → swap → save → side-effects。
// 顺序保证 malformed patch 不会污染 settings.json。
// 前端 toast 拿到 error 直接展示。
func (s *SettingsService) Update(patchJSON string) error {
	patch := json.RawMessage(patchJSON)
	_, cur, err := s.app.MutateSettings(func(settings *Settings) error {
		previous := make(map[string]consentState, len(settings.AI.Profiles))
		for _, profile := range settings.AI.Profiles {
			previous[profile.Slot] = consentState{consent: profile.WorkflowConsent, expected: expectedAIConsent(profile)}
		}
		if err := ApplyMergePatch(settings, patch); err != nil {
			return fmt.Errorf("apply patch: %w", err)
		}
		// Consent is tied to the exact slot/profile artifact. Editing semantic
		// profile fields revokes an unchanged prior consent automatically.
		for index := range settings.AI.Profiles {
			profile := &settings.AI.Profiles[index]
			old, exists := previous[profile.Slot]
			if exists && old.consent != "" && profile.WorkflowConsent == old.consent && expectedAIConsent(*profile) != old.expected {
				profile.WorkflowConsent = ""
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

type consentState struct {
	consent  artifact.Digest
	expected artifact.Digest
}

func expectedAIConsent(configured AIModelSettings) artifact.Digest {
	profile, err := ai.SealModelProfile(configured.profileDraft())
	if err != nil {
		return ""
	}
	digest, err := ai.WorkflowConsentDigest(configured.Slot, profile)
	if err != nil {
		return ""
	}
	return digest
}

func (s *SettingsService) deleteRemovedAISecrets(before, after *Settings) {
	if s.secrets == nil {
		return
	}
	remaining := make(map[string]struct{}, len(after.AI.Profiles))
	for _, profile := range after.AI.Profiles {
		remaining[profile.Slot] = struct{}{}
	}
	for _, profile := range before.AI.Profiles {
		if _, ok := remaining[profile.Slot]; ok {
			continue
		}
		if err := s.secrets.DeleteSlot(profile.Slot); err != nil {
			logger := s.app.RootLogger()
			logger.Warn().Err(err).Str("tag", "AI_SECRET").
				Str("slot", profile.Slot).Msg("delete orphaned AI credential")
		}
	}
}
