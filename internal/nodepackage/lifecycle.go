package nodepackage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/durablefs"
)

const (
	RegistryFormat        = "yotta.node-package-registry"
	RegistryVersion       = "3"
	legacyRegistryVersion = "2"
	registryFilename      = "registry.json"
	generationsDir        = "generations"
	maxRegistryBytes      = 16 << 20
	maxStorePackages      = 4_096
	maxStoreReleases      = 128
)

var quarantineReasonPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)

type PackageRelease struct {
	ManifestDigest   artifact.Digest `json:"manifestDigest"`
	PackageVersion   string          `json:"packageVersion"`
	Trust            PackageTrust    `json:"trust"`
	QuarantineReason string          `json:"quarantineReason,omitempty"`
}

type PackageInstallation struct {
	PackageID          string           `json:"packageId"`
	PublisherNamespace string           `json:"publisherNamespace"`
	Current            artifact.Digest  `json:"current"`
	Rollback           artifact.Digest  `json:"rollback,omitempty"`
	Enabled            bool             `json:"enabled"`
	Releases           []PackageRelease `json:"releases"`
}

type registryDocument struct {
	Format   string                `json:"format"`
	Version  string                `json:"version"`
	Trust    json.RawMessage       `json:"trust"`
	Grants   []PackageTrustGrant   `json:"grants,omitempty"`
	Packages []PackageInstallation `json:"packages"`
}

type PackageTrustGrant struct {
	PublisherKeyID artifact.Digest `json:"publisherKeyId"`
	PackageID      string          `json:"packageId"`
}

type ArchiveTrust struct {
	PublisherNamespace string
	PackageID          string
	PackageVersion     string
	ManifestDigest     artifact.Digest
	PublisherKeyID     artifact.Digest
	Granted            bool
}

type Store struct {
	mu      sync.RWMutex
	root    string
	policy  TrustPolicy
	grants  map[string]PackageTrustGrant
	entries map[string]PackageInstallation
}

func CreateStore(ctx context.Context, root string, bootstrap TrustPolicy) (*Store, error) {
	if !bootstrap.Valid() {
		return nil, errors.New("node package store bootstrap trust policy is invalid")
	}
	return openStore(ctx, root, &bootstrap)
}

func OpenStore(ctx context.Context, root string) (*Store, error) {
	return openStore(ctx, root, nil)
}

// OpenStoreIfPresent opens only an already initialized authority registry. It
// never creates a package store or trust root as a startup side effect.
func OpenStoreIfPresent(ctx context.Context, root string) (*Store, bool, error) {
	registry := filepath.Join(root, registryFilename)
	info, err := os.Lstat(registry)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, false, errors.New("node package registry is not a regular file")
	}
	store, err := OpenStore(ctx, root)
	return store, err == nil, err
}

func openStore(ctx context.Context, root string, bootstrap *TrustPolicy) (*Store, error) {
	if ctx == nil {
		return nil, errors.New("node package store requires a context")
	}
	resolved, err := filepath.Abs(root)
	if err != nil || strings.TrimSpace(root) == "" {
		return nil, errors.New("node package store root is invalid")
	}
	root = filepath.Clean(resolved)
	rootInfo, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(root, 0o700); err != nil {
			return nil, fmt.Errorf("create node package store root: %w", err)
		}
		rootInfo, err = os.Lstat(root)
	}
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("node package store root is not a directory")
	}
	if err := os.MkdirAll(filepath.Join(root, generationsDir), 0o700); err != nil {
		return nil, fmt.Errorf("create node package store: %w", err)
	}
	generationsInfo, err := os.Lstat(filepath.Join(root, generationsDir))
	if err != nil || !generationsInfo.IsDir() || generationsInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("node package generations path is not a directory")
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("secure node package store: %w", err)
	}
	if err := os.Chmod(filepath.Join(root, generationsDir), 0o700); err != nil {
		return nil, fmt.Errorf("secure node package generations directory: %w", err)
	}
	if err := cleanupInterrupted(root); err != nil {
		return nil, err
	}
	if err := validateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, policy, grants, found, err := loadRegistry(ctx, root)
	if err != nil {
		return nil, err
	}
	if !found {
		if bootstrap == nil {
			return nil, errors.New("new node package store requires a bootstrap trust policy")
		}
		policy = *bootstrap
		entries = map[string]PackageInstallation{}
		grants = map[string]PackageTrustGrant{}
		raw, err := marshalRegistry(entries, policy, grants)
		if err != nil {
			return nil, err
		}
		if err := durablefs.WriteFile(filepath.Join(root, registryFilename), raw, 0o600); err != nil {
			return nil, fmt.Errorf("write initial node package registry: %w", err)
		}
	} else if bootstrap != nil {
		return nil, errors.New("node package store is already initialized")
	}
	if err := cleanupOrphanGenerations(root, entries); err != nil {
		return nil, err
	}
	return &Store{root: root, policy: policy, grants: grants, entries: entries}, nil
}

func grantKey(keyID artifact.Digest, packageID string) string {
	return keyID.String() + "\x00" + packageID
}

func (s *Store) GrantPackageTrust(
	ctx context.Context,
	publisherKeyID artifact.Digest,
	packageID string,
) error {
	if ctx == nil || !publisherKeyID.Valid() || strings.TrimSpace(packageID) != packageID || packageID == "" {
		return errors.New("node package trust grant is invalid")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key, found := s.policy.state.keys[publisherKeyID]
	if !found {
		return errors.New("node package trust grant uses an unknown publisher key")
	}
	namespace, err := validateNamespace(key.namespace)
	if err != nil || validateOwnedURI(namespace, packageID, "package ID") != nil {
		return errors.New("node package trust grant is outside the publisher authority")
	}
	next := cloneGrants(s.grants)
	next[grantKey(publisherKeyID, packageID)] = PackageTrustGrant{
		PublisherKeyID: publisherKeyID, PackageID: packageID,
	}
	raw, err := marshalRegistry(s.entries, s.policy, next)
	if err != nil {
		return err
	}
	writeErr := durablefs.WriteFile(filepath.Join(s.root, registryFilename), raw, 0o600)
	if writeErr != nil && !durablefs.Committed(writeErr) {
		return fmt.Errorf("write node package registry trust grant: %w", writeErr)
	}
	s.grants = next
	if writeErr != nil {
		return fmt.Errorf("node package registry trust grant committed without confirmed durability: %w", writeErr)
	}
	return nil
}

func (s *Store) InspectArchiveTrust(ctx context.Context, archivePath string) (ArchiveTrust, error) {
	if ctx == nil {
		return ArchiveTrust{}, errors.New("node package archive trust inspection requires a context")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	manifest, envelope, err := VerifyArchiveSignature(ctx, archivePath, s.policy)
	if err != nil {
		return ArchiveTrust{}, err
	}
	trust, err := VerifySignature(manifest, envelope, s.policy)
	if err != nil {
		return ArchiveTrust{}, err
	}
	_, granted := s.grants[grantKey(trust.SignerKeyID, manifest.PackageID())]
	return ArchiveTrust{
		PublisherNamespace: manifest.PublisherNamespace(),
		PackageID:          manifest.PackageID(), PackageVersion: manifest.PackageVersion(),
		ManifestDigest: manifest.Digest(), PublisherKeyID: trust.SignerKeyID,
		Granted: granted,
	}, nil
}

func (s *Store) List() []PackageInstallation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	packages := make([]PackageInstallation, 0, len(s.entries))
	for _, installed := range s.entries {
		packages = append(packages, cloneInstallation(installed))
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].PackageID < packages[j].PackageID })
	return packages
}

func (s *Store) Get(packageID string) (PackageInstallation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	installed, found := s.entries[packageID]
	return cloneInstallation(installed), found
}

func (s *Store) TrustPolicy() TrustPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy
}

func (s *Store) ApplyTrustPolicy(ctx context.Context, nextPolicy TrustPolicy) error {
	if ctx == nil {
		return errors.New("node package trust policy update requires a context")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateTrustTransition(s.policy, nextPolicy); err != nil {
		return err
	}
	next := cloneEntries(s.entries)
	for packageID, installed := range next {
		for index := range installed.Releases {
			if err := ctx.Err(); err != nil {
				return err
			}
			release := &installed.Releases[index]
			manifest, err := OpenExtracted(ctx, s.generationPath(release.ManifestDigest))
			if err != nil || manifest.Digest() != release.ManifestDigest {
				return errors.New("node package generation failed integrity verification during trust policy update")
			}
			if err := nextPolicy.verifyStored(manifest, release.Trust); err != nil {
				return err
			}
			if reason, blocked := nextPolicy.blockReason(release.Trust.SignerKeyID, release.ManifestDigest); blocked {
				if release.QuarantineReason == "" {
					release.QuarantineReason = reason
				}
				if installed.Current == release.ManifestDigest {
					installed.Enabled = false
				}
			}
		}
		next[packageID] = installed
	}
	return s.commitLocked(next, nextPolicy)
}

func (s *Store) InstallArchive(ctx context.Context, archivePath string) (PackageInstallation, error) {
	if ctx == nil {
		return PackageInstallation{}, errors.New("node package installation requires a context")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	verifiedManifest, envelope, err := VerifyArchiveSignature(ctx, archivePath, s.policy)
	if err != nil {
		return PackageInstallation{}, err
	}
	trust, err := VerifySignature(verifiedManifest, envelope, s.policy)
	if err != nil {
		return PackageInstallation{}, err
	}
	if _, granted := s.grants[grantKey(trust.SignerKeyID, verifiedManifest.PackageID())]; !granted {
		return PackageInstallation{}, errors.New("node package install requires an explicit package-scoped publisher trust grant")
	}
	incoming, err := os.MkdirTemp(s.root, ".incoming-*")
	if err != nil {
		return PackageInstallation{}, fmt.Errorf("create node package incoming directory: %w", err)
	}
	defer os.RemoveAll(incoming)
	candidate := filepath.Join(incoming, "generation")
	manifest, err := ExtractArchive(ctx, archivePath, candidate)
	if err != nil {
		return PackageInstallation{}, err
	}
	if manifest.Digest() != verifiedManifest.Digest() {
		return PackageInstallation{}, errors.New("node package archive changed between signature verification and extraction")
	}
	generation := s.generationPath(manifest.Digest())
	publishCandidate := false
	if info, err := os.Lstat(generation); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return PackageInstallation{}, errors.New("node package generation path is not a directory")
		}
		existing, err := OpenExtracted(ctx, generation)
		if err != nil || existing.Digest() != manifest.Digest() {
			return PackageInstallation{}, errors.New("existing node package generation failed integrity verification")
		}
		if !generationReferenced(s.entries, manifest.Digest()) {
			if err := os.RemoveAll(generation); err != nil {
				return PackageInstallation{}, fmt.Errorf("remove inert node package generation: %w", err)
			}
			publishCandidate = true
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return PackageInstallation{}, fmt.Errorf("inspect node package generation: %w", err)
	} else {
		publishCandidate = true
	}
	if publishCandidate {
		if err := durablefs.Replace(candidate, generation); err != nil {
			return PackageInstallation{}, fmt.Errorf("publish durable node package generation: %w", err)
		}
	}

	next := cloneEntries(s.entries)
	installed, found := next[manifest.PackageID()]
	if found && installed.PublisherNamespace != manifest.PublisherNamespace() {
		return PackageInstallation{}, errors.New("node package publisher namespace changed for an existing package ID")
	}
	if !found {
		installed = PackageInstallation{PackageID: manifest.PackageID(), PublisherNamespace: manifest.PublisherNamespace()}
	}
	if installed.Current == manifest.Digest() {
		return cloneInstallation(installed), nil
	}
	release := PackageRelease{
		ManifestDigest: manifest.Digest(), PackageVersion: manifest.PackageVersion(),
		Trust: trust,
	}
	if index := releaseIndex(installed.Releases, manifest.Digest()); index < 0 {
		if len(installed.Releases) >= maxStoreReleases {
			return PackageInstallation{}, errors.New("node package release history exceeds its budget")
		}
		installed.Releases = append(installed.Releases, release)
		sort.Slice(installed.Releases, func(i, j int) bool {
			return installed.Releases[i].ManifestDigest < installed.Releases[j].ManifestDigest
		})
	} else if installed.Releases[index].QuarantineReason != "" {
		return PackageInstallation{}, errors.New("quarantined node package generation cannot be reinstalled")
	}
	if installed.Current.Valid() {
		installed.Rollback = installed.Current
	}
	installed.Current = manifest.Digest()
	installed.Enabled = true
	next[installed.PackageID] = installed
	if err := s.commitLocked(next, s.policy); err != nil {
		return cloneInstallation(installed), err
	}
	return cloneInstallation(installed), nil
}

func (s *Store) Disable(packageID string) (PackageInstallation, error) {
	return s.mutate(packageID, func(installed *PackageInstallation) error {
		installed.Enabled = false
		return nil
	})
}

func (s *Store) Enable(packageID string) (PackageInstallation, error) {
	return s.mutate(packageID, func(installed *PackageInstallation) error {
		index := releaseIndex(installed.Releases, installed.Current)
		if index < 0 || installed.Releases[index].QuarantineReason != "" {
			return errors.New("quarantined node package generation cannot be enabled")
		}
		if reason, blocked := s.policy.blockReason(installed.Releases[index].Trust.SignerKeyID, installed.Current); blocked {
			return fmt.Errorf("node package generation is blocked by trust policy: %s", reason)
		}
		installed.Enabled = true
		return nil
	})
}

func (s *Store) Quarantine(packageID string, digest artifact.Digest, reasonCode string) (PackageInstallation, error) {
	if !digest.Valid() || !quarantineReasonPattern.MatchString(reasonCode) {
		return PackageInstallation{}, errors.New("node package quarantine request is invalid")
	}
	return s.mutate(packageID, func(installed *PackageInstallation) error {
		index := releaseIndex(installed.Releases, digest)
		if index < 0 {
			return errors.New("node package generation is not installed")
		}
		installed.Releases[index].QuarantineReason = reasonCode
		if installed.Current == digest {
			installed.Enabled = false
		}
		return nil
	})
}

func (s *Store) Rollback(ctx context.Context, packageID string) (PackageInstallation, error) {
	if ctx == nil {
		return PackageInstallation{}, errors.New("node package rollback requires a context")
	}
	return s.mutate(packageID, func(installed *PackageInstallation) error {
		if !installed.Rollback.Valid() {
			return errors.New("node package has no rollback generation")
		}
		index := releaseIndex(installed.Releases, installed.Rollback)
		if index < 0 || installed.Releases[index].QuarantineReason != "" {
			return errors.New("node package rollback generation is unavailable or quarantined")
		}
		if reason, blocked := s.policy.blockReason(installed.Releases[index].Trust.SignerKeyID, installed.Rollback); blocked {
			return fmt.Errorf("node package rollback generation is blocked by trust policy: %s", reason)
		}
		manifest, err := OpenExtracted(ctx, s.generationPath(installed.Rollback))
		if err != nil || manifest.Digest() != installed.Rollback {
			return errors.New("node package rollback generation failed integrity verification")
		}
		installed.Current, installed.Rollback = installed.Rollback, installed.Current
		installed.Enabled = false
		return nil
	})
}

func (s *Store) Uninstall(packageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	installed, found := s.entries[packageID]
	if !found {
		return errors.New("node package is not installed")
	}
	next := cloneEntries(s.entries)
	delete(next, packageID)
	if err := s.commitLocked(next, s.policy); err != nil {
		return err
	}
	for _, release := range installed.Releases {
		_ = os.RemoveAll(s.generationPath(release.ManifestDigest))
	}
	return nil
}

func (s *Store) mutate(packageID string, apply func(*PackageInstallation) error) (PackageInstallation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneEntries(s.entries)
	installed, found := next[packageID]
	if !found {
		return PackageInstallation{}, errors.New("node package is not installed")
	}
	if err := apply(&installed); err != nil {
		return PackageInstallation{}, err
	}
	next[packageID] = installed
	if err := s.commitLocked(next, s.policy); err != nil {
		return cloneInstallation(installed), err
	}
	return cloneInstallation(installed), nil
}

func (s *Store) commitLocked(next map[string]PackageInstallation, policy TrustPolicy) error {
	raw, err := marshalRegistry(next, policy, s.grants)
	if err != nil {
		return err
	}
	writeErr := durablefs.WriteFile(filepath.Join(s.root, registryFilename), raw, 0o600)
	if writeErr != nil && !durablefs.Committed(writeErr) {
		return fmt.Errorf("write node package registry: %w", writeErr)
	}
	s.entries = next
	s.policy = policy
	if writeErr != nil {
		return fmt.Errorf("node package registry committed without confirmed durability: %w", writeErr)
	}
	return nil
}

func loadRegistry(ctx context.Context, root string) (map[string]PackageInstallation, TrustPolicy, map[string]PackageTrustGrant, bool, error) {
	registryPath := filepath.Join(root, registryFilename)
	info, statErr := os.Lstat(registryPath)
	if errors.Is(statErr, os.ErrNotExist) {
		return nil, TrustPolicy{}, nil, false, nil
	}
	if statErr != nil {
		return nil, TrustPolicy{}, nil, false, fmt.Errorf("inspect node package registry: %w", statErr)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > maxRegistryBytes {
		return nil, TrustPolicy{}, nil, false, errors.New("node package registry is invalid or exceeds its byte budget")
	}
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, TrustPolicy{}, nil, false, fmt.Errorf("read node package registry: %w", err)
	}
	if len(raw) == 0 || len(raw) > maxRegistryBytes {
		return nil, TrustPolicy{}, nil, false, errors.New("node package registry exceeds its byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, 32, 131072, maxRegistryBytes); err != nil {
		return nil, TrustPolicy{}, nil, false, fmt.Errorf("inspect node package registry: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return nil, TrustPolicy{}, nil, false, errors.New("node package registry is not canonical JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document registryDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, TrustPolicy{}, nil, false, fmt.Errorf("decode node package registry: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, TrustPolicy{}, nil, false, errors.New("node package registry contains trailing JSON values")
	}
	if document.Format != RegistryFormat ||
		document.Version != RegistryVersion && document.Version != legacyRegistryVersion {
		return nil, TrustPolicy{}, nil, false, errors.New("unsupported node package registry format")
	}
	policy, err := OpenTrustPolicy(document.Trust)
	if err != nil {
		return nil, TrustPolicy{}, nil, false, fmt.Errorf("open node package registry trust policy: %w", err)
	}
	if len(document.Packages) > maxStorePackages {
		return nil, TrustPolicy{}, nil, false, errors.New("node package registry package count exceeds its budget")
	}
	entries := make(map[string]PackageInstallation, len(document.Packages))
	previous := ""
	for _, installed := range document.Packages {
		if installed.PackageID <= previous {
			return nil, TrustPolicy{}, nil, false, errors.New("node package registry package order or identity is invalid")
		}
		previous = installed.PackageID
		if err := validateInstallation(ctx, root, installed, policy); err != nil {
			return nil, TrustPolicy{}, nil, false, fmt.Errorf("validate installed node package %q: %w", installed.PackageID, err)
		}
		entries[installed.PackageID] = cloneInstallation(installed)
	}
	grants := make(map[string]PackageTrustGrant)
	if document.Version == legacyRegistryVersion {
		for _, installed := range entries {
			for _, release := range installed.Releases {
				grant := PackageTrustGrant{
					PublisherKeyID: release.Trust.SignerKeyID, PackageID: installed.PackageID,
				}
				grants[grantKey(grant.PublisherKeyID, grant.PackageID)] = grant
			}
		}
	} else {
		previousGrant := ""
		for _, grant := range document.Grants {
			key := grantKey(grant.PublisherKeyID, grant.PackageID)
			publisherKey, found := policy.state.keys[grant.PublisherKeyID]
			namespace, namespaceErr := validateNamespace(publisherKey.namespace)
			if !found || key <= previousGrant || namespaceErr != nil ||
				validateOwnedURI(namespace, grant.PackageID, "package ID") != nil {
				return nil, TrustPolicy{}, nil, false, errors.New("node package registry trust grant is invalid")
			}
			previousGrant = key
			grants[key] = grant
		}
	}
	if document.Version == legacyRegistryVersion {
		current, err := marshalRegistry(entries, policy, grants)
		if err != nil {
			return nil, TrustPolicy{}, nil, false, fmt.Errorf("migrate node package registry: %w", err)
		}
		if err := durablefs.WriteFile(registryPath, current, 0o600); err != nil {
			return nil, TrustPolicy{}, nil, false, fmt.Errorf("publish migrated node package registry: %w", err)
		}
	}
	return entries, policy, grants, true, nil
}

func validateInstallation(ctx context.Context, root string, installed PackageInstallation, policy TrustPolicy) error {
	if installed.PackageID == "" || installed.PublisherNamespace == "" || !installed.Current.Valid() ||
		len(installed.Releases) == 0 || len(installed.Releases) > maxStoreReleases {
		return errors.New("node package installation identity is invalid")
	}
	previous := artifact.Digest("")
	foundCurrent, foundRollback := false, installed.Rollback == ""
	for _, release := range installed.Releases {
		if !release.ManifestDigest.Valid() || release.ManifestDigest <= previous {
			return errors.New("node package release identity or trust is invalid")
		}
		previous = release.ManifestDigest
		if release.QuarantineReason != "" && !quarantineReasonPattern.MatchString(release.QuarantineReason) {
			return errors.New("node package quarantine reason is invalid")
		}
		manifest, err := OpenExtracted(ctx, generationPath(root, release.ManifestDigest))
		if err != nil || manifest.Digest() != release.ManifestDigest || manifest.PackageID() != installed.PackageID ||
			manifest.PublisherNamespace() != installed.PublisherNamespace || manifest.PackageVersion() != release.PackageVersion {
			return errors.New("node package generation does not match its registry release")
		}
		if err := policy.verifyStored(manifest, release.Trust); err != nil {
			return err
		}
		if _, blocked := policy.blockReason(release.Trust.SignerKeyID, release.ManifestDigest); blocked && release.QuarantineReason == "" {
			return errors.New("node package trust-blocked generation is not quarantined")
		}
		if release.ManifestDigest == installed.Current {
			foundCurrent = true
			if release.QuarantineReason != "" && installed.Enabled {
				return errors.New("quarantined current node package generation is enabled")
			}
		}
		if release.ManifestDigest == installed.Rollback {
			foundRollback = true
		}
	}
	if !foundCurrent || !foundRollback || installed.Current == installed.Rollback {
		return errors.New("node package current or rollback pointer is invalid")
	}
	return nil
}

func marshalRegistry(
	entries map[string]PackageInstallation,
	policy TrustPolicy,
	grants map[string]PackageTrustGrant,
) ([]byte, error) {
	if !policy.Valid() {
		return nil, errors.New("node package registry trust policy is invalid")
	}
	if len(entries) > maxStorePackages {
		return nil, errors.New("node package registry package count exceeds its budget")
	}
	packages := make([]PackageInstallation, 0, len(entries))
	for _, installed := range entries {
		packages = append(packages, cloneInstallation(installed))
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].PackageID < packages[j].PackageID })
	orderedGrants := make([]PackageTrustGrant, 0, len(grants))
	for _, grant := range grants {
		orderedGrants = append(orderedGrants, grant)
	}
	sort.Slice(orderedGrants, func(i, j int) bool {
		return grantKey(orderedGrants[i].PublisherKeyID, orderedGrants[i].PackageID) <
			grantKey(orderedGrants[j].PublisherKeyID, orderedGrants[j].PackageID)
	})
	raw, err := artifact.Marshal(registryDocument{
		Format: RegistryFormat, Version: RegistryVersion, Trust: policy.Bytes(),
		Grants: orderedGrants, Packages: packages,
	})
	if err != nil {
		return nil, err
	}
	if len(raw) > maxRegistryBytes {
		return nil, errors.New("node package registry exceeds its byte budget")
	}
	return raw, nil
}

func cloneGrants(source map[string]PackageTrustGrant) map[string]PackageTrustGrant {
	result := make(map[string]PackageTrustGrant, len(source))
	for key, grant := range source {
		result[key] = grant
	}
	return result
}

func cleanupInterrupted(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".incoming-") || strings.HasPrefix(entry.Name(), ".durable-") {
			if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
				return fmt.Errorf("clean interrupted node package incoming directory: %w", err)
			}
		}
	}
	return nil
}

func validateStoreRoot(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != generationsDir && entry.Name() != registryFilename {
			return fmt.Errorf("unexpected node package store entry %q", entry.Name())
		}
	}
	return nil
}

func cleanupOrphanGenerations(root string, installed map[string]PackageInstallation) error {
	referenced := make(map[string]struct{})
	for _, entry := range installed {
		for _, release := range entry.Releases {
			referenced[generationName(release.ManifestDigest)] = struct{}{}
		}
	}
	directory := filepath.Join(root, generationsDir)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() || !validGenerationName(entry.Name()) {
			return fmt.Errorf("unexpected node package generation store entry %q", entry.Name())
		}
		if _, found := referenced[entry.Name()]; !found {
			if err := os.RemoveAll(filepath.Join(directory, entry.Name())); err != nil {
				return fmt.Errorf("clean orphan node package generation: %w", err)
			}
		}
	}
	return nil
}

func (s *Store) generationPath(digest artifact.Digest) string { return generationPath(s.root, digest) }
func generationPath(root string, digest artifact.Digest) string {
	return filepath.Join(root, generationsDir, generationName(digest))
}
func generationName(digest artifact.Digest) string {
	return strings.TrimPrefix(digest.String(), "sha256:")
}
func validGenerationName(name string) bool {
	_, err := artifact.ParseDigest("sha256:" + name)
	return err == nil
}
func releaseIndex(releases []PackageRelease, digest artifact.Digest) int {
	for index := range releases {
		if releases[index].ManifestDigest == digest {
			return index
		}
	}
	return -1
}
func generationReferenced(entries map[string]PackageInstallation, digest artifact.Digest) bool {
	for _, installed := range entries {
		if releaseIndex(installed.Releases, digest) >= 0 {
			return true
		}
	}
	return false
}
func cloneInstallation(source PackageInstallation) PackageInstallation {
	clone := source
	clone.Releases = append([]PackageRelease(nil), source.Releases...)
	return clone
}
func cloneEntries(source map[string]PackageInstallation) map[string]PackageInstallation {
	clone := make(map[string]PackageInstallation, len(source))
	for key, installed := range source {
		clone[key] = cloneInstallation(installed)
	}
	return clone
}
