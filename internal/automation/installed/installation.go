package installed

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

var slotPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)

type InstallationDraft struct {
	Slot                  string
	Label                 string
	Profile               ProfileDraft
	RuntimeMouseCounts360 int64
}

type Installation struct {
	Slot             string
	Profile          Profile
	Manifest         InstallationManifest
	ProviderID       string
	ProviderArtifact artifact.Digest
	TargetID         string
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
		manifest, err := sealInstallationManifestForProfile(draft.Slot, draft.Label, profile, registry)
		if err != nil {
			return Installations{}, err
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
		installed.Slot, installed.TargetID = draft.Slot, TargetID(draft.Slot)
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
func sealInstallationManifestForProfile(slot, label string, profile Profile, registry adapterRegistry) (InstallationManifest, error) {
	if !slotPattern.MatchString(slot) || !profile.Valid() {
		return InstallationManifest{}, errors.New("automation target configuration is invalid")
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
		providerArtifact, profile, registered,
	)
	if err != nil {
		return InstallationManifest{}, fmt.Errorf("seal automation target manifest for slot %q: %w", slot, err)
	}
	return manifest, nil
}
