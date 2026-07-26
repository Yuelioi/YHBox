package appbootstrap

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
)

func projectWorkflowCapabilityScopes(
	ctx context.Context,
	source schema.WorkflowSource,
	catalog nodecatalog.Snapshot,
	packages *nodepackage.Store,
) ([]workflowinstallation.CapabilityScope, error) {
	if ctx == nil || !catalog.Valid() {
		return nil, errors.New("Workflow capability projection is unavailable")
	}
	contracts := make(map[nodecontract.NodeRef]nodecontract.Contract)
	definitions := make(map[capability.Ref]capability.Definition)
	for _, dependency := range source.Dependencies {
		resolved := false
		if packages != nil {
			authority, err := packages.ReleaseAuthority(ctx, dependency.PackageID, dependency.ManifestDigest)
			if err != nil {
				return nil, fmt.Errorf("resolve node package Release authority %q: %w", dependency.PackageID, err)
			}
			if authority.PublisherNamespace != dependency.PublisherNamespace ||
				authority.PackageID != dependency.PackageID ||
				authority.PackageVersion != dependency.PackageVersion ||
				authority.ManifestDigest != dependency.ManifestDigest {
				return nil, errors.New("node package Release authority identity does not match Workflow dependency")
			}
			for _, contract := range authority.Contracts {
				contracts[contract.NodeRef()] = contract
			}
			for _, definition := range authority.Capabilities {
				definitions[definition.Ref()] = definition
			}
			resolved = true
		}
		for _, ref := range dependency.NodeRefs {
			if contract, found := contracts[ref]; found && contract.NodeRef() == ref {
				continue
			}
			entry, found := catalog.Lookup(ref.NodeTypeID)
			if !found || entry.Contract.NodeRef() != ref ||
				entry.Implementation.PackageID != dependency.PackageID ||
				entry.Implementation.ArtifactDigest != dependency.ManifestDigest {
				if resolved {
					return nil, fmt.Errorf("node package Release does not contain declared node %q", ref.NodeTypeID)
				}
				return nil, fmt.Errorf("node package Release authority for %q is unavailable", dependency.PackageID)
			}
			contracts[ref] = entry.Contract
		}
	}
	result := make([]workflowinstallation.CapabilityScope, 0)
	seen := make(map[string]struct{})
	for _, graph := range source.Graphs {
		for _, node := range graph.Nodes {
			if node.Disabled {
				continue
			}
			contract, found := contracts[node.NodeRef]
			if !found {
				entry, ok := catalog.Lookup(node.NodeRef.NodeTypeID)
				if !ok || entry.Contract.NodeRef() != node.NodeRef {
					return nil, fmt.Errorf("exact node contract %q is unavailable", node.NodeRef.NodeTypeID)
				}
				contract = entry.Contract
			}
			requirements, err := compiler.ResolveNodeCapabilityRequirements(
				source.TargetDefaults, node, contract,
			)
			if err != nil {
				return nil, fmt.Errorf("resolve node %q capabilities: %w", node.ID, err)
			}
			for _, requirement := range requirements {
				definition, ok := definitions[requirement.Capability]
				if !ok {
					definition, ok = catalog.LookupCapability(requirement.Capability.CapabilityID)
				}
				if !ok || definition.Ref() != requirement.Capability {
					return nil, fmt.Errorf("capability definition %q is unavailable", requirement.Capability.CapabilityID)
				}
				machine := definition.Machine()
				scope := workflowinstallation.CapabilityScope{
					NodeTypeID: node.NodeRef.NodeTypeID, CapabilityID: requirement.Capability.CapabilityID,
					Operations: append([]string(nil), requirement.Operations...),
					TargetSlot: requirement.TargetSlot, CredentialSlot: requirement.CredentialSlot,
					Scope: string(requirement.Scope), Risk: string(machine.Risk), Consent: string(machine.Consent),
				}
				key := capabilityProjectionKey(scope)
				if _, duplicate := seen[key]; duplicate {
					continue
				}
				seen[key] = struct{}{}
				result = append(result, scope)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left := capabilityProjectionKey(result[i])
		right := capabilityProjectionKey(result[j])
		return left < right
	})
	return result, nil
}

func capabilityProjectionKey(scope workflowinstallation.CapabilityScope) string {
	return strings.Join([]string{
		scope.NodeTypeID, scope.CapabilityID, strings.Join(scope.Operations, "\x1f"),
		scope.TargetSlot, scope.CredentialSlot, scope.Scope, scope.Risk, scope.Consent,
	}, "\x00")
}
