package installed

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yottaapp/yotta/internal/automation/target"
)

// AuthoringTargets is the host-only projection used by trusted local tooling.
// Workflow execution continues to use Resource Broker grants and never receives
// this surface or a native window handle.
type AuthoringTargets struct {
	state *authoringTargetState
}

type authoringTargetState struct {
	mu      sync.RWMutex
	current Generation
}

func NewAuthoringTargets(generation Generation) (AuthoringTargets, error) {
	if !generation.Valid() {
		return AuthoringTargets{}, errors.New("authoring targets require an installed automation generation")
	}
	return AuthoringTargets{state: &authoringTargetState{current: generation}}, nil
}

// Replace atomically publishes one already-prepared generation to every
// trusted local authoring service holding a copy of this handle. Retirement is
// owned by the caller after execution and authoring have both published next.
func (targets AuthoringTargets) Replace(generation Generation) error {
	if targets.state == nil || !generation.Valid() {
		return errors.New("replacement authoring generation is unavailable")
	}
	targets.state.mu.Lock()
	targets.state.current = generation
	targets.state.mu.Unlock()
	return nil
}

func (targets AuthoringTargets) ResolveWindow(ctx context.Context, slot string) (target.WindowHandle, error) {
	provider, release, err := targets.provider(slot)
	if err != nil {
		return target.WindowHandle{}, err
	}
	defer release()
	if ctx == nil {
		return target.WindowHandle{}, errors.New("resolve automation target context is required")
	}
	resolved, err := provider.driver.ResolveTarget(ctx)
	if err != nil {
		return target.WindowHandle{}, err
	}
	if resolved.Ref.HWND == 0 {
		return target.WindowHandle{}, errors.New("automation target is not a desktop window")
	}
	return target.WindowHandle{
		HWND: resolved.Ref.HWND, Title: resolved.DisplayName, ClientW: resolved.Resolution.W, ClientH: resolved.Resolution.H,
	}, nil
}

func (targets AuthoringTargets) ResolveTarget(ctx context.Context, slot string) (target.Target, error) {
	provider, release, err := targets.provider(slot)
	if err != nil {
		return target.Target{}, err
	}
	defer release()
	if ctx == nil {
		return target.Target{}, errors.New("resolve automation target context is required")
	}
	return provider.driver.ResolveTarget(ctx)
}

func (targets AuthoringTargets) CapturePNG(ctx context.Context, slot string) ([]byte, error) {
	provider, release, err := targets.provider(slot)
	if err != nil {
		return nil, err
	}
	defer release()
	if ctx == nil {
		return nil, errors.New("capture automation target context is required")
	}
	return provider.driver.Capture(ctx)
}

func (targets AuthoringTargets) Activate(ctx context.Context, slot string) error {
	provider, release, err := targets.provider(slot)
	if err != nil {
		return err
	}
	defer release()
	if ctx == nil {
		return errors.New("activate automation target context is required")
	}
	return provider.driver.Execute(ctx, OperationActivate, struct{}{})
}

// AcquireRecordingTarget activates and resolves a desktop target while keeping
// its exact installation generation alive for the caller's native session.
func (targets AuthoringTargets) AcquireRecordingTarget(ctx context.Context, slot string) (target.WindowHandle, int, func(), error) {
	provider, release, err := targets.provider(slot)
	if err != nil {
		return target.WindowHandle{}, 0, nil, err
	}
	fail := func(cause error) (target.WindowHandle, int, func(), error) {
		release()
		return target.WindowHandle{}, 0, nil, cause
	}
	if ctx == nil {
		return fail(errors.New("recording automation target context is required"))
	}
	desktop, ok := DesktopProfile(provider.profile)
	if !ok {
		return fail(errors.New("recording requires a desktop automation target"))
	}
	if err := provider.driver.Execute(ctx, OperationActivate, struct{}{}); err != nil {
		return fail(err)
	}
	resolved, err := provider.driver.ResolveTarget(ctx)
	if err != nil {
		return fail(err)
	}
	if resolved.Ref.HWND == 0 {
		return fail(errors.New("recording target is not a desktop window"))
	}
	return target.WindowHandle{
		HWND: resolved.Ref.HWND, Title: resolved.DisplayName, ClientW: resolved.Resolution.W, ClientH: resolved.Resolution.H,
	}, int(desktop.MouseCounts360), release, nil
}

func (targets AuthoringTargets) CaptureBackend(slot string) (string, error) {
	provider, release, err := targets.provider(slot)
	if err != nil {
		return "", err
	}
	defer release()
	desktop, ok := DesktopProfile(provider.profile)
	if !ok {
		return "", errors.New("automation target does not use a desktop capture backend")
	}
	return desktop.CaptureBackend, nil
}

func (targets AuthoringTargets) provider(slot string) (*provider, func(), error) {
	if targets.state == nil {
		return nil, nil, errors.New("authoring targets are unavailable")
	}
	targets.state.mu.RLock()
	generation := targets.state.current
	lease, err := generation.Acquire()
	targets.state.mu.RUnlock()
	if err != nil {
		return nil, nil, err
	}
	provider, err := lease.provider(slot)
	if err != nil {
		lease.Release()
		return nil, nil, fmt.Errorf("automation target slot %q is not installed", slot)
	}
	return provider, lease.Release, nil
}

func providersBySlot(installations Installations) (map[string]*provider, error) {
	if !installations.Valid() {
		return nil, errors.New("authoring targets require installed automation targets")
	}
	bySlot := make(map[string]*provider, len(installations.Entries()))
	for _, installation := range installations.Entries() {
		provider, ok := installation.Provider.(*provider)
		if !ok || provider == nil || !installation.Profile.Valid() || !installation.Manifest.Valid() {
			return nil, fmt.Errorf("automation target slot %q has no trusted host provider", installation.Slot)
		}
		bySlot[installation.Slot] = provider
	}
	return bySlot, nil
}
