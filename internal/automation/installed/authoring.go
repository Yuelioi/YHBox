package installed

import (
	"context"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/target"
)

// AuthoringTargets is the host-only projection used by trusted local tooling.
// Workflow execution continues to use Resource Broker grants and never receives
// this surface or a native window handle.
type AuthoringTargets struct {
	bySlot map[string]*provider
}

func NewAuthoringTargets(installations Installations) (AuthoringTargets, error) {
	if !installations.Valid() {
		return AuthoringTargets{}, errors.New("authoring targets require installed automation targets")
	}
	bySlot := make(map[string]*provider, len(installations.Entries()))
	for _, installation := range installations.Entries() {
		provider, ok := installation.Provider.(*provider)
		if !ok || provider == nil || !installation.Profile.Valid() {
			return AuthoringTargets{}, fmt.Errorf("automation target slot %q has no trusted host provider", installation.Slot)
		}
		bySlot[installation.Slot] = provider
	}
	return AuthoringTargets{bySlot: bySlot}, nil
}

func (targets AuthoringTargets) ResolveWindow(ctx context.Context, slot string) (target.WindowHandle, error) {
	provider, err := targets.provider(slot)
	if err != nil {
		return target.WindowHandle{}, err
	}
	if ctx == nil {
		return target.WindowHandle{}, errors.New("resolve automation target context is required")
	}
	if err := VerifyProfile(provider.profile); err != nil {
		return target.WindowHandle{}, failure(CodeIdentityChanged, err)
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
	provider, err := targets.provider(slot)
	if err != nil {
		return target.Target{}, err
	}
	if ctx == nil {
		return target.Target{}, errors.New("resolve automation target context is required")
	}
	if err := VerifyProfile(provider.profile); err != nil {
		return target.Target{}, failure(CodeIdentityChanged, err)
	}
	return provider.driver.ResolveTarget(ctx)
}

func (targets AuthoringTargets) CapturePNG(ctx context.Context, slot string) ([]byte, error) {
	provider, err := targets.provider(slot)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, errors.New("capture automation target context is required")
	}
	if err := VerifyProfile(provider.profile); err != nil {
		return nil, failure(CodeIdentityChanged, err)
	}
	return provider.driver.Capture(ctx)
}

func (targets AuthoringTargets) Activate(ctx context.Context, slot string) error {
	provider, err := targets.provider(slot)
	if err != nil {
		return err
	}
	if ctx == nil {
		return errors.New("activate automation target context is required")
	}
	if err := VerifyProfile(provider.profile); err != nil {
		return failure(CodeIdentityChanged, err)
	}
	return provider.driver.Execute(ctx, OperationActivate, struct{}{})
}

func (targets AuthoringTargets) CaptureBackend(slot string) (string, error) {
	provider, err := targets.provider(slot)
	if err != nil {
		return "", err
	}
	return provider.profile.Machine().CaptureBackend, nil
}

func (targets AuthoringTargets) provider(slot string) (*provider, error) {
	if targets.bySlot == nil {
		return nil, errors.New("authoring targets are unavailable")
	}
	provider := targets.bySlot[slot]
	if provider == nil {
		return nil, fmt.Errorf("automation target slot %q is not installed", slot)
	}
	return provider, nil
}
