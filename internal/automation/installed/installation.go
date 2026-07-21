package installed

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

const workflowConsentDomain = "yotta/installed-automation-target-consent/v1"

var slotPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)

type InstallationDraft struct {
	Slot                  string
	Label                 string
	Profile               ProfileDraft
	Consent               artifact.Digest
	RuntimeMouseCounts360 int64
}

type Installation struct {
	Slot             string
	Profile          Profile
	Manifest         InstallationManifest
	ProviderID       string
	ProviderArtifact artifact.Digest
	TargetID         string
	Consent          artifact.Digest
	Provider         resource.Provider
	Descriptor       InstallationDescriptor
}

type installationState struct {
	entries   []Installation
	providers []*provider
}
type Installations struct{ state *installationState }

func Install(drafts []InstallationDraft) (result Installations, resultErr error) {
	return installWithRegistry(drafts, defaultAdapterRegistry())
}

func installWithRegistry(drafts []InstallationDraft, registry adapterRegistry) (result Installations, resultErr error) {
	ordered := append([]InstallationDraft(nil), drafts...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Slot < ordered[j].Slot })
	entries := make([]Installation, 0, len(ordered))
	providers := make([]*provider, 0, len(ordered))
	defer func() {
		if resultErr == nil {
			return
		}
		for _, created := range providers {
			resultErr = errors.Join(resultErr, created.CloseHost())
		}
	}()
	shared := map[artifact.Digest]Installation{}
	previous := ""
	for _, draft := range ordered {
		if !slotPattern.MatchString(draft.Slot) || draft.Slot <= previous {
			return Installations{}, errors.New("invalid or duplicate automation target installation slot")
		}
		previous = draft.Slot
		profile, err := sealProfileWithRegistry(draft.Profile, registry)
		if err != nil {
			return Installations{}, fmt.Errorf("seal automation target profile for slot %q: %w", draft.Slot, err)
		}
		if err := verifyProfileWithRegistry(profile, registry); err != nil {
			return Installations{}, fmt.Errorf("verify automation target profile for slot %q: %w", draft.Slot, err)
		}
		manifest, err := sealInstallationManifestForProfile(draft.Slot, draft.Label, profile, draft.Consent != "", registry)
		if err != nil {
			return Installations{}, err
		}
		expected, err := workflowConsentDigest(manifest)
		if err != nil {
			return Installations{}, err
		}
		if draft.Consent != "" && draft.Consent != expected {
			return Installations{}, fmt.Errorf("automation target slot %q has stale workflow consent", draft.Slot)
		}
		installed, exists := shared[profile.Digest()]
		if !exists {
			created, err := newProvider(profile, manifest, registry)
			if err != nil {
				return Installations{}, err
			}
			created.runtimeMouseCounts360 = draft.RuntimeMouseCounts360
			document := manifest.Machine()
			installed = Installation{
				Profile: profile, ProviderID: document.ProviderID,
				ProviderArtifact: document.ProviderArtifact, Provider: created,
			}
			shared[profile.Digest()] = installed
			providers = append(providers, created)
		}
		installed.Slot, installed.TargetID, installed.Consent = draft.Slot, TargetID(draft.Slot), draft.Consent
		installed.Manifest = manifest
		installed.Descriptor = manifest.Descriptor()
		entries = append(entries, installed)
	}
	return Installations{state: &installationState{entries: entries, providers: providers}}, nil
}

func (i Installations) Valid() bool { return i.state != nil }
func (i Installations) Entries() []Installation {
	if !i.Valid() {
		return nil
	}
	return append([]Installation(nil), i.state.entries...)
}
func (i Installations) Close() error {
	if !i.Valid() {
		return nil
	}
	var result error
	for _, installed := range i.state.providers {
		result = errors.Join(result, installed.CloseHost())
	}
	return result
}

func TargetID(slot string) string { return "automation-target/" + slot }
func ValidateInstallationSlot(slot string) error {
	if !slotPattern.MatchString(slot) {
		return errors.New("invalid automation target installation slot")
	}
	return nil
}
func WorkflowConsentDigest(slot string, profile Profile) (artifact.Digest, error) {
	manifest, err := sealInstallationManifestForProfile(slot, "", profile, false, defaultAdapterRegistry())
	if err != nil {
		return "", err
	}
	return workflowConsentDigest(manifest)
}

func sealInstallationManifestForProfile(slot, label string, profile Profile, consented bool, registry adapterRegistry) (InstallationManifest, error) {
	if !slotPattern.MatchString(slot) || !profile.Valid() {
		return InstallationManifest{}, errors.New("automation target consent identity is invalid")
	}
	registered, err := registry.registration(profile)
	if err != nil {
		return InstallationManifest{}, err
	}
	providerArtifact, err := ProviderArtifactDigest(profile)
	if err != nil {
		return InstallationManifest{}, err
	}
	manifest, err := sealInstallationManifest(
		slot, label, TargetID(slot), "automation-"+profile.Digest().String()[7:39],
		providerArtifact, profile, registered, consented,
	)
	if err != nil {
		return InstallationManifest{}, fmt.Errorf("seal automation target manifest for slot %q: %w", slot, err)
	}
	return manifest, nil
}

func workflowConsentDigest(manifest InstallationManifest) (artifact.Digest, error) {
	if !manifest.Valid() {
		return "", errors.New("automation target consent manifest is invalid")
	}
	document := manifest.Machine()
	raw, err := artifact.Marshal(map[string]any{
		"slot": document.Slot, "profileDigest": document.ProfileDigest, "providerAbi": document.ProviderABI,
		"capabilities": document.Capabilities,
	})
	if err != nil {
		return "", err
	}
	return artifact.Sum(workflowConsentDomain, raw)
}
