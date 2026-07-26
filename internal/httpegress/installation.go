package httpegress

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
	Slot    string
	Profile ProfileDraft
}
type Installation struct {
	Slot             string
	Profile          Profile
	ProviderID       string
	ProviderArtifact artifact.Digest
	TargetID         string
	Provider         resource.Provider
}
type installationState struct {
	entries   []Installation
	providers []resource.Provider
}
type Installations struct{ state *installationState }

func Install(drafts []InstallationDraft) (Installations, error) {
	ordered := append([]InstallationDraft(nil), drafts...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Slot < ordered[j].Slot })
	entries := make([]Installation, 0, len(ordered))
	providers := make([]resource.Provider, 0, len(ordered))
	previous := ""
	shared := map[artifact.Digest]Installation{}
	for _, draft := range ordered {
		if !slotPattern.MatchString(draft.Slot) || draft.Slot <= previous {
			return Installations{}, errors.New("invalid or duplicate HTTP installation slot")
		}
		previous = draft.Slot
		profile, err := SealProfile(draft.Profile)
		if err != nil {
			return Installations{}, fmt.Errorf("seal HTTP profile for slot %q: %w", draft.Slot, err)
		}
		installed, exists := shared[profile.Digest()]
		if !exists {
			provider, err := NewProvider(profile)
			if err != nil {
				return Installations{}, err
			}
			providerID := "http-egress-" + profile.Digest().String()[7:39]
			providerArtifact, err := ProviderArtifactDigest(profile)
			if err != nil {
				return Installations{}, err
			}
			installed = Installation{Profile: profile, ProviderID: providerID, ProviderArtifact: providerArtifact, Provider: provider}
			shared[profile.Digest()] = installed
			providers = append(providers, provider)
		}
		installed.Slot, installed.TargetID = draft.Slot, TargetID(draft.Slot)
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
func (i Installations) CloseIdleConnections() {
	if !i.Valid() {
		return
	}
	for _, provider := range i.state.providers {
		if closer, ok := provider.(interface{ CloseIdleConnections() }); ok {
			closer.CloseIdleConnections()
		}
	}
}
func TargetID(slot string) string { return "http-origin/" + slot }
func ValidateInstallationSlot(slot string) error {
	if !slotPattern.MatchString(slot) {
		return errors.New("invalid HTTP installation slot")
	}
	return nil
}
