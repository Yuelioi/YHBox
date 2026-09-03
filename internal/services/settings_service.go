package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/apperr"
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
		previousAI := make(map[string]evaluationState, len(settings.AI.Profiles))
		for _, profile := range settings.AI.Profiles {
			previousAI[profile.Slot] = evaluationState{
				subject:          expectedAIEvaluationSubject(profile),
				evaluationReport: profile.EvaluationReport.Digest, evaluated: profile.Evaluation != ai.EvaluationUnverified,
			}
		}
		previousApplications := make(map[string]struct{}, len(settings.Applications.Profiles))
		for _, configured := range settings.Applications.Profiles {
			previousApplications[configured.Slot] = struct{}{}
		}
		if err := ApplyMergePatch(settings, patch); err != nil {
			return fmt.Errorf("%w: %v", &settingsUpdateError{ID: "settings.update.invalid", Category: apperr.CategoryValidation}, err)
		}
		removeTargetsForRemovedApplications(settings, previousApplications)
		for index := range settings.AI.Profiles {
			profile := &settings.AI.Profiles[index]
			old, exists := previousAI[profile.Slot]
			if exists && old.evaluated && old.subject != expectedAIEvaluationSubject(*profile) && profile.EvaluationReport.Digest == old.evaluationReport {
				profile.Evaluation = ai.EvaluationUnverified
				profile.EvaluationSuite = ""
				profile.EvaluationReport = ai.EvalReportArtifact{}
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
					Msg("自启计划任务更新失败（settings 仍已保存）")
			}
		}
		if s.app.logs != nil {
			s.app.logs.Configure(cur.UI.Logger)
		}
		s.deleteRemovedAISecrets(old, cur)
	})
	if err != nil && cur == nil {
		var provider apperr.EnvelopeProvider
		if errors.As(err, &provider) {
			return err
		}
		return fmt.Errorf("%w: %v", &settingsUpdateError{ID: "settings.update_failed", Category: apperr.CategoryInfrastructure, Retryable: true}, err)
	}
	var commitErr error
	if err != nil {
		commitErr = fmt.Errorf("%w: %v", &settingsUpdateError{ID: "settings.update.committed_sync_failed", Category: apperr.CategoryInfrastructure}, err)
	}

	// 通知所有 webview（尤其独立悬浮窗这种自带 store 的窗口）设置已变 → 各自 reload。
	s.app.Emit("settings:changed", map[string]any{})
	return commitErr
}

type settingsUpdateError struct {
	ID        string
	Category  string
	Retryable bool
}

func (e *settingsUpdateError) Error() string { return e.ID }
func (e *settingsUpdateError) RPCErrorEnvelope() apperr.Envelope {
	return apperr.Envelope{ID: e.ID, Category: e.Category, Retryable: e.Retryable}
}

func removeTargetsForRemovedApplications(settings *Settings, previous map[string]struct{}) {
	remainingApplications := make(map[string]struct{}, len(settings.Applications.Profiles))
	for _, application := range settings.Applications.Profiles {
		remainingApplications[application.Slot] = struct{}{}
	}
	removedApplications := make(map[string]struct{})
	for slot := range previous {
		if _, exists := remainingApplications[slot]; !exists {
			removedApplications[slot] = struct{}{}
		}
	}
	if len(removedApplications) == 0 {
		return
	}
	targets := settings.Automation.Targets[:0]
	for _, target := range settings.Automation.Targets {
		_, removed := removedApplications[target.applicationSlot()]
		if target.requiresApplication() && removed {
			continue
		}
		targets = append(targets, target)
	}
	settings.Automation.Targets = targets
}

type evaluationState struct {
	subject          artifact.Digest
	evaluationReport artifact.Digest
	evaluated        bool
}

func expectedAIEvaluationSubject(configured AIModelSettings) artifact.Digest {
	profile, err := ai.SealModelProfile(configured.profileDraft())
	if err != nil {
		return ""
	}
	digest, err := ai.EvaluationSubjectDigest(profile)
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
