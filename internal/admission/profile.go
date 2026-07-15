// Package admission owns Yotta 3.1 host planning and policy admission.
package admission

import (
	"errors"
	"net/url"
	"regexp"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
)

const hostProfileDigestDomain = "yotta/host-profile/v1"

var (
	identityPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	versionedURI    = regexp.MustCompile(`/v[1-9][0-9]*$`)
)

type ProviderCapability struct {
	Capability   capability.Ref `json:"capability"`
	ResourceKind string         `json:"resourceKind"`
}

type ProviderDescriptor struct {
	ID               string               `json:"id"`
	ArtifactDigest   artifact.Digest      `json:"artifactDigest"`
	ABI              string               `json:"abi"`
	PluginInstanceID string               `json:"pluginInstanceId"`
	OperatingSystems []string             `json:"operatingSystems"`
	Architectures    []string             `json:"architectures"`
	HostAPIs         []string             `json:"hostApis"`
	Capabilities     []ProviderCapability `json:"capabilities"`
}

type AutomationTarget struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	ProviderID string `json:"providerId"`
}

type CredentialBinding struct {
	ID         string         `json:"id"`
	ProviderID string         `json:"providerId"`
	Capability capability.Ref `json:"capability"`
}

type HostProfileDraft struct {
	OS                string               `json:"os"`
	Architecture      string               `json:"architecture"`
	HostAPIGeneration string               `json:"hostApiGeneration"`
	Providers         []ProviderDescriptor `json:"providers"`
	Targets           []AutomationTarget   `json:"targets"`
	Credentials       []CredentialBinding  `json:"credentials"`
}

type hostProfileState struct {
	digest      artifact.Digest
	document    HostProfileDraft
	providers   map[string]ProviderDescriptor
	targets     map[string]AutomationTarget
	credentials map[string]CredentialBinding
}

// HostProfile is an immutable, content-addressed snapshot produced by trusted
// host discovery. Workflow Source and node config never construct one.
type HostProfile struct{ state *hostProfileState }

func SealHostProfile(draft HostProfileDraft) (HostProfile, error) {
	if !identityPattern.MatchString(draft.OS) || !identityPattern.MatchString(draft.Architecture) || !identityPattern.MatchString(draft.HostAPIGeneration) ||
		len(draft.Providers) > 1024 || len(draft.Targets) > 16384 || len(draft.Credentials) > 16384 {
		return HostProfile{}, errors.New("invalid host profile identity or budget")
	}
	providers := append([]ProviderDescriptor(nil), draft.Providers...)
	for index := range providers {
		provider := &providers[index]
		provider.OperatingSystems = normalizedIdentities(provider.OperatingSystems)
		provider.Architectures = normalizedIdentities(provider.Architectures)
		provider.HostAPIs = normalizedIdentities(provider.HostAPIs)
		provider.Capabilities = append([]ProviderCapability(nil), provider.Capabilities...)
		sort.Slice(provider.Capabilities, func(i, j int) bool {
			return provider.Capabilities[i].Capability.CapabilityID < provider.Capabilities[j].Capability.CapabilityID
		})
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].ID < providers[j].ID })
	providerIndex := make(map[string]ProviderDescriptor, len(providers))
	for _, provider := range providers {
		if !identityPattern.MatchString(provider.ID) || !provider.ArtifactDigest.Valid() || !validVersionedURI(provider.ABI) ||
			!identityPattern.MatchString(provider.PluginInstanceID) || !validIdentitySet(provider.OperatingSystems) ||
			!validIdentitySet(provider.Architectures) || !validIdentitySet(provider.HostAPIs) || len(provider.Capabilities) == 0 || len(provider.Capabilities) > 256 {
			return HostProfile{}, errors.New("invalid provider descriptor")
		}
		if _, duplicate := providerIndex[provider.ID]; duplicate {
			return HostProfile{}, errors.New("duplicate provider descriptor")
		}
		for index, provided := range provider.Capabilities {
			if provided.Capability.Validate() != nil || !identityPattern.MatchString(provided.ResourceKind) ||
				index > 0 && provider.Capabilities[index-1].Capability.CapabilityID == provided.Capability.CapabilityID {
				return HostProfile{}, errors.New("invalid provider capability descriptor")
			}
		}
		providerIndex[provider.ID] = provider
	}
	targets := append([]AutomationTarget(nil), draft.Targets...)
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	targetIndex := make(map[string]AutomationTarget, len(targets))
	for _, target := range targets {
		if !identityPattern.MatchString(target.ID) || !identityPattern.MatchString(target.Kind) || providerIndex[target.ProviderID].ID == "" {
			return HostProfile{}, errors.New("invalid automation target descriptor")
		}
		if _, duplicate := targetIndex[target.ID]; duplicate {
			return HostProfile{}, errors.New("duplicate automation target")
		}
		targetIndex[target.ID] = target
	}
	credentials := append([]CredentialBinding(nil), draft.Credentials...)
	sort.Slice(credentials, func(i, j int) bool { return credentials[i].ID < credentials[j].ID })
	credentialIndex := make(map[string]CredentialBinding, len(credentials))
	for _, credential := range credentials {
		if !identityPattern.MatchString(credential.ID) || providerIndex[credential.ProviderID].ID == "" || credential.Capability.Validate() != nil {
			return HostProfile{}, errors.New("invalid credential binding metadata")
		}
		if _, duplicate := credentialIndex[credential.ID]; duplicate {
			return HostProfile{}, errors.New("duplicate credential binding metadata")
		}
		credentialIndex[credential.ID] = credential
	}
	document := HostProfileDraft{
		OS: draft.OS, Architecture: draft.Architecture, HostAPIGeneration: draft.HostAPIGeneration,
		Providers: providers, Targets: targets, Credentials: credentials,
	}
	canonical, err := artifact.Marshal(document)
	if err != nil {
		return HostProfile{}, err
	}
	digest, err := artifact.Sum(hostProfileDigestDomain, canonical)
	if err != nil {
		return HostProfile{}, err
	}
	return HostProfile{state: &hostProfileState{digest: digest, document: document, providers: providerIndex, targets: targetIndex, credentials: credentialIndex}}, nil
}

func validVersionedURI(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.IsAbs() && parsed.Fragment == "" && versionedURI.MatchString(parsed.Path)
}

func normalizedIdentities(source []string) []string {
	result := append([]string(nil), source...)
	sort.Strings(result)
	return result
}

func validIdentitySet(values []string) bool {
	if len(values) == 0 || len(values) > 64 {
		return false
	}
	for index, value := range values {
		if !identityPattern.MatchString(value) || index > 0 && values[index-1] == value {
			return false
		}
	}
	return true
}

func (p HostProfile) Valid() bool { return p.state != nil && p.state.digest.Valid() }
func (p HostProfile) Digest() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.digest
}
