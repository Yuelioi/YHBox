// Package installationplan owns the transport-neutral list of exact release
// artifacts needed to install one Workflow Release. It intentionally does not
// contain or grant local trust, package installation, target, credential, or
// execution-consent state.
package installationplan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	Format                       = "yotta.installation-plan"
	Version                      = "1"
	WorkflowArtifactMediaType    = "application/vnd.yotta.workflow+zip"
	NodePackageArtifactMediaType = "application/vnd.yotta.node-package+zip"
	maxPlanBytes                 = 8 << 20
	maxArtifactBytes             = 1 << 30
	identityDomain               = "yotta/installation-plan/v1"
)

var (
	workflowIDPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$`)
	semverPattern     = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	versionedPath     = regexp.MustCompile(`/v(?:0|[1-9][0-9]*)$`)
)

type ArtifactDescriptor struct {
	Digest    artifact.Digest `json:"digest"`
	Size      int64           `json:"size"`
	MediaType string          `json:"mediaType"`
}

type WorkflowRelease struct {
	PublisherNamespace string             `json:"publisherNamespace"`
	WorkflowID         string             `json:"workflowId"`
	ReleaseVersion     string             `json:"releaseVersion"`
	ReleaseDigest      artifact.Digest    `json:"releaseDigest"`
	SourceHash         artifact.Digest    `json:"sourceHash"`
	Artifact           ArtifactDescriptor `json:"artifact"`
}

type NodePackageRelease struct {
	PublisherNamespace string             `json:"publisherNamespace"`
	PackageID          string             `json:"packageId"`
	PackageVersion     string             `json:"packageVersion"`
	ManifestDigest     artifact.Digest    `json:"manifestDigest"`
	Artifact           ArtifactDescriptor `json:"artifact"`
}

type Draft struct {
	Workflow WorkflowRelease
	Packages []NodePackageRelease
}

type document struct {
	Format   string               `json:"format"`
	Version  string               `json:"version"`
	Workflow WorkflowRelease      `json:"workflow"`
	Packages []NodePackageRelease `json:"packages"`
}

type state struct {
	document document
	bytes    []byte
	digest   artifact.Digest
}

type Plan struct{ state *state }

// SealForSource creates a deterministic plan only when its Workflow and Node
// Package release identities match one canonical Workflow Source exactly.
func SealForSource(sourceArtifact []byte, draft Draft) (Plan, error) {
	source, canonical, sourceHash, diagnostics, err := schema.CanonicalSource(sourceArtifact)
	if err != nil {
		return Plan{}, fmt.Errorf("open Installation Plan Workflow Source: %w", err)
	}
	if schema.HasErrors(diagnostics) || !bytes.Equal(sourceArtifact, canonical) {
		return Plan{}, errors.New("Installation Plan requires canonical valid Workflow Source bytes")
	}
	packages := append([]NodePackageRelease(nil), draft.Packages...)
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].PackageID < packages[j].PackageID
	})
	doc := document{
		Format: Format, Version: Version, Workflow: draft.Workflow, Packages: packages,
	}
	if err := validateDocument(doc); err != nil {
		return Plan{}, err
	}
	if draft.Workflow.WorkflowID != source.Workflow.ID || draft.Workflow.SourceHash != sourceHash {
		return Plan{}, errors.New("Installation Plan Workflow identity does not match its Source")
	}
	if err := matchDependencies(source.Dependencies, packages); err != nil {
		return Plan{}, err
	}
	return sealDocument(doc)
}

func Open(raw []byte) (Plan, error) {
	if len(raw) == 0 || len(raw) > maxPlanBytes {
		return Plan{}, errors.New("Installation Plan exceeds its byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, 8, 65_536, maxPlanBytes); err != nil {
		return Plan{}, fmt.Errorf("inspect Installation Plan: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return Plan{}, err
	}
	if !bytes.Equal(raw, canonical) {
		return Plan{}, errors.New("Installation Plan is not canonical JSON")
	}
	var doc document
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return Plan{}, fmt.Errorf("decode Installation Plan: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return Plan{}, err
	}
	if err := validateDocument(doc); err != nil {
		return Plan{}, err
	}
	for index := 1; index < len(doc.Packages); index++ {
		if comparePackages(doc.Packages[index-1], doc.Packages[index]) >= 0 {
			return Plan{}, errors.New("Installation Plan packages are not uniquely sorted")
		}
	}
	return sealDocument(doc)
}

func (p Plan) Valid() bool {
	return p.state != nil && p.state.digest.Valid() && len(p.state.bytes) != 0
}

func (p Plan) Digest() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.digest
}

func (p Plan) Bytes() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.bytes...)
}

func (p Plan) Machine() Draft {
	if !p.Valid() {
		return Draft{}
	}
	return Draft{
		Workflow: p.state.document.Workflow,
		Packages: append([]NodePackageRelease(nil), p.state.document.Packages...),
	}
}

func (p Plan) ValidateSource(sourceArtifact []byte) error {
	if !p.Valid() {
		return errors.New("Installation Plan is invalid")
	}
	source, canonical, sourceHash, diagnostics, err := schema.CanonicalSource(sourceArtifact)
	if err != nil || schema.HasErrors(diagnostics) || !bytes.Equal(sourceArtifact, canonical) {
		return errors.New("Installation Plan Source is not canonical and valid")
	}
	workflow := p.state.document.Workflow
	if workflow.WorkflowID != source.Workflow.ID || workflow.SourceHash != sourceHash {
		return errors.New("Installation Plan Workflow identity does not match its Source")
	}
	return matchDependencies(source.Dependencies, p.state.document.Packages)
}

func sealDocument(doc document) (Plan, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return Plan{}, err
	}
	raw, err = artifact.Canonicalize(raw)
	if err != nil {
		return Plan{}, err
	}
	if len(raw) > maxPlanBytes {
		return Plan{}, errors.New("Installation Plan exceeds its byte budget")
	}
	digest, err := artifact.Sum(identityDomain, raw)
	if err != nil {
		return Plan{}, err
	}
	return Plan{state: &state{document: doc, bytes: raw, digest: digest}}, nil
}

func validateDocument(doc document) error {
	if doc.Format != Format || doc.Version != Version || doc.Packages == nil ||
		len(doc.Packages) > schema.MaxDependencies {
		return errors.New("Installation Plan identity or package count is invalid")
	}
	workflow := doc.Workflow
	if err := validatePublisherNamespace(workflow.PublisherNamespace); err != nil ||
		!workflowIDPattern.MatchString(workflow.WorkflowID) ||
		!semverPattern.MatchString(workflow.ReleaseVersion) ||
		!workflow.ReleaseDigest.Valid() || !workflow.SourceHash.Valid() {
		return errors.New("Installation Plan Workflow Release is invalid")
	}
	if err := validateArtifact(workflow.Artifact, WorkflowArtifactMediaType); err != nil {
		return fmt.Errorf("Installation Plan Workflow artifact: %w", err)
	}
	for index, packageRelease := range doc.Packages {
		namespace, err := parsePublisherNamespace(packageRelease.PublisherNamespace)
		if err != nil || validateOwnedPackageID(namespace, packageRelease.PackageID) != nil ||
			!semverPattern.MatchString(packageRelease.PackageVersion) ||
			!packageRelease.ManifestDigest.Valid() {
			return fmt.Errorf("Installation Plan Node Package release %d is invalid", index)
		}
		if err := validateArtifact(packageRelease.Artifact, NodePackageArtifactMediaType); err != nil {
			return fmt.Errorf("Installation Plan Node Package artifact %d: %w", index, err)
		}
	}
	return nil
}

func validateArtifact(descriptor ArtifactDescriptor, expectedMediaType string) error {
	if !descriptor.Digest.Valid() || descriptor.Size <= 0 || descriptor.Size > maxArtifactBytes ||
		descriptor.MediaType != expectedMediaType {
		return errors.New("descriptor is invalid")
	}
	return nil
}

func matchDependencies(
	dependencies []schema.NodePackageDependency,
	packages []NodePackageRelease,
) error {
	if len(dependencies) != len(packages) {
		return errors.New("Installation Plan does not contain the exact Source dependency set")
	}
	for index, dependency := range dependencies {
		packageRelease := packages[index]
		if dependency.PublisherNamespace != packageRelease.PublisherNamespace ||
			dependency.PackageID != packageRelease.PackageID ||
			dependency.PackageVersion != packageRelease.PackageVersion ||
			dependency.ManifestDigest != packageRelease.ManifestDigest {
			return errors.New("Installation Plan Node Package identity does not match its Source dependency")
		}
	}
	return nil
}

func comparePackages(left, right NodePackageRelease) int {
	return strings.Compare(left.PackageID, right.PackageID)
}

func validatePublisherNamespace(value string) error {
	_, err := parsePublisherNamespace(value)
	return err
}

func parsePublisherNamespace(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" || parsed.Path == "" ||
		parsed.Host != strings.ToLower(parsed.Host) || path.Clean(parsed.Path) != parsed.Path ||
		strings.HasSuffix(parsed.Path, "/") {
		return nil, errors.New("publisher namespace is invalid")
	}
	return parsed, nil
}

func validateOwnedPackageID(namespace *url.URL, value string) error {
	if namespace == nil {
		return errors.New("publisher namespace is invalid")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != namespace.Scheme || parsed.Host != namespace.Host ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" ||
		path.Clean(parsed.Path) != parsed.Path || !versionedPath.MatchString(parsed.Path) ||
		!strings.HasPrefix(parsed.EscapedPath(), namespace.EscapedPath()+"/") {
		return errors.New("package ID is outside its publisher namespace")
	}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("Installation Plan contains trailing JSON values")
		}
		return err
	}
	return nil
}
