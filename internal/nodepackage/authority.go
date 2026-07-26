package nodepackage

import (
	"context"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

// ReleaseAuthority is the host-neutral capability surface of one exact,
// verified cached package release. It does not enable or execute the package.
type ReleaseAuthority struct {
	PublisherNamespace string
	PackageID          string
	PackageVersion     string
	ManifestDigest     artifact.Digest
	Capabilities       []capability.Definition
	Contracts          []nodecontract.Contract
}

func (s *Store) ReleaseAuthority(
	ctx context.Context,
	packageID string,
	manifestDigest artifact.Digest,
) (ReleaseAuthority, error) {
	if ctx == nil || !manifestDigest.Valid() {
		return ReleaseAuthority{}, errors.New("node package Release authority request is invalid")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	installed, found := s.entries[packageID]
	if !found {
		return ReleaseAuthority{}, errors.New("node package is not cached")
	}
	index := releaseIndex(installed.Releases, manifestDigest)
	if index < 0 {
		return ReleaseAuthority{}, errors.New("node package Release is not cached")
	}
	release := installed.Releases[index]
	if release.QuarantineReason != "" {
		return ReleaseAuthority{}, errors.New("node package Release is quarantined")
	}
	if reason, blocked := s.policy.blockReason(release.Trust.SignerKeyID, release.ManifestDigest); blocked {
		return ReleaseAuthority{}, fmt.Errorf("node package Release is blocked by trust policy: %s", reason)
	}
	manifest, err := OpenExtracted(ctx, s.generationPath(manifestDigest))
	if err != nil || manifest.Digest() != manifestDigest || manifest.PackageID() != installed.PackageID ||
		manifest.PackageVersion() != release.PackageVersion ||
		manifest.PublisherNamespace() != installed.PublisherNamespace {
		return ReleaseAuthority{}, errors.New("node package Release failed authority integrity verification")
	}
	if err := s.policy.verifyStored(manifest, release.Trust); err != nil {
		return ReleaseAuthority{}, fmt.Errorf("verify node package Release authority: %w", err)
	}
	nodes := manifest.Nodes()
	contracts := make([]nodecontract.Contract, 0, len(nodes))
	for _, node := range nodes {
		contracts = append(contracts, node.Contract)
	}
	return ReleaseAuthority{
		PublisherNamespace: manifest.PublisherNamespace(),
		PackageID:          manifest.PackageID(), PackageVersion: manifest.PackageVersion(),
		ManifestDigest: manifest.Digest(), Capabilities: manifest.Capabilities(),
		Contracts: contracts,
	}, nil
}
