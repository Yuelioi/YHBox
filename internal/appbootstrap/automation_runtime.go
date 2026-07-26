package appbootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appcontrol"
	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodes"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
)

// AutomationTargetRuntime is the single owner of installed target
// generations. Settings submits intent; authoring and Runs hold stable handles
// or leases and never coordinate provider publication themselves.
type AutomationTargetRuntime struct {
	mu              sync.Mutex
	application     *appcore.Application
	ai              ai.Installations
	http            httpegress.Installations
	current         automationinstalled.Generation
	retired         []automationinstalled.Generation
	authoring       automationinstalled.AuthoringTargets
	workflowTargets []workflowinstallation.TargetState
	environment     automationEnvironmentConfig
	closed          bool
	closeErr        error
}

func (runtime *AutomationTargetRuntime) WorkflowTargetStates() []workflowinstallation.TargetState {
	if runtime == nil {
		return nil
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return append([]workflowinstallation.TargetState(nil), runtime.workflowTargets...)
}

type automationEnvironmentConfig struct {
	builtins            nodes.Builtins
	blobDigest          artifact.Digest
	streamDigest        artifact.Digest
	workspaceFileDigest artifact.Digest
	scriptRuntime       *scriptengine.Runtime
	pluginFeatures      []string
	now                 func() time.Time
	grantTTL            time.Duration
	baseProviders       map[string]run.InstalledProvider
}

// PreparedAutomation is a fully constructed but unpublished target generation.
// Commit and Abort are single-use terminal operations.
type PreparedAutomation struct {
	mu           sync.Mutex
	runtime      *AutomationTargetRuntime
	applications appcontrol.Installations
	automation   automationinstalled.Installations
	state        uint8
}

func (runtime *AutomationTargetRuntime) AuthoringTargets() automationinstalled.AuthoringTargets {
	if runtime == nil {
		return automationinstalled.AuthoringTargets{}
	}
	return runtime.authoring
}

func (runtime *AutomationTargetRuntime) Prepare(applicationDrafts []appcontrol.InstallationDraft, automationDrafts []automationinstalled.InstallationDraft) (*PreparedAutomation, error) {
	if runtime == nil || runtime.application == nil {
		return nil, errors.New("automation target runtime is unavailable")
	}
	runtime.mu.Lock()
	closed := runtime.closed
	runtime.mu.Unlock()
	if closed {
		return nil, errors.New("automation target runtime is closed")
	}
	applications, err := appcontrol.Install(applicationDrafts)
	if err != nil {
		return nil, fmt.Errorf("prepare installed applications: %w", err)
	}
	automation, err := automationinstalled.Install(automationDrafts)
	if err != nil {
		return nil, fmt.Errorf("prepare installed automation targets: %w", err)
	}
	return &PreparedAutomation{runtime: runtime, applications: applications, automation: automation}, nil
}

func (prepared *PreparedAutomation) Commit() error {
	if prepared == nil || prepared.runtime == nil {
		return errors.New("prepared automation generation is unavailable")
	}
	prepared.mu.Lock()
	if prepared.state != 0 {
		prepared.mu.Unlock()
		return errors.New("prepared automation generation is already finalized")
	}
	prepared.state = 1
	prepared.mu.Unlock()
	if err := prepared.runtime.publish(prepared.applications, prepared.automation); err != nil {
		_ = prepared.automation.Close()
		prepared.mu.Lock()
		prepared.state = 3
		prepared.mu.Unlock()
		return err
	}
	prepared.mu.Lock()
	prepared.state = 2
	prepared.mu.Unlock()
	return nil
}

func (prepared *PreparedAutomation) Abort() {
	if prepared == nil {
		return
	}
	prepared.mu.Lock()
	if prepared.state != 0 {
		prepared.mu.Unlock()
		return
	}
	prepared.state = 3
	prepared.mu.Unlock()
	_ = prepared.automation.Close()
}

func (runtime *AutomationTargetRuntime) publish(applications appcontrol.Installations, installations automationinstalled.Installations) error {
	if runtime == nil || runtime.application == nil || !applications.Valid() || !installations.Valid() {
		return errors.New("replacement automation installations are unavailable")
	}
	nextGeneration, err := automationinstalled.NewGeneration(installations)
	if err != nil {
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = nextGeneration.Retire()
		}
	}()

	runtime.mu.Lock()
	if runtime.closed {
		runtime.mu.Unlock()
		return errors.New("automation target runtime is closed")
	}
	environment := runtime.environment
	profile, err := buildHostProfile(
		environment.builtins, environment.blobDigest, environment.streamDigest, environment.workspaceFileDigest,
		environment.scriptRuntime, environment.pluginFeatures, runtime.ai, runtime.http, applications, installations,
	)
	if err != nil {
		runtime.mu.Unlock()
		return err
	}
	policy, err := NewBuiltinPolicy(environment.now, environment.grantTTL, runtime.ai, runtime.http, applications, installations)
	if err != nil {
		runtime.mu.Unlock()
		return err
	}
	providers := cloneProviders(environment.baseProviders)
	for _, installed := range applications.Entries() {
		if existing, ok := providers[installed.ProviderID]; ok && (existing.ArtifactDigest != installed.ProviderArtifact || existing.ABI != appcontrol.ProviderABI) {
			runtime.mu.Unlock()
			return errors.New("conflicting replacement application provider")
		}
		providers[installed.ProviderID] = run.InstalledProvider{ArtifactDigest: installed.ProviderArtifact, ABI: appcontrol.ProviderABI, Provider: installed.Provider}
	}
	for _, installed := range installations.Entries() {
		if existing, ok := providers[installed.ProviderID]; ok && (existing.ArtifactDigest != installed.ProviderArtifact || existing.ABI != automationinstalled.ProviderABI) {
			runtime.mu.Unlock()
			return errors.New("conflicting replacement automation provider")
		}
		providers[installed.ProviderID] = run.InstalledProvider{ArtifactDigest: installed.ProviderArtifact, ABI: automationinstalled.ProviderABI, Provider: installed.Provider}
	}

	previous := runtime.current
	if err := runtime.authoring.Replace(nextGeneration); err != nil {
		runtime.mu.Unlock()
		return err
	}
	if err := runtime.application.ReplaceExecutionEnvironment(profile, policy, providers, providerLease(nextGeneration)); err != nil {
		_ = runtime.authoring.Replace(previous)
		runtime.mu.Unlock()
		return err
	}
	runtime.current = nextGeneration
	runtime.workflowTargets = workflowTargetStates(installations)
	published = true
	runtime.mu.Unlock()

	_ = previous.Retire()
	runtime.mu.Lock()
	runtime.retired = append(runtime.retired, previous)
	runtime.reapRetiredLocked()
	runtime.mu.Unlock()
	return nil
}

func workflowTargetStates(installations automationinstalled.Installations) []workflowinstallation.TargetState {
	entries := installations.Entries()
	result := make([]workflowinstallation.TargetState, 0, len(entries))
	for _, installed := range entries {
		profile := installed.Profile.Machine()
		result = append(result, workflowinstallation.TargetState{
			TargetInstallationID: installed.TargetID,
			TargetKind:           profile.TargetKind, AdapterKind: profile.AdapterKind,
			ProfileVersion: profile.ProfileVersion,
			Available:      true, Authorized: true,
		})
	}
	return result
}

func (runtime *AutomationTargetRuntime) reapRetiredLocked() {
	remaining := runtime.retired[:0]
	for _, generation := range runtime.retired {
		closed, err := generation.Closed()
		if !closed {
			remaining = append(remaining, generation)
			continue
		}
		runtime.closeErr = errors.Join(runtime.closeErr, err)
	}
	runtime.retired = remaining
}

func (runtime *AutomationTargetRuntime) Close(ctx context.Context) error {
	if runtime == nil || runtime.application == nil {
		return nil
	}
	runtime.mu.Lock()
	runtime.closed = true
	runtime.reapRetiredLocked()
	current, retired, closeErr := runtime.current, append([]automationinstalled.Generation(nil), runtime.retired...), runtime.closeErr
	runtime.retired = nil
	runtime.mu.Unlock()
	err := runtime.application.Close(ctx)
	err = errors.Join(err, closeErr, current.Retire(), current.WaitClosed(ctx))
	for _, generation := range retired {
		err = errors.Join(err, generation.WaitClosed(ctx))
	}
	return err
}

func providerLease(generation automationinstalled.Generation) func() (func(), error) {
	return func() (func(), error) {
		lease, err := generation.Acquire()
		if err != nil {
			return nil, err
		}
		return lease.Release, nil
	}
}

func cloneProviders(source map[string]run.InstalledProvider) map[string]run.InstalledProvider {
	result := make(map[string]run.InstalledProvider, len(source))
	for id, provider := range source {
		result[id] = provider
	}
	return result
}

// AuthoringTargets returns the stable live handle shared by recording, assets
// and local tools. Publication updates every copy of this handle.
func (runtime *Runtime) AuthoringTargets() automationinstalled.AuthoringTargets {
	if runtime == nil || runtime.automationTargets == nil {
		return automationinstalled.AuthoringTargets{}
	}
	return runtime.automationTargets.AuthoringTargets()
}

func (runtime *Runtime) PrepareAutomation(applicationDrafts []appcontrol.InstallationDraft, automationDrafts []automationinstalled.InstallationDraft) (*PreparedAutomation, error) {
	if runtime == nil || runtime.automationTargets == nil {
		return nil, errors.New("automation target runtime is unavailable")
	}
	return runtime.automationTargets.Prepare(applicationDrafts, automationDrafts)
}

func (runtime *Runtime) ReplaceAutomation(applications appcontrol.Installations, installations automationinstalled.Installations) error {
	if runtime == nil || runtime.automationTargets == nil {
		return errors.New("automation target runtime is unavailable")
	}
	return runtime.automationTargets.publish(applications, installations)
}

func (runtime *Runtime) Start(ctx context.Context) error {
	if runtime == nil || runtime.Application == nil {
		return errors.New("app runtime is unavailable")
	}
	return runtime.Application.Start(ctx)
}

func (runtime *Runtime) Close(ctx context.Context) error {
	if runtime == nil {
		return nil
	}
	err := runtime.automationTargets.Close(ctx)
	runtime.ai.CloseIdleConnections()
	runtime.http.CloseIdleConnections()
	return err
}
