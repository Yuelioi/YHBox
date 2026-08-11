package appbootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appcontrol"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/httpegress"
)

// PreparedInstallations is a fully constructed but unpublished execution
// environment generation.
// Commit and Abort are single-use terminal operations.
type PreparedInstallations struct {
	mu           sync.Mutex
	runtime      *Runtime
	factory      executionEnvironmentFactory
	applications appcontrol.Installations
	automation   automationinstalled.Installations
	ownsNetwork  bool
	state        uint8
}

// PreparedAutomation is kept as the source-compatible name for callers that
// only replace application and automation installations.
type PreparedAutomation = PreparedInstallations

func (runtime *Runtime) PrepareAutomation(
	applicationDrafts []appcontrol.InstallationDraft,
	automationDrafts []automationinstalled.InstallationDraft,
) (*PreparedAutomation, error) {
	factory, err := runtime.installationFactory()
	if err != nil {
		return nil, err
	}
	return runtime.prepareInstallations(factory, applicationDrafts, automationDrafts, false)
}

// PrepareInstallations constructs a complete replacement snapshot. Nothing is
// published until Commit succeeds. The settings service prepares this value
// before its durable save and publishes it afterward; those commits are
// ordered, but they are not a rollback-capable cross-layer transaction.
func (runtime *Runtime) PrepareInstallations(
	aiDrafts []ai.InstallationDraft,
	credentials ai.CredentialStore,
	httpDrafts []httpegress.InstallationDraft,
	applicationDrafts []appcontrol.InstallationDraft,
	automationDrafts []automationinstalled.InstallationDraft,
) (*PreparedInstallations, error) {
	current, err := runtime.installationFactory()
	if err != nil {
		return nil, err
	}
	aiInstallations, err := ai.Install(aiDrafts, credentials)
	if err != nil {
		return nil, fmt.Errorf("prepare AI installations: %w", err)
	}
	aiInstallations, err = aiInstallations.ForEvaluationArtifacts(runtime.Builtins.AIEvaluationArtifacts())
	if err != nil {
		aiInstallations.CloseIdleConnections()
		return nil, fmt.Errorf("prepare AI evaluation bindings: %w", err)
	}
	httpInstallations, err := httpegress.Install(httpDrafts)
	if err != nil {
		aiInstallations.CloseIdleConnections()
		return nil, fmt.Errorf("prepare HTTP installations: %w", err)
	}
	factory, err := current.withInstallations(aiInstallations, httpInstallations)
	if err != nil {
		aiInstallations.CloseIdleConnections()
		httpInstallations.CloseIdleConnections()
		return nil, err
	}
	prepared, err := runtime.prepareInstallations(factory, applicationDrafts, automationDrafts, true)
	if err != nil {
		aiInstallations.CloseIdleConnections()
		httpInstallations.CloseIdleConnections()
	}
	return prepared, err
}

func (runtime *Runtime) installationFactory() (executionEnvironmentFactory, error) {
	if runtime == nil || runtime.Application == nil {
		return executionEnvironmentFactory{}, errors.New("automation target runtime is unavailable")
	}
	runtime.automationMu.Lock()
	defer runtime.automationMu.Unlock()
	closed := runtime.closed
	if closed {
		return executionEnvironmentFactory{}, errors.New("automation target runtime is closed")
	}
	return runtime.factory, nil
}

func (runtime *Runtime) prepareInstallations(
	factory executionEnvironmentFactory,
	applicationDrafts []appcontrol.InstallationDraft,
	automationDrafts []automationinstalled.InstallationDraft,
	ownsNetwork bool,
) (*PreparedInstallations, error) {
	applications, err := appcontrol.Install(applicationDrafts)
	if err != nil {
		return nil, fmt.Errorf("prepare installed applications: %w", err)
	}
	automation, err := automationinstalled.Install(automationDrafts)
	if err != nil {
		return nil, fmt.Errorf("prepare installed automation targets: %w", err)
	}
	return &PreparedInstallations{
		runtime: runtime, factory: factory, applications: applications,
		automation: automation, ownsNetwork: ownsNetwork,
	}, nil
}

func (prepared *PreparedInstallations) Commit() error {
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
	if err := prepared.runtime.publish(
		prepared.factory,
		prepared.applications,
		prepared.automation,
		prepared.ownsNetwork,
	); err != nil {
		_ = prepared.automation.Close()
		prepared.closeOwnedNetwork()
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

func (prepared *PreparedInstallations) Abort() {
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
	prepared.closeOwnedNetwork()
}

func (prepared *PreparedInstallations) closeOwnedNetwork() {
	if prepared == nil || !prepared.ownsNetwork {
		return
	}
	prepared.factory.ai.CloseIdleConnections()
	prepared.factory.http.CloseIdleConnections()
}

func (runtime *Runtime) publish(
	factory executionEnvironmentFactory,
	applications appcontrol.Installations,
	installations automationinstalled.Installations,
	replaceNetwork bool,
) error {
	if runtime == nil || runtime.Application == nil || !applications.Valid() || !installations.Valid() {
		return errors.New("replacement automation installations are unavailable")
	}
	next, err := factory.seal(applications, installations)
	if err != nil {
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = next.generation.Retire()
		}
	}()

	runtime.automationMu.Lock()
	if runtime.closed {
		runtime.automationMu.Unlock()
		return errors.New("automation target runtime is closed")
	}
	previous := runtime.current
	previousAI, previousHTTP := runtime.ai, runtime.http
	if err := runtime.authoring.Replace(next.generation); err != nil {
		runtime.automationMu.Unlock()
		return err
	}
	if err := runtime.Application.ReplaceExecutionEnvironment(next.profile, next.policy, next.providers, next.acquire); err != nil {
		_ = runtime.authoring.Replace(previous.generation)
		runtime.automationMu.Unlock()
		return err
	}
	runtime.current = next
	if replaceNetwork {
		runtime.factory = factory
		runtime.ai = factory.ai
		runtime.http = factory.http
	}
	published = true
	runtime.automationMu.Unlock()

	_ = previous.generation.Retire()
	if replaceNetwork {
		previousAI.CloseIdleConnections()
		previousHTTP.CloseIdleConnections()
	}
	runtime.automationMu.Lock()
	runtime.retired = append(runtime.retired, previous.generation)
	runtime.reapRetiredLocked()
	runtime.automationMu.Unlock()
	return nil
}

func (runtime *Runtime) reapRetiredLocked() {
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

// AuthoringTargets returns the stable live handle shared by recording, assets
// and local tools. Publication updates every copy of this handle.
func (runtime *Runtime) AuthoringTargets() automationinstalled.AuthoringTargets {
	if runtime == nil {
		return automationinstalled.AuthoringTargets{}
	}
	return runtime.authoring
}

func (runtime *Runtime) Close(ctx context.Context) error {
	if runtime == nil || runtime.Application == nil {
		return nil
	}
	runtime.automationMu.Lock()
	runtime.closed = true
	runtime.reapRetiredLocked()
	current, retired, closeErr := runtime.current, append([]automationinstalled.Generation(nil), runtime.retired...), runtime.closeErr
	runtime.retired = nil
	runtime.automationMu.Unlock()
	err := runtime.Application.Close(ctx)
	err = errors.Join(err, closeErr, current.generation.Retire(), current.generation.WaitClosed(ctx))
	for _, generation := range retired {
		err = errors.Join(err, generation.WaitClosed(ctx))
	}
	runtime.ai.CloseIdleConnections()
	runtime.http.CloseIdleConnections()
	return err
}
