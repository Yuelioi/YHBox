package appbootstrap

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

type builtinPolicy struct {
	now                 func() time.Time
	ttl                 time.Duration
	blobDigest          artifact.Digest
	streamDigest        artifact.Digest
	workspaceFileDigest artifact.Digest
	aiTargets           map[string]ai.Installation
	httpTargets         map[string]httpegress.Installation
	applicationTargets  map[string]appcontrol.Installation
	automationTargets   map[string]automationinstalled.Installation
}

// NewBuiltinPolicy creates the explicit local policy for code compiled into
// Yotta. It cannot authorize third-party providers, credentials, or remote
// targets; plugin policies enter through their own admission owner.
func NewBuiltinPolicy(now func() time.Time, ttl time.Duration, aiInstallations ai.Installations, httpInstallations httpegress.Installations, applicationInstallations appcontrol.Installations, automationInstallations automationinstalled.Installations) (admission.Policy, error) {
	if now == nil || ttl <= 0 || ttl > 24*time.Hour || !aiInstallations.Valid() || !httpInstallations.Valid() || !applicationInstallations.Valid() || !automationInstallations.Valid() {
		return nil, errors.New("built-in policy requires clock and bounded TTL")
	}
	blobDigest, err := blob.ProviderArtifactDigest()
	if err != nil {
		return nil, err
	}
	streamDigest, err := stream.ProviderArtifactDigest()
	if err != nil {
		return nil, err
	}
	workspaceFileDigest, err := workspacefs.ProviderArtifactDigest()
	if err != nil {
		return nil, err
	}
	aiTargets := make(map[string]ai.Installation, len(aiInstallations.Entries()))
	for _, installed := range aiInstallations.Entries() {
		if _, duplicate := aiTargets[installed.TargetID]; duplicate {
			return nil, errors.New("built-in policy received duplicate AI targets")
		}
		aiTargets[installed.TargetID] = installed
	}
	httpTargets := make(map[string]httpegress.Installation, len(httpInstallations.Entries()))
	for _, installed := range httpInstallations.Entries() {
		if _, duplicate := httpTargets[installed.TargetID]; duplicate {
			return nil, errors.New("built-in policy received duplicate HTTP targets")
		}
		httpTargets[installed.TargetID] = installed
	}
	applicationTargets := make(map[string]appcontrol.Installation, len(applicationInstallations.Entries()))
	for _, installed := range applicationInstallations.Entries() {
		if _, duplicate := applicationTargets[installed.TargetID]; duplicate {
			return nil, errors.New("built-in policy received duplicate application targets")
		}
		applicationTargets[installed.TargetID] = installed
	}
	automationTargets := make(map[string]automationinstalled.Installation, len(automationInstallations.Entries()))
	for _, installed := range automationInstallations.Entries() {
		if _, duplicate := automationTargets[installed.TargetID]; duplicate {
			return nil, errors.New("built-in policy received duplicate installed automation targets")
		}
		automationTargets[installed.TargetID] = installed
	}
	return &builtinPolicy{now: now, ttl: ttl, blobDigest: blobDigest, streamDigest: streamDigest, workspaceFileDigest: workspaceFileDigest, aiTargets: aiTargets, httpTargets: httpTargets, applicationTargets: applicationTargets, automationTargets: automationTargets}, nil
}

func (p *builtinPolicy) Authorize(_ context.Context, request admission.PolicyRequest) (admission.PolicyDecision, error) {
	consents := make(map[artifact.Digest]struct{})
	for _, binding := range request.Bindings {
		if binding.PluginInstanceID != "builtin" {
			return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
		}
		switch binding.ProviderID {
		case blob.ProviderID:
			if binding.ProviderArtifactDigest != p.blobDigest || binding.ProviderABI != blob.ProviderABI ||
				binding.TargetID != "workspace" || binding.TargetKind != "blob-store" || binding.ResourceKind == "" || binding.CredentialBindingID != "" {
				return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
			}
		case stream.ProviderID:
			if binding.ProviderArtifactDigest != p.streamDigest || binding.ProviderABI != stream.ProviderABI ||
				binding.TargetID != "memory" || binding.TargetKind != "stream-session" || binding.ResourceKind != stream.Kind || binding.CredentialBindingID != "" {
				return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
			}
		case workspacefs.ProviderID:
			if binding.ProviderArtifactDigest != p.workspaceFileDigest || binding.ProviderABI != workspacefs.ProviderABI ||
				binding.TargetID != workspacefs.TargetID || binding.TargetKind != workspacefs.TargetKind || binding.ResourceKind != workspacefs.Kind || binding.CredentialBindingID != "" {
				return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
			}
		default:
			if installed, ok := p.aiTargets[binding.TargetID]; ok {
				if binding.ProviderID != installed.ProviderID || binding.ProviderArtifactDigest != installed.ProviderArtifact || binding.ProviderABI != ai.ProviderABI || binding.TargetKind != "ai-model" || binding.ResourceKind != ai.KindModelSession || binding.CredentialBindingID != installed.CredentialBindingID {
					return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
				}
				if installed.Consent == "" {
					return admission.PolicyDecision{Outcome: admission.PolicyConsentRequired}, nil
				}
				consents[installed.Consent] = struct{}{}
				continue
			}
			if installed, ok := p.httpTargets[binding.TargetID]; ok {
				if binding.ProviderID != installed.ProviderID || binding.ProviderArtifactDigest != installed.ProviderArtifact || binding.ProviderABI != httpegress.ProviderABI || binding.TargetKind != httpegress.TargetKind || binding.ResourceKind != httpegress.KindHTTPSession || binding.CredentialBindingID != "" {
					return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
				}
				if installed.Consent == "" {
					return admission.PolicyDecision{Outcome: admission.PolicyConsentRequired}, nil
				}
				consents[installed.Consent] = struct{}{}
				continue
			}
			if installed, ok := p.applicationTargets[binding.TargetID]; ok {
				if binding.ProviderID != installed.ProviderID || binding.ProviderArtifactDigest != installed.ProviderArtifact || binding.ProviderABI != appcontrol.ProviderABI || binding.TargetKind != appcontrol.TargetKind || binding.ResourceKind != appcontrol.KindApplication || binding.CredentialBindingID != "" {
					return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
				}
				if installed.Consent == "" {
					return admission.PolicyDecision{Outcome: admission.PolicyConsentRequired}, nil
				}
				consents[installed.Consent] = struct{}{}
				continue
			}
			installed, ok := p.automationTargets[binding.TargetID]
			manifest := installed.Manifest.Machine()
			validKind := ok && installed.Manifest.SupportsResourceKind(binding.ResourceKind)
			if !ok || binding.ProviderID != manifest.ProviderID || binding.ProviderArtifactDigest != manifest.ProviderArtifact || binding.ProviderABI != manifest.ProviderABI || binding.TargetKind != manifest.TargetKind || !validKind || binding.CredentialBindingID != "" {
				return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
			}
			if installed.Consent == "" {
				return admission.PolicyDecision{Outcome: admission.PolicyConsentRequired}, nil
			}
			consents[installed.Consent] = struct{}{}
		}
	}
	lineage := make([]artifact.Digest, 0, len(consents))
	for consent := range consents {
		lineage = append(lineage, consent)
	}
	sort.Slice(lineage, func(i, j int) bool { return lineage[i] < lineage[j] })
	return admission.PolicyDecision{
		Outcome: admission.PolicyApproved, Generation: "builtin-local-v3", ExpiresAt: p.now().UTC().Add(p.ttl), ConsentLineage: lineage,
	}, nil
}
