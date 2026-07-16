package services

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/httpegress"
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
		previousHTTP := make(map[string]consentState, len(settings.Network.HTTPOrigins))
		for _, origin := range settings.Network.HTTPOrigins {
			previousHTTP[origin.Slot] = consentState{consent: origin.WorkflowConsent, expected: expectedHTTPConsent(origin)}
		}
		previousApplications := make(map[string]consentState, len(settings.Applications.Profiles))
		for _, configured := range settings.Applications.Profiles {
			previousApplications[configured.Slot] = consentState{consent: configured.WorkflowConsent, expected: expectedApplicationConsent(configured)}
		}
		previousAutomation := make(map[string]consentState, len(settings.Automation.Win32Targets))
		for _, configured := range settings.Automation.Win32Targets {
			previousAutomation[configured.Slot] = consentState{consent: configured.WorkflowConsent, expected: expectedAutomationConsent(*settings, configured)}
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
		for index := range settings.Network.HTTPOrigins {
			origin := &settings.Network.HTTPOrigins[index]
			old, exists := previousHTTP[origin.Slot]
			if exists && old.consent != "" && origin.WorkflowConsent == old.consent && expectedHTTPConsent(*origin) != old.expected {
				origin.WorkflowConsent = ""
			}
		}
		for index := range settings.Applications.Profiles {
			configured := &settings.Applications.Profiles[index]
			old, exists := previousApplications[configured.Slot]
			if exists && old.consent != "" && configured.WorkflowConsent == old.consent && expectedApplicationConsent(*configured) != old.expected {
				configured.WorkflowConsent = ""
			}
		}
		for index := range settings.Automation.Win32Targets {
			configured := &settings.Automation.Win32Targets[index]
			old, exists := previousAutomation[configured.Slot]
			if exists && old.consent != "" && configured.WorkflowConsent == old.consent && expectedAutomationConsent(*settings, *configured) != old.expected {
				configured.WorkflowConsent = ""
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

func expectedHTTPConsent(configured HTTPOriginSettings) artifact.Digest {
	profile, err := httpegress.SealProfile(configured.profileDraft())
	if err != nil {
		return ""
	}
	digest, err := httpegress.WorkflowConsentDigest(configured.Slot, profile)
	if err != nil {
		return ""
	}
	return digest
}

func expectedApplicationConsent(configured InstalledApplicationSettings) artifact.Digest {
	profile, err := appcontrol.SealProfile(configured.profileDraft())
	if err != nil {
		return ""
	}
	digest, err := appcontrol.WorkflowConsentDigest(configured.Slot, profile)
	if err != nil {
		return ""
	}
	return digest
}

func expectedAutomationConsent(settings Settings, configured InstalledAutomationTargetSettings) artifact.Digest {
	var application InstalledApplicationSettings
	found := false
	for _, candidate := range settings.Applications.Profiles {
		if candidate.Slot == configured.ApplicationSlot {
			application, found = candidate, true
			break
		}
	}
	if !found {
		return ""
	}
	profile, err := automationinstalled.SealProfile(configured.profileDraft(application))
	if err != nil {
		return ""
	}
	digest, err := automationinstalled.WorkflowConsentDigest(configured.Slot, profile)
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
