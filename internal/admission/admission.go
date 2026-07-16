package admission

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const (
	CodeTargetUnavailable      = "admission.target_unavailable"
	CodeTargetAmbiguous        = "admission.target_ambiguous"
	CodeProviderIncompatible   = "admission.provider_incompatible"
	CodeUnsupportedHost        = "admission.unsupported_host"
	CodeCredentialUnavailable  = "admission.credential_unavailable"
	CodeCredentialAmbiguous    = "admission.credential_ambiguous"
	CodeConsentRequired        = "admission.consent_required"
	CodePolicyDenied           = "admission.policy_denied"
	CodePolicyInvalid          = "admission.policy_invalid"
	CodePersistenceFailed      = "admission.persistence_failed"
	CodePersistenceUnconfirmed = "admission.persistence_unconfirmed"
)

type Error struct {
	Code          string
	GraphID       string
	NodeID        string
	RequirementID string
	Slot          string
	Commit        run.CommitOutcome
	Cause         error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Code
}
func (e *Error) Unwrap() error { return e.Cause }

type Selection struct {
	Targets     map[string]string
	Credentials map[string]string
}

type Request struct {
	Program   compiler.ProgramSnapshot
	Principal string
	Selection Selection
}

type Result struct {
	Grant  capability.RunGrant
	Record run.Record
}

type PolicyOutcome string

const (
	PolicyApproved        PolicyOutcome = "approved"
	PolicyDenied          PolicyOutcome = "denied"
	PolicyConsentRequired PolicyOutcome = "consent-required"
)

type PolicyRequest struct {
	ProgramHash  artifact.Digest
	PlanDigest   artifact.Digest
	RunID        string
	Principal    string
	HostProfile  artifact.Digest
	Bindings     []capability.Binding
	Requirements []capability.PlanEntry
}

type PolicyDecision struct {
	Outcome        PolicyOutcome
	Generation     string
	ExpiresAt      time.Time
	ConsentLineage []artifact.Digest
}

type Policy interface {
	Authorize(context.Context, PolicyRequest) (PolicyDecision, error)
}

type PolicyFunc func(context.Context, PolicyRequest) (PolicyDecision, error)

func (f PolicyFunc) Authorize(ctx context.Context, request PolicyRequest) (PolicyDecision, error) {
	return f(ctx, request)
}

type recordCreator interface {
	Create(context.Context, run.Record) (run.CommitOutcome, error)
}

type Options struct {
	Now         func() time.Time
	NewRunID    func() (string, error)
	MaxGrantTTL time.Duration
}

type Admitter struct {
	catalog nodecatalog.Snapshot
	profile HostProfile
	store   recordCreator
	policy  Policy
	now     func() time.Time
	newID   func() (string, error)
	maxTTL  time.Duration
}

func New(catalog nodecatalog.Snapshot, profile HostProfile, store recordCreator, policy Policy, options Options) (*Admitter, error) {
	if !catalog.Valid() || !profile.Valid() || store == nil || policy == nil {
		return nil, errors.New("admitter requires trusted Catalog, Host Profile, Run Store, and Policy")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.NewRunID == nil {
		options.NewRunID = runid.New
	}
	if options.MaxGrantTTL <= 0 || options.MaxGrantTTL > 24*time.Hour {
		return nil, errors.New("admitter requires a bounded grant TTL")
	}
	return &Admitter{catalog: catalog, profile: profile, store: store, policy: policy, now: options.Now, newID: options.NewRunID, maxTTL: options.MaxGrantTTL}, nil
}

func (a *Admitter) Admit(ctx context.Context, request Request) (Result, error) {
	if ctx == nil || !request.Program.Valid() || !identityPattern.MatchString(request.Principal) {
		return Result{}, &Error{Code: CodePolicyInvalid}
	}
	if request.Program.CatalogHash() != a.catalog.Hash() {
		return Result{}, &Error{Code: CodeProviderIncompatible}
	}
	if err := a.requireHostFeatures(request.Program.Nodes()); err != nil {
		return Result{}, err
	}
	runID, err := a.newID()
	if err != nil || runid.Validate(runID) != nil {
		return Result{}, &Error{Code: CodePolicyInvalid, Cause: err}
	}
	bindings, err := a.plan(request.Program.CapabilityPlan(), runID, request.Selection)
	if err != nil {
		return Result{}, err
	}
	policyRequest := PolicyRequest{
		ProgramHash: request.Program.Hash(), PlanDigest: request.Program.CapabilityPlan().Digest(), RunID: runID,
		Principal: request.Principal, HostProfile: a.profile.Digest(), Bindings: append([]capability.Binding(nil), bindings...),
		Requirements: request.Program.CapabilityPlan().Entries(),
	}
	decision, err := a.policy.Authorize(ctx, policyRequest)
	if err != nil {
		return Result{}, &Error{Code: CodePolicyDenied, Cause: err}
	}
	switch decision.Outcome {
	case PolicyDenied:
		return Result{}, &Error{Code: CodePolicyDenied}
	case PolicyConsentRequired:
		return Result{}, &Error{Code: CodeConsentRequired}
	case PolicyApproved:
	default:
		return Result{}, &Error{Code: CodePolicyInvalid}
	}
	now := a.now().UTC()
	if decision.ExpiresAt.Location() != time.UTC || !decision.ExpiresAt.After(now) || decision.ExpiresAt.After(now.Add(a.maxTTL)) {
		return Result{}, &Error{Code: CodePolicyInvalid}
	}
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: request.Program.Hash(), Plan: request.Program.CapabilityPlan(), RunID: runID, Principal: request.Principal,
		PolicyGeneration: decision.Generation, IssuedAt: now, ExpiresAt: decision.ExpiresAt,
		Bindings: bindings, ConsentLineage: decision.ConsentLineage,
	}, a.catalog)
	if err != nil {
		return Result{}, &Error{Code: CodePolicyInvalid, Cause: err}
	}
	record, err := run.NewQueuedRecord(run.QueueRequest{
		ProgramHash: request.Program.Hash(), CatalogHash: a.catalog.Hash(), CapabilityPlanDigest: request.Program.CapabilityPlan().Digest(),
		Grant: grant, QueuedAt: now,
	})
	if err != nil {
		return Result{}, &Error{Code: CodePolicyInvalid, Cause: err}
	}
	commit, err := a.store.Create(ctx, record)
	if err != nil {
		if commit == run.CommitPublished {
			return Result{Grant: grant, Record: record}, &Error{Code: CodePersistenceUnconfirmed, Commit: commit, Cause: err}
		}
		return Result{}, &Error{Code: CodePersistenceFailed, Cause: err}
	}
	if commit != run.CommitDurable {
		return Result{}, &Error{Code: CodePersistenceFailed, Commit: commit, Cause: errors.New("run store returned an invalid commit outcome")}
	}
	return Result{Grant: grant, Record: record}, nil
}

func (a *Admitter) requireHostFeatures(nodes []compiler.NodeView) error {
	available := make(map[string]struct{}, len(a.profile.state.document.Features))
	for _, feature := range a.profile.state.document.Features {
		available[feature] = struct{}{}
	}
	for _, node := range nodes {
		for _, requirement := range node.HostFeatures {
			if _, ok := available[requirement.FeatureID]; !ok {
				return &Error{
					Code: CodeUnsupportedHost, GraphID: node.GraphID, NodeID: node.ID,
					RequirementID: requirement.ID,
				}
			}
		}
	}
	return nil
}

type targetCandidate struct {
	target   AutomationTarget
	provider ProviderDescriptor
}

type plannedRequirement struct {
	entry       capability.PlanEntry
	definition  capability.Definition
	candidates  map[string]targetCandidate
	mismatch    bool
	unsupported bool
}

func (a *Admitter) plan(plan capability.Plan, runID string, selection Selection) ([]capability.Binding, error) {
	entries := plan.Entries()
	planned := make([]plannedRequirement, 0, len(entries))
	bySlot := map[string][]int{}
	for _, entry := range entries {
		definition, ok := a.catalog.LookupCapability(entry.Requirement.Capability.CapabilityID)
		if !ok || definition.Ref() != entry.Requirement.Capability {
			return nil, admissionError(CodeProviderIncompatible, entry, entry.Requirement.TargetSlot)
		}
		item := plannedRequirement{entry: entry, definition: definition, candidates: map[string]targetCandidate{}}
		for _, target := range a.profile.state.document.Targets {
			provider := a.profile.state.providers[target.ProviderID]
			provided, exact, related := providerCapability(provider, entry.Requirement.Capability)
			if related && (!exact || provider.ABI != definition.Machine().ProviderABI) {
				item.mismatch = true
				continue
			}
			if exact && provider.ABI == definition.Machine().ProviderABI && !providerSupportsHost(provider, a.profile.state.document) {
				item.unsupported = true
				continue
			}
			if !exact || provider.ABI != definition.Machine().ProviderABI || !contains(definition.Machine().TargetKinds, target.Kind) {
				continue
			}
			_ = provided
			key := provider.ID + "\x00" + target.ID
			item.candidates[key] = targetCandidate{target: target, provider: provider}
		}
		planned = append(planned, item)
		bySlot[entry.Requirement.TargetSlot] = append(bySlot[entry.Requirement.TargetSlot], len(planned)-1)
	}
	chosen := map[string]targetCandidate{}
	for slot, indexes := range bySlot {
		intersection := map[string]targetCandidate{}
		for key, candidate := range planned[indexes[0]].candidates {
			intersection[key] = candidate
		}
		mismatch := false
		unsupported := false
		for _, index := range indexes {
			mismatch = mismatch || planned[index].mismatch
			unsupported = unsupported || planned[index].unsupported
			for key := range intersection {
				if _, ok := planned[index].candidates[key]; !ok {
					delete(intersection, key)
				}
			}
		}
		selected := selection.Targets[slot]
		if selected == "" {
			selected = a.profile.state.targetSlots[slot]
		}
		if selected != "" {
			for key, candidate := range intersection {
				if candidate.target.ID != selected {
					delete(intersection, key)
				}
			}
		}
		if len(intersection) == 0 {
			code := CodeTargetUnavailable
			if unsupported {
				code = CodeUnsupportedHost
			} else if mismatch {
				code = CodeProviderIncompatible
			}
			return nil, admissionError(code, planned[indexes[0]].entry, slot)
		}
		if len(intersection) != 1 {
			return nil, admissionError(CodeTargetAmbiguous, planned[indexes[0]].entry, slot)
		}
		for _, candidate := range intersection {
			chosen[slot] = candidate
		}
	}
	for slot := range selection.Targets {
		if _, ok := bySlot[slot]; !ok {
			return nil, &Error{Code: CodeTargetUnavailable, Slot: slot}
		}
	}
	credentialBySlot, err := a.planCredentials(planned, chosen, selection.Credentials)
	if err != nil {
		return nil, err
	}
	bindings := make([]capability.Binding, 0, len(planned))
	for _, item := range planned {
		candidate := chosen[item.entry.Requirement.TargetSlot]
		provided, _, _ := providerCapability(candidate.provider, item.entry.Requirement.Capability)
		sessionID, err := sessionIdentity(runID, item.entry.Requirement.TargetSlot, candidate)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, capability.Binding{
			GraphID: item.entry.GraphID, NodeID: item.entry.NodeID, RequirementID: item.entry.Requirement.ID,
			ProviderID: candidate.provider.ID, ProviderArtifactDigest: candidate.provider.ArtifactDigest, ProviderABI: candidate.provider.ABI,
			TargetID: candidate.target.ID, TargetKind: candidate.target.Kind, ResourceKind: provided.ResourceKind,
			PluginInstanceID: candidate.provider.PluginInstanceID, SessionID: sessionID,
			CredentialBindingID: credentialBySlot[item.entry.Requirement.CredentialSlot],
		})
	}
	return bindings, nil
}

func providerSupportsHost(provider ProviderDescriptor, profile HostProfileDraft) bool {
	return contains(provider.OperatingSystems, profile.OS) && contains(provider.Architectures, profile.Architecture) && contains(provider.HostAPIs, profile.HostAPIGeneration)
}

func (a *Admitter) planCredentials(planned []plannedRequirement, chosen map[string]targetCandidate, selections map[string]string) (map[string]string, error) {
	bySlot := map[string][]plannedRequirement{}
	for _, item := range planned {
		if item.entry.Requirement.CredentialSlot != "" {
			bySlot[item.entry.Requirement.CredentialSlot] = append(bySlot[item.entry.Requirement.CredentialSlot], item)
		}
	}
	result := map[string]string{}
	for slot, items := range bySlot {
		candidates := map[string]struct{}{}
		for id := range a.profile.state.credentials {
			candidates[id] = struct{}{}
		}
		for _, item := range items {
			provider := chosen[item.entry.Requirement.TargetSlot].provider
			for id := range candidates {
				credential := a.profile.state.credentials[id]
				if credential.ProviderID != provider.ID || credential.Capability != item.entry.Requirement.Capability {
					delete(candidates, id)
				}
			}
		}
		selected := selections[slot]
		if selected == "" {
			selected = a.profile.state.credentialSlots[slot]
		}
		if selected != "" {
			for id := range candidates {
				if id != selected {
					delete(candidates, id)
				}
			}
		}
		if len(candidates) == 0 {
			return nil, admissionError(CodeCredentialUnavailable, items[0].entry, slot)
		}
		if len(candidates) != 1 {
			return nil, admissionError(CodeCredentialAmbiguous, items[0].entry, slot)
		}
		for id := range candidates {
			result[slot] = id
		}
	}
	for slot := range selections {
		if _, ok := bySlot[slot]; !ok {
			return nil, &Error{Code: CodeCredentialUnavailable, Slot: slot}
		}
	}
	return result, nil
}

func providerCapability(provider ProviderDescriptor, wanted capability.Ref) (ProviderCapability, bool, bool) {
	for _, provided := range provider.Capabilities {
		if provided.Capability.CapabilityID == wanted.CapabilityID {
			return provided, provided.Capability == wanted, true
		}
	}
	return ProviderCapability{}, false, false
}

func sessionIdentity(runID, targetSlot string, candidate targetCandidate) (string, error) {
	raw, err := artifact.Marshal(struct {
		RunID      string `json:"runId"`
		TargetSlot string `json:"targetSlot"`
		ProviderID string `json:"providerId"`
		TargetID   string `json:"targetId"`
	}{RunID: runID, TargetSlot: targetSlot, ProviderID: candidate.provider.ID, TargetID: candidate.target.ID})
	if err != nil {
		return "", err
	}
	digest, err := artifact.Sum("yotta/run-session-id/v1", raw)
	return string(digest), err
}

func admissionError(code string, entry capability.PlanEntry, slot string) *Error {
	return &Error{Code: code, GraphID: entry.GraphID, NodeID: entry.NodeID, RequirementID: entry.Requirement.ID, Slot: slot}
}

func contains(values []string, wanted string) bool {
	index := sort.SearchStrings(values, wanted)
	return index < len(values) && values[index] == wanted
}
