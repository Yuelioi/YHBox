package appbootstrap

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
)

type executionEnvironmentFactory struct {
	builtins            nodes.Builtins
	blobDigest          artifact.Digest
	streamDigest        artifact.Digest
	workspaceFileDigest artifact.Digest
	scriptRuntime       *scriptengine.Runtime
	pluginFeatures      []string
	now                 func() time.Time
	grantTTL            time.Duration
	ai                  ai.Installations
	http                httpegress.Installations
	baseProviders       map[string]run.InstalledProvider
}

// executionEnvironmentFactory is the single projection seam for installed
// target facts. Each call seals a profile, policy, provider snapshot, and
// automation generation that must be published together.
type sealedExecutionEnvironment struct {
	profile    admission.HostProfile
	policy     admission.Policy
	providers  map[string]run.InstalledProvider
	generation automationinstalled.Generation
}

func newExecutionEnvironmentFactory(config executionEnvironmentFactory) (executionEnvironmentFactory, error) {
	if !config.blobDigest.Valid() || !config.streamDigest.Valid() ||
		!config.workspaceFileDigest.Valid() || config.scriptRuntime == nil ||
		config.now == nil || config.grantTTL <= 0 || config.grantTTL > 24*time.Hour ||
		!config.ai.Valid() || !config.http.Valid() || config.baseProviders == nil {
		return executionEnvironmentFactory{}, errors.New("execution environment factory requires trusted fixed installations")
	}
	baseProviders := maps.Clone(config.baseProviders)
	if err := mergeEnvironmentInstallations(baseProviders, config.ai.Entries(), ai.ProviderABI, "AI",
		func(installed ai.Installation) (string, artifact.Digest, resource.Provider) {
			return installed.ProviderID, installed.ProviderArtifact, installed.Provider
		}); err != nil {
		return executionEnvironmentFactory{}, err
	}
	if err := mergeEnvironmentInstallations(baseProviders, config.http.Entries(), httpegress.ProviderABI, "HTTP",
		func(installed httpegress.Installation) (string, artifact.Digest, resource.Provider) {
			return installed.ProviderID, installed.ProviderArtifact, installed.Provider
		}); err != nil {
		return executionEnvironmentFactory{}, err
	}
	config.baseProviders = baseProviders
	config.pluginFeatures = append([]string(nil), config.pluginFeatures...)
	return config, nil
}

func (factory executionEnvironmentFactory) seal(
	applications appcontrol.Installations,
	automation automationinstalled.Installations,
) (_ *sealedExecutionEnvironment, resultErr error) {
	config := factory
	if !applications.Valid() || !automation.Valid() || config.baseProviders == nil {
		return nil, errors.New("execution environment installations are unavailable")
	}
	generation, err := automationinstalled.NewGeneration(automation)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, generation.Retire())
		}
	}()
	profile, err := buildHostProfile(
		config.builtins,
		config.blobDigest,
		config.streamDigest,
		config.workspaceFileDigest,
		config.scriptRuntime,
		config.pluginFeatures,
		config.ai,
		config.http,
		applications,
		automation,
	)
	if err != nil {
		return nil, err
	}
	policy, err := NewBuiltinPolicy(
		config.now,
		config.grantTTL,
		config.ai,
		config.http,
		applications,
		automation,
	)
	if err != nil {
		return nil, err
	}
	providers := maps.Clone(config.baseProviders)
	if err := mergeEnvironmentInstallations(providers, applications.Entries(), appcontrol.ProviderABI, "application",
		func(installed appcontrol.Installation) (string, artifact.Digest, resource.Provider) {
			return installed.ProviderID, installed.ProviderArtifact, installed.Provider
		}); err != nil {
		return nil, err
	}
	if err := mergeEnvironmentInstallations(providers, automation.Entries(), automationinstalled.ProviderABI, "automation",
		func(installed automationinstalled.Installation) (string, artifact.Digest, resource.Provider) {
			return installed.ProviderID, installed.ProviderArtifact, installed.Provider
		}); err != nil {
		return nil, err
	}
	return &sealedExecutionEnvironment{
		profile: profile, policy: policy, providers: providers, generation: generation,
	}, nil
}

func mergeEnvironmentInstallations[T any](
	providers map[string]run.InstalledProvider,
	installations []T,
	abi, kind string,
	project func(T) (string, artifact.Digest, resource.Provider),
) error {
	for _, installed := range installations {
		id, digest, provider := project(installed)
		if id == "" || !digest.Valid() || abi == "" || provider == nil {
			return fmt.Errorf("invalid %s provider installation", kind)
		}
		if existing, ok := providers[id]; ok {
			if existing.ArtifactDigest != digest || existing.ABI != abi || existing.Provider != provider {
				return fmt.Errorf("conflicting %s provider installation", kind)
			}
			continue
		}
		providers[id] = run.InstalledProvider{ArtifactDigest: digest, ABI: abi, Provider: provider}
	}
	return nil
}

func (environment *sealedExecutionEnvironment) acquire() (func(), error) {
	if environment == nil {
		return nil, errors.New("execution environment is unavailable")
	}
	lease, err := environment.generation.Acquire()
	if err != nil {
		return nil, err
	}
	return lease.Release, nil
}
