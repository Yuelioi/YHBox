package appbootstrap

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
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
}

// NewBuiltinPolicy creates the explicit local policy for code compiled into
// Yotta. It cannot authorize third-party providers, credentials, or remote
// targets; plugin policies enter through their own admission owner.
func NewBuiltinPolicy(now func() time.Time, ttl time.Duration, installations ai.Installations) (admission.Policy, error) {
	if now == nil || ttl <= 0 || ttl > 24*time.Hour || !installations.Valid() {
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
	aiTargets := make(map[string]ai.Installation, len(installations.Entries()))
	for _, installed := range installations.Entries() {
		if _, duplicate := aiTargets[installed.TargetID]; duplicate {
			return nil, errors.New("built-in policy received duplicate AI targets")
		}
		aiTargets[installed.TargetID] = installed
	}
	return &builtinPolicy{now: now, ttl: ttl, blobDigest: blobDigest, streamDigest: streamDigest, workspaceFileDigest: workspaceFileDigest, aiTargets: aiTargets}, nil
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
			installed, ok := p.aiTargets[binding.TargetID]
			if !ok || binding.ProviderID != installed.ProviderID || binding.ProviderArtifactDigest != installed.ProviderArtifact ||
				binding.ProviderABI != ai.ProviderABI || binding.TargetKind != "ai-model" || binding.ResourceKind != ai.KindModelSession ||
				binding.CredentialBindingID != installed.CredentialBindingID {
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
		Outcome: admission.PolicyApproved, Generation: "builtin-local-v2", ExpiresAt: p.now().UTC().Add(p.ttl), ConsentLineage: lineage,
	}, nil
}
