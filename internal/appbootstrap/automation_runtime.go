package appbootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yottaapp/yotta/internal/appcontrol"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
)

// PreparedAutomation is a fully constructed but unpublished target generation.
// Commit and Abort are single-use terminal operations.
type PreparedAutomation struct {
	mu           sync.Mutex
	runtime      *Runtime
	applications appcontrol.Installations
	automation   automationinstalled.Installations
	state        uint8
}

func (runtime *Runtime) PrepareAutomation(applicationDrafts []appcontrol.InstallationDraft, automationDrafts []automationinstalled.InstallationDraft) (*PreparedAutomation, error) {
	if runtime == nil || runtime.Application == nil {
		return nil, errors.New("automation target runtime is unavailable")
	}
	runtime.automationMu.Lock()
	closed := runtime.closed
	runtime.automationMu.Unlock()
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

func (runtime *Runtime) publish(applications appcontrol.Installations, installations automationinstalled.Installations) error {
	if runtime == nil || runtime.Application == nil || !applications.Valid() || !installations.Valid() {
		return errors.New("replacement automation installations are unavailable")
	}
	next, err := runtime.factory.seal(applications, installations)
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
	published = true
	runtime.automationMu.Unlock()

	_ = previous.generation.Retire()
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
