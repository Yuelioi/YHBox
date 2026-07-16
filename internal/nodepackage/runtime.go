package nodepackage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

type RuntimeHost struct {
	APIGeneration   string
	OperatingSystem string
	Architecture    string
}

type RuntimePayload struct {
	root     string
	metadata Payload
}

type RuntimeNode struct {
	Contract       nodecontract.Contract
	Implementation Implementation
	Lock           nodecatalog.ImplementationLock
	Payload        RuntimePayload
}

type RuntimePackage struct {
	PackageID      string
	PackageVersion string
	ManifestDigest artifact.Digest
	Types          []datatype.Definition
	Capabilities   []capability.Definition
	Nodes          []RuntimeNode
}

func (s *Store) RuntimePackages(ctx context.Context, host RuntimeHost) ([]RuntimePackage, error) {
	if ctx == nil {
		return nil, errors.New("node package runtime projection requires a context")
	}
	if _, ok := parseHostAPI(host.APIGeneration); !ok || !identityPattern.MatchString(host.OperatingSystem) ||
		!identityPattern.MatchString(host.Architecture) {
		return nil, errors.New("node package runtime host identity is invalid")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	packageIDs := make([]string, 0, len(s.entries))
	for packageID := range s.entries {
		packageIDs = append(packageIDs, packageID)
	}
	slices.Sort(packageIDs)
	result := make([]RuntimePackage, 0, len(packageIDs))
	for _, packageID := range packageIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		installed := s.entries[packageID]
		if !installed.Enabled {
			continue
		}
		releaseIndex := releaseIndex(installed.Releases, installed.Current)
		if releaseIndex < 0 {
			return nil, fmt.Errorf("enabled node package %q has no current release", packageID)
		}
		release := installed.Releases[releaseIndex]
		if release.QuarantineReason != "" {
			return nil, fmt.Errorf("enabled node package %q is quarantined", packageID)
		}
		if reason, blocked := s.policy.blockReason(release.Trust.SignerKeyID, release.ManifestDigest); blocked {
			return nil, fmt.Errorf("enabled node package %q is blocked by trust policy: %s", packageID, reason)
		}
		generation := s.generationPath(release.ManifestDigest)
		manifest, err := OpenExtracted(ctx, generation)
		if err != nil || manifest.Digest() != release.ManifestDigest || manifest.PackageID() != installed.PackageID ||
			manifest.PackageVersion() != release.PackageVersion || manifest.PublisherNamespace() != installed.PublisherNamespace {
			return nil, fmt.Errorf("enabled node package %q failed runtime integrity verification", packageID)
		}
		if err := s.policy.verifyStored(manifest, release.Trust); err != nil {
			return nil, fmt.Errorf("verify enabled node package %q: %w", packageID, err)
		}
		if !manifest.SupportsHostAPI(host.APIGeneration) {
			continue
		}
		runtimePackage := RuntimePackage{
			PackageID: packageID, PackageVersion: manifest.PackageVersion(), ManifestDigest: manifest.Digest(),
			Types: manifest.Types(), Capabilities: manifest.Capabilities(),
		}
		for _, node := range manifest.Nodes() {
			if !supportsRuntimeHost(node.Implementation.Platforms, host) {
				continue
			}
			runtimePackage.Nodes = append(runtimePackage.Nodes, RuntimeNode{
				Contract: node.Contract, Implementation: node.Implementation,
				Lock: nodecatalog.ImplementationLock{
					PackageID: packageID, ArtifactDigest: manifest.Digest(), ABI: node.Implementation.ABI,
					Entrypoint: node.Implementation.Entrypoint,
				},
				Payload: RuntimePayload{root: generation, metadata: node.Implementation.Payload},
			})
		}
		if len(runtimePackage.Nodes) != 0 {
			result = append(result, runtimePackage)
		}
	}
	return result, nil
}

func (p RuntimePayload) Metadata() Payload { return p.metadata }

func (p RuntimePayload) Read(ctx context.Context, maxBytes int64) ([]byte, error) {
	if ctx == nil || maxBytes <= 0 || p.root == "" {
		return nil, errors.New("runtime payload read request is invalid")
	}
	if p.metadata.Size > maxBytes {
		return nil, fmt.Errorf("runtime payload exceeds %d byte host budget", maxBytes)
	}
	filename := filepath.Join(p.root, filepath.FromSlash(p.metadata.Path))
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != p.metadata.Size {
		return nil, errors.New("runtime payload identity changed")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open runtime payload: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	raw, err := io.ReadAll(io.TeeReader(io.LimitReader(file, maxBytes+1), hash))
	if err != nil {
		return nil, fmt.Errorf("read runtime payload: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(raw)) != p.metadata.Size || artifact.Digest(fmt.Sprintf("sha256:%x", hash.Sum(nil))) != p.metadata.Digest {
		return nil, errors.New("runtime payload digest changed")
	}
	return raw, nil
}

func supportsRuntimeHost(platforms PlatformSupport, host RuntimeHost) bool {
	return slices.Contains(platforms.OperatingSystems, host.OperatingSystem) && slices.Contains(platforms.Architectures, host.Architecture)
}
