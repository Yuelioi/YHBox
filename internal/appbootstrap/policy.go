package appbootstrap

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

type builtinPolicy struct {
	blobDigest          artifact.Digest
	streamDigest        artifact.Digest
	workspaceFileDigest artifact.Digest
	aiTargets           map[string]ai.Installation
}

// NewBuiltinPolicy creates the explicit local policy for code compiled into
// Yotta. It cannot authorize third-party providers, credentials, or remote
// targets; plugin policies enter through their own admission owner.
func NewBuiltinPolicy(aiInstallations ai.Installations) (admission.Policy, error) {
	if !aiInstallations.Valid() {
		return nil, errors.New("built-in policy requires configured AI installations")
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
	return &builtinPolicy{blobDigest: blobDigest, streamDigest: streamDigest, workspaceFileDigest: workspaceFileDigest, aiTargets: aiTargets}, nil
}

func (p *builtinPolicy) Authorize(_ context.Context, request admission.PolicyRequest) (admission.PolicyDecision, error) {
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
				if installed.Profile.Machine().Evaluation != ai.EvaluationApproved &&
					requiresApprovedAI(request.Requirements, binding) {
					return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
				}
				continue
			}
			return admission.PolicyDecision{Outcome: admission.PolicyDenied}, nil
		}
	}
	return admission.PolicyDecision{Outcome: admission.PolicyApproved, Generation: "builtin-local-v6"}, nil
}

func requiresApprovedAI(requirements []capability.PlanEntry, binding capability.Binding) bool {
	for _, entry := range requirements {
		if entry.GraphID != binding.GraphID || entry.NodeID != binding.NodeID ||
			entry.Requirement.ID != binding.RequirementID {
			continue
		}
		for _, operation := range entry.Requirement.Operations {
			if operation == ai.OperationAgentStart || operation == ai.OperationAgentContinue {
				return true
			}
		}
	}
	return false
}
