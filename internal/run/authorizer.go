// Package run owns admission and lifecycle semantics.
package run

import (
	"bytes"
	"context"
	"errors"
	"sync"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/resource"
)

var ErrGrantDenied = errors.New("run grant denied resource authority")

// GrantAuthorizer is the sole adapter from a sealed Run Grant to Resource
// Broker authorization. Revocation is immediate for subsequent calls.
type GrantAuthorizer struct {
	mu      sync.RWMutex
	grant   capability.RunGrant
	revoked bool
	entries map[string]capability.GrantEntry
}

func NewGrantAuthorizer(grant capability.RunGrant) (*GrantAuthorizer, error) {
	if !grant.Valid() {
		return nil, errors.New("valid run grant is required")
	}
	entries := make(map[string]capability.GrantEntry)
	for _, entry := range grant.Entries() {
		key := grantKey(entry.Binding.GraphID, entry.Binding.NodeID, entry.Binding.RequirementID)
		if _, duplicate := entries[key]; duplicate {
			return nil, errors.New("run grant contains duplicate authority")
		}
		entries[key] = entry
	}
	return &GrantAuthorizer{grant: grant, entries: entries}, nil
}

func (a *GrantAuthorizer) Revoke() {
	a.mu.Lock()
	a.revoked = true
	a.mu.Unlock()
}

func (a *GrantAuthorizer) Scope(graphID, nodeID, requirementID, invocationID string) (resource.Scope, error) {
	scope, _, err := a.session(graphID, nodeID, requirementID, invocationID)
	return scope, err
}

func (a *GrantAuthorizer) session(graphID, nodeID, requirementID, invocationID string) (resource.Scope, capability.GrantEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	entry, ok := a.entries[grantKey(graphID, nodeID, requirementID)]
	if a.revoked || !ok || invocationID == "" {
		return resource.Scope{}, capability.GrantEntry{}, ErrGrantDenied
	}
	scope := resource.Scope{
		ProgramHash: a.grant.ProgramHash(), CapabilityPlanDigest: a.grant.PlanDigest(), GrantDigest: a.grant.Digest(),
		PolicyGeneration: a.grant.PolicyGeneration(), RunID: a.grant.RunID(), Principal: a.grant.Principal(),
		PluginInstanceID: entry.Binding.PluginInstanceID, SessionID: entry.Binding.SessionID,
		GraphID: graphID, NodeID: nodeID, RequirementID: requirementID, InvocationID: invocationID,
	}
	if err := scope.Validate(); err != nil {
		return resource.Scope{}, capability.GrantEntry{}, ErrGrantDenied
	}
	return scope, entry, nil
}

func (a *GrantAuthorizer) AuthorizeOpen(_ context.Context, request resource.OpenRequest) (resource.OpenAuthorization, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	entry, err := a.authority(request.Scope)
	if err != nil {
		return resource.OpenAuthorization{}, err
	}
	if request.ProviderID != entry.Binding.ProviderID || request.TargetID != entry.Binding.TargetID || request.Kind != entry.Binding.ResourceKind ||
		!operationSubset(request.Operations, entry.Operations) {
		return resource.OpenAuthorization{}, ErrGrantDenied
	}
	return resource.OpenAuthorization{
		CapabilityScope: append([]byte(nil), entry.Scope...), CredentialBindingID: entry.Binding.CredentialBindingID,
	}, nil
}

func (a *GrantAuthorizer) AuthorizeBorrow(_ context.Context, request resource.BorrowRequest) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	owner, err := a.authority(request.Owner)
	if err != nil {
		return err
	}
	borrower, err := a.authority(request.Borrower)
	if err != nil {
		return err
	}
	if request.ProviderID != owner.Binding.ProviderID || request.TargetID != owner.Binding.TargetID || request.Kind != owner.Binding.ResourceKind ||
		request.ProviderID != borrower.Binding.ProviderID || request.TargetID != borrower.Binding.TargetID || request.Kind != borrower.Binding.ResourceKind ||
		!bytes.Equal(owner.Scope, borrower.Scope) || !operationSubset(request.Operations, owner.Operations) ||
		!operationSubset(request.Operations, borrower.Operations) {
		return ErrGrantDenied
	}
	return nil
}

func (a *GrantAuthorizer) AuthorizeCall(_ context.Context, request resource.AuthorizationCall) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	entry, err := a.authority(request.Scope)
	if err != nil {
		return err
	}
	if request.ProviderID != entry.Binding.ProviderID || request.TargetID != entry.Binding.TargetID || request.Kind != entry.Binding.ResourceKind ||
		!operationSubset([]string{request.Operation}, entry.Operations) {
		return ErrGrantDenied
	}
	return nil
}

func (a *GrantAuthorizer) authority(scope resource.Scope) (capability.GrantEntry, error) {
	if a.revoked || scope.ProgramHash != a.grant.ProgramHash() ||
		scope.CapabilityPlanDigest != a.grant.PlanDigest() || scope.GrantDigest != a.grant.Digest() ||
		scope.PolicyGeneration != a.grant.PolicyGeneration() || scope.RunID != a.grant.RunID() || scope.Principal != a.grant.Principal() {
		return capability.GrantEntry{}, ErrGrantDenied
	}
	entry, ok := a.entries[grantKey(scope.GraphID, scope.NodeID, scope.RequirementID)]
	if !ok || scope.PluginInstanceID != entry.Binding.PluginInstanceID || scope.SessionID != entry.Binding.SessionID {
		return capability.GrantEntry{}, ErrGrantDenied
	}
	return entry, nil
}

func operationSubset(requested, granted []string) bool {
	allowed := make(map[string]struct{}, len(granted))
	for _, operation := range granted {
		allowed[operation] = struct{}{}
	}
	for _, operation := range requested {
		if _, ok := allowed[operation]; !ok {
			return false
		}
	}
	return true
}

func grantKey(graphID, nodeID, requirementID string) string {
	return graphID + "\x00" + nodeID + "\x00" + requirementID
}
