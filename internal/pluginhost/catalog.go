package pluginhost

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodepackage"
)

type CatalogProjection struct {
	Catalog  nodecatalog.Snapshot
	Packages []nodepackage.RuntimePackage
}

func MergeCatalog(base nodecatalog.Snapshot, packages []nodepackage.RuntimePackage, generatorVersion string) (CatalogProjection, error) {
	if !base.Valid() {
		return CatalogProjection{}, errors.New("plugin catalog merge requires a valid base Catalog")
	}
	types := base.Types()
	capabilities := base.Capabilities()
	bindings := base.Bindings()
	entrypoints := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		entrypoints[binding.Implementation.Entrypoint] = binding.Contract.NodeRef().NodeTypeID
	}
	for _, runtimePackage := range packages {
		if runtimePackage.PackageID == "" || !runtimePackage.ManifestDigest.Valid() {
			return CatalogProjection{}, errors.New("plugin catalog contains an invalid runtime package")
		}
		types = append(types, runtimePackage.Types...)
		capabilities = append(capabilities, runtimePackage.Capabilities...)
		for _, node := range runtimePackage.Nodes {
			if node.Lock.PackageID != runtimePackage.PackageID || node.Lock.ArtifactDigest != runtimePackage.ManifestDigest ||
				node.Lock.ABI != node.Implementation.ABI || node.Lock.Entrypoint != node.Implementation.Entrypoint {
				return CatalogProjection{}, fmt.Errorf("plugin node %q does not match its package lock", node.Contract.NodeRef().NodeTypeID)
			}
			if existing, duplicate := entrypoints[node.Lock.Entrypoint]; duplicate {
				return CatalogProjection{}, fmt.Errorf("plugin entrypoint %q conflicts with node %q", node.Lock.Entrypoint, existing)
			}
			entrypoints[node.Lock.Entrypoint] = node.Contract.NodeRef().NodeTypeID
			bindings = append(bindings, nodecatalog.Binding{Contract: node.Contract, Implementation: node.Lock})
		}
	}
	merged, err := nodecatalog.Seal(types, capabilities, bindings, generatorVersion)
	if err != nil {
		return CatalogProjection{}, fmt.Errorf("merge plugin Catalog: %w", err)
	}
	return CatalogProjection{Catalog: merged, Packages: clonePackages(packages)}, nil
}

func clonePackages(source []nodepackage.RuntimePackage) []nodepackage.RuntimePackage {
	result := append([]nodepackage.RuntimePackage(nil), source...)
	for index := range result {
		result[index].Types = append(result[index].Types[:0:0], result[index].Types...)
		result[index].Capabilities = append(result[index].Capabilities[:0:0], result[index].Capabilities...)
		result[index].Nodes = append(result[index].Nodes[:0:0], result[index].Nodes...)
	}
	return result
}
