package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/admission"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/targetruntime"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

type leasedAdmission struct {
	record    run.Record
	providers map[string]run.InstalledProvider
	targets   targetruntime.Snapshot
	release   func()
}

func (a *Application) replaceAdmissionEnvironment(
	profile admission.HostProfile,
	policy admission.Policy,
	providers map[string]run.InstalledProvider,
	targets func() (targetruntime.Snapshot, func(), error),
) error {
	if !profile.Valid() || policy == nil {
		return errors.New("replacement execution environment is invalid")
	}
	next, err := cloneProviderSnapshot(providers)
	if err != nil {
		return err
	}
	a.admissionMu.Lock()
	defer a.admissionMu.Unlock()
	if err := a.admitter.ReplaceEnvironment(profile, policy); err != nil {
		return err
	}
	a.providers = next
	a.targetSnapshot = targets
	return nil
}

func (a *Application) admitRun(
	ctx context.Context,
	program compiler.ProgramSnapshot,
	principal string,
	selection admission.Selection,
) (_ leasedAdmission, resultErr error) {
	if !program.Valid() {
		return leasedAdmission{}, errors.New("run admission requires a valid Program")
	}
	if err := a.programs.Put(ctx, program); err != nil {
		return leasedAdmission{}, fmt.Errorf("persist Program before admission: %w", err)
	}

	a.admissionMu.RLock()
	defer a.admissionMu.RUnlock()
	release := func() {}
	targets, err := targetruntime.NewSnapshot(nil)
	if err != nil {
		return leasedAdmission{}, fmt.Errorf("create empty configured target snapshot: %w", err)
	}
	if a.targetSnapshot != nil {
		targets, release, err = a.targetSnapshot()
		if err != nil {
			return leasedAdmission{}, fmt.Errorf("snapshot configured targets: %w", err)
		}
		if !targets.Valid() || release == nil {
			return leasedAdmission{}, errors.New("execution environment returned an invalid configured target snapshot")
		}
	}
	defer func() {
		if resultErr != nil {
			release()
		}
	}()
	admitted, err := a.admitter.Admit(ctx, admission.Request{
		Program: program, Principal: principal, Selection: selection,
	})
	if err != nil {
		return leasedAdmission{record: admitted.Record}, err
	}
	return leasedAdmission{record: admitted.Record, providers: a.providers, targets: targets, release: release}, nil
}

func cloneProviderSnapshot(source map[string]run.InstalledProvider) (map[string]run.InstalledProvider, error) {
	if source == nil {
		return nil, errors.New("execution environment providers are unavailable")
	}
	result := make(map[string]run.InstalledProvider, len(source))
	for id, provider := range source {
		if id == "" || !provider.ArtifactDigest.Valid() || provider.ABI == "" || provider.Provider == nil {
			return nil, errors.New("execution environment contains an invalid provider")
		}
		result[id] = provider
	}
	return result, nil
}
