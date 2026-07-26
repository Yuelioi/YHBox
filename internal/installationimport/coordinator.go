// Package installationimport materializes every exact artifact in one
// Installation Plan through a shared online/offline path. A staged Session is
// complete transport state only: it grants no publisher trust, installs no
// package, imports no Workflow, and grants no execution consent.
package installationimport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/installationplan"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/offlinepack"
)

type ArtifactSource func(
	context.Context,
	installationplan.ArtifactDescriptor,
) (io.ReadCloser, error)

type RedistributionCheck func(
	context.Context,
	installationplan.ArtifactDescriptor,
) error

type PackageStore interface {
	InspectArchiveTrust(context.Context, string) (nodepackage.ArchiveTrust, error)
	GrantPackageTrust(context.Context, artifact.Digest, string) error
	InstallArchive(context.Context, string) (nodepackage.PackageInstallation, error)
	Get(string) (nodepackage.PackageInstallation, bool)
}

type PackageCandidate struct {
	Release        installationplan.NodePackageRelease
	PublisherKeyID artifact.Digest
	TrustGranted   bool
	Installed      bool
}

type Session struct {
	mu        sync.RWMutex
	root      string
	plan      installationplan.Plan
	artifacts map[artifact.Digest]string
}

func StageOnline(
	ctx context.Context,
	parent string,
	plan installationplan.Plan,
	open ArtifactSource,
) (*Session, error) {
	if open == nil {
		return nil, errors.New("online installation import requires an artifact source")
	}
	return stage(ctx, parent, plan, open)
}

func StageOffline(ctx context.Context, parent, archivePath string) (*Session, error) {
	pack, err := offlinepack.Open(ctx, archivePath)
	if err != nil {
		return nil, err
	}
	session, stageErr := stage(ctx, parent, pack.Plan(), pack.OpenArtifact)
	closeErr := pack.Close()
	if stageErr != nil {
		return nil, stageErr
	}
	if closeErr != nil {
		_ = session.Close()
		return nil, fmt.Errorf("close offline pack after staging: %w", closeErr)
	}
	return session, nil
}

// WriteOfflinePack is the completeness gate above the transport writer.
// Every exact artifact must still be available and explicitly redistributable;
// a missing, delisted, or forbidden release must be returned as an error by
// check, and no pack will be published.
func WriteOfflinePack(
	ctx context.Context,
	destination string,
	plan installationplan.Plan,
	check RedistributionCheck,
	open ArtifactSource,
) error {
	if ctx == nil || !plan.Valid() || check == nil || open == nil {
		return errors.New("offline installation export request is invalid")
	}
	descriptors, err := planArtifacts(plan)
	if err != nil {
		return err
	}
	for _, descriptor := range descriptors {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := check(ctx, descriptor); err != nil {
			return fmt.Errorf(
				"installation artifact %s is not eligible for complete offline redistribution: %w",
				descriptor.Digest,
				err,
			)
		}
	}
	return offlinepack.Write(ctx, destination, plan, offlinepack.ArtifactSource(open))
}

func stage(
	ctx context.Context,
	parent string,
	plan installationplan.Plan,
	open ArtifactSource,
) (*Session, error) {
	if ctx == nil || !plan.Valid() || strings.TrimSpace(parent) == "" || open == nil {
		return nil, errors.New("installation import staging request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parent, err := filepath.Abs(parent)
	if err != nil {
		return nil, err
	}
	parentInfo, err := os.Stat(parent)
	if err != nil || !parentInfo.IsDir() {
		return nil, errors.New("installation import staging parent does not exist")
	}
	descriptors, err := planArtifacts(plan)
	if err != nil {
		return nil, err
	}
	root, err := os.MkdirTemp(parent, ".yotta-installation-import-")
	if err != nil {
		return nil, err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(root)
		}
	}()
	paths := make(map[artifact.Digest]string, len(descriptors))
	for _, descriptor := range descriptors {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		reader, err := open(ctx, descriptor)
		if err != nil || reader == nil {
			if err == nil {
				err = errors.New("artifact source returned a nil reader")
			}
			return nil, fmt.Errorf("open installation artifact %s: %w", descriptor.Digest, err)
		}
		destination := filepath.Join(root, strings.TrimPrefix(descriptor.Digest.String(), "sha256:"))
		if err := copyArtifact(ctx, destination, descriptor, reader); err != nil {
			return nil, err
		}
		paths[descriptor.Digest] = destination
	}
	complete = true
	return &Session{root: root, plan: plan, artifacts: paths}, nil
}

func (s *Session) Plan() installationplan.Plan {
	if s == nil {
		return installationplan.Plan{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.plan
}

func (s *Session) WorkflowArtifact(ctx context.Context) (string, error) {
	if s == nil {
		return "", errors.New("installation import session is invalid")
	}
	plan := s.Plan()
	if !plan.Valid() {
		return "", errors.New("installation import session is closed")
	}
	return s.verifiedPath(ctx, plan.Machine().Workflow.Artifact)
}

func (s *Session) InspectPackages(
	ctx context.Context,
	store PackageStore,
) ([]PackageCandidate, error) {
	if s == nil || store == nil {
		return nil, errors.New("package inspection request is invalid")
	}
	plan := s.Plan()
	if !plan.Valid() {
		return nil, errors.New("installation import session is closed")
	}
	releases := plan.Machine().Packages
	result := make([]PackageCandidate, 0, len(releases))
	for _, release := range releases {
		candidate, err := s.inspectPackage(ctx, store, release)
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	return result, nil
}

// GrantPublisherTrust is the explicit package-scoped trust action. It does not
// install the package whose signature was inspected.
func (s *Session) GrantPublisherTrust(
	ctx context.Context,
	store PackageStore,
	packageID string,
) (PackageCandidate, error) {
	release, err := s.packageRelease(packageID)
	if err != nil {
		return PackageCandidate{}, err
	}
	candidate, err := s.inspectPackage(ctx, store, release)
	if err != nil {
		return PackageCandidate{}, err
	}
	if err := store.GrantPackageTrust(ctx, candidate.PublisherKeyID, release.PackageID); err != nil {
		return PackageCandidate{}, err
	}
	candidate.TrustGranted = true
	return candidate, nil
}

// InstallPackage is a separate explicit action. It requires the exact
// publisherKey+packageId trust grant and installs only the selected release.
func (s *Session) InstallPackage(
	ctx context.Context,
	store PackageStore,
	packageID string,
) (nodepackage.PackageInstallation, error) {
	release, err := s.packageRelease(packageID)
	if err != nil {
		return nodepackage.PackageInstallation{}, err
	}
	candidate, err := s.inspectPackage(ctx, store, release)
	if err != nil {
		return nodepackage.PackageInstallation{}, err
	}
	if !candidate.TrustGranted {
		return nodepackage.PackageInstallation{}, errors.New("package installation requires its separate publisher trust action")
	}
	archivePath, err := s.verifiedPath(ctx, release.Artifact)
	if err != nil {
		return nodepackage.PackageInstallation{}, err
	}
	installed, err := store.InstallArchive(ctx, archivePath)
	if err != nil {
		return nodepackage.PackageInstallation{}, err
	}
	if installed.PackageID != release.PackageID ||
		installed.PublisherNamespace != release.PublisherNamespace ||
		installed.Current != release.ManifestDigest {
		return nodepackage.PackageInstallation{}, errors.New("installed Node Package identity does not match the Installation Plan")
	}
	return installed, nil
}

func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.root == "" {
		return nil
	}
	root := s.root
	s.root = ""
	s.plan = installationplan.Plan{}
	s.artifacts = nil
	return os.RemoveAll(root)
}

func (s *Session) packageRelease(packageID string) (installationplan.NodePackageRelease, error) {
	if s == nil || strings.TrimSpace(packageID) != packageID || packageID == "" {
		return installationplan.NodePackageRelease{}, errors.New("package selection is invalid")
	}
	plan := s.Plan()
	if !plan.Valid() {
		return installationplan.NodePackageRelease{}, errors.New("installation import session is closed")
	}
	releases := plan.Machine().Packages
	index := sort.Search(len(releases), func(index int) bool {
		return releases[index].PackageID >= packageID
	})
	if index == len(releases) || releases[index].PackageID != packageID {
		return installationplan.NodePackageRelease{}, errors.New("package is not declared by the Installation Plan")
	}
	return releases[index], nil
}

func (s *Session) inspectPackage(
	ctx context.Context,
	store PackageStore,
	release installationplan.NodePackageRelease,
) (PackageCandidate, error) {
	if store == nil {
		return PackageCandidate{}, errors.New("package store is required")
	}
	archivePath, err := s.verifiedPath(ctx, release.Artifact)
	if err != nil {
		return PackageCandidate{}, err
	}
	trust, err := store.InspectArchiveTrust(ctx, archivePath)
	if err != nil {
		return PackageCandidate{}, err
	}
	if trust.PublisherNamespace != release.PublisherNamespace ||
		trust.PackageID != release.PackageID ||
		trust.PackageVersion != release.PackageVersion ||
		trust.ManifestDigest != release.ManifestDigest {
		return PackageCandidate{}, errors.New("signed Node Package identity does not match the Installation Plan")
	}
	installed := false
	if current, found := store.Get(release.PackageID); found {
		for _, candidate := range current.Releases {
			if candidate.ManifestDigest == release.ManifestDigest &&
				candidate.PackageVersion == release.PackageVersion {
				installed = true
				break
			}
		}
	}
	return PackageCandidate{
		Release: release, PublisherKeyID: trust.PublisherKeyID,
		TrustGranted: trust.Granted, Installed: installed,
	}, nil
}

func (s *Session) verifiedPath(
	ctx context.Context,
	descriptor installationplan.ArtifactDescriptor,
) (string, error) {
	if s == nil || ctx == nil {
		return "", errors.New("installation artifact verification request is invalid")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	path, found := s.artifacts[descriptor.Digest]
	if s.root == "" || !found {
		return "", errors.New("installation artifact is unavailable")
	}
	if err := verifyFile(ctx, path, descriptor); err != nil {
		return "", err
	}
	return path, nil
}

func copyArtifact(
	ctx context.Context,
	destination string,
	descriptor installationplan.ArtifactDescriptor,
	reader io.ReadCloser,
) error {
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		_ = reader.Close()
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(
		io.MultiWriter(file, hash),
		io.LimitReader(&contextReader{ctx: ctx, reader: reader}, descriptor.Size+1),
	)
	readerCloseErr := reader.Close()
	fileCloseErr := file.Close()
	if copyErr != nil || readerCloseErr != nil || fileCloseErr != nil ||
		written != descriptor.Size || rawDigest(hash.Sum(nil)) != descriptor.Digest {
		_ = os.Remove(destination)
		return fmt.Errorf("installation artifact %s changed identity while staging", descriptor.Digest)
	}
	return nil
}

func verifyFile(
	ctx context.Context,
	path string,
	descriptor installationplan.ArtifactDescriptor,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() != descriptor.Size {
		return fmt.Errorf("staged installation artifact %s is invalid", descriptor.Digest)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(hash, &contextReader{ctx: ctx, reader: file})
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written != descriptor.Size ||
		rawDigest(hash.Sum(nil)) != descriptor.Digest {
		return fmt.Errorf("staged installation artifact %s changed identity", descriptor.Digest)
	}
	return nil
}

func planArtifacts(plan installationplan.Plan) ([]installationplan.ArtifactDescriptor, error) {
	machine := plan.Machine()
	descriptors := make([]installationplan.ArtifactDescriptor, 0, len(machine.Packages)+1)
	descriptors = append(descriptors, machine.Workflow.Artifact)
	for _, packageRelease := range machine.Packages {
		descriptors = append(descriptors, packageRelease.Artifact)
	}
	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].Digest < descriptors[j].Digest
	})
	unique := descriptors[:0]
	for _, descriptor := range descriptors {
		if len(unique) != 0 && unique[len(unique)-1].Digest == descriptor.Digest {
			if unique[len(unique)-1] != descriptor {
				return nil, errors.New("Installation Plan contains conflicting artifact descriptors")
			}
			continue
		}
		unique = append(unique, descriptor)
	}
	return unique, nil
}

func rawDigest(sum []byte) artifact.Digest {
	return artifact.Digest("sha256:" + hex.EncodeToString(sum))
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(target []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(target)
}
