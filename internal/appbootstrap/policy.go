package appbootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/stream"
)

type builtinPolicy struct {
	now          func() time.Time
	ttl          time.Duration
	blobDigest   artifact.Digest
	streamDigest artifact.Digest
}

// NewBuiltinPolicy creates the explicit local policy for code compiled into
// Yotta. It cannot authorize third-party providers, credentials, or remote
// targets; plugin policies enter through their own admission owner.
func NewBuiltinPolicy(now func() time.Time, ttl time.Duration) (admission.Policy, error) {
	if now == nil || ttl <= 0 || ttl > 24*time.Hour {
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
	return &builtinPolicy{now: now, ttl: ttl, blobDigest: blobDigest, streamDigest: streamDigest}, nil
}

func (p *builtinPolicy) Authorize(_ context.Context, request admission.PolicyRequest) (admission.PolicyDecision, error) {
	for _, binding := range request.Bindings {
		if binding.PluginInstanceID != "builtin" || binding.CredentialBindingID != "" {
			return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
		}
		switch binding.ProviderID {
		case blob.ProviderID:
			if binding.ProviderArtifactDigest != p.blobDigest || binding.ProviderABI != blob.ProviderABI ||
				binding.TargetID != "workspace" || binding.TargetKind != "blob-store" {
				return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
			}
		case stream.ProviderID:
			if binding.ProviderArtifactDigest != p.streamDigest || binding.ProviderABI != stream.ProviderABI ||
				binding.TargetID != "memory" || binding.TargetKind != "stream-session" {
				return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
			}
		default:
			return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
		}
	}
	return admission.PolicyDecision{
		Outcome: admission.PolicyApproved, Generation: "builtin-local-v1", ExpiresAt: p.now().UTC().Add(p.ttl),
	}, nil
}
