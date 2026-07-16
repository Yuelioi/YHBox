// Package nodepackage owns the immutable manifest at the trust boundary
// between an installed Node Package and Yotta's plugin hosts.
package nodepackage

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
	"strconv"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	Format         = "yotta.node-package"
	Version        = "1"
	MaxBytes       = 16 << 20
	MaxDefinitions = 4_096
	MaxPayloadSize = int64(16 << 30)

	manifestDigestDomain = "yotta/node-package-manifest/v1"
)

var (
	versionedURIPattern = regexp.MustCompile("/v[1-9][0-9]*$")
	semverPattern       = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	hostAPIPattern      = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	identityPattern     = regexp.MustCompile("^[a-z0-9][a-z0-9._-]{0,63}$")
	pathSegmentPattern  = regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
	entrypointPattern   = regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,255}$")
	mediaTypePattern    = regexp.MustCompile("^[a-z0-9][a-z0-9!#$&^_.+-]+/[a-z0-9][a-z0-9!#$&^_.+-]+$")
)

// HostAPIRange is half-open so a package cannot silently claim compatibility
// with a future breaking host generation.
type HostAPIRange struct {
	Min          string `json:"min"`
	MaxExclusive string `json:"maxExclusive"`
}

// Payload locks one archive file. Digest is raw SHA-256 over the exact bytes,
// matching BlobRef identity rather than a domain-separated contract hash.
type Payload struct {
	Path      string          `json:"path"`
	Digest    artifact.Digest `json:"digest"`
	Size      int64           `json:"size"`
	MediaType string          `json:"mediaType"`
}

type PlatformSupport struct {
	OperatingSystems []string `json:"operatingSystems"`
	Architectures    []string `json:"architectures"`
}

type Implementation struct {
	ABI        nodecontract.ABIRequirement `json:"abi"`
	Entrypoint string                      `json:"entrypoint"`
	Payload    Payload                     `json:"payload"`
	Platforms  PlatformSupport             `json:"platforms"`
}

type NodeDraft struct {
	Contract       nodecontract.Contract
	Implementation Implementation
}

type Draft struct {
	PublisherNamespace string
	PackageID          string
	PackageVersion     string
	HostAPI            HostAPIRange
	Types              []datatype.Definition
	Capabilities       []capability.Definition
	Nodes              []NodeDraft
	Documentation      []Payload
}

type typeRecord struct {
	TypeRef  datatype.TypeRef `json:"typeRef"`
	Semantic json.RawMessage  `json:"semantic"`
}

type capabilityRecord struct {
	Ref      capability.Ref  `json:"ref"`
	Semantic json.RawMessage `json:"semantic"`
}

type nodeRecord struct {
	NodeRef        nodecontract.NodeRef `json:"nodeRef"`
	Semantic       json.RawMessage      `json:"semantic"`
	Implementation Implementation       `json:"implementation"`
}

type document struct {
	Format             string             `json:"format"`
	Version            string             `json:"version"`
	PublisherNamespace string             `json:"publisherNamespace"`
	PackageID          string             `json:"packageId"`
	PackageVersion     string             `json:"packageVersion"`
	HostAPI            HostAPIRange       `json:"hostApi"`
	Types              []typeRecord       `json:"types"`
	Capabilities       []capabilityRecord `json:"capabilities"`
	Nodes              []nodeRecord       `json:"nodes"`
	Documentation      []Payload          `json:"documentation"`
}

type state struct {
	document     document
	digest       artifact.Digest
	bytes        []byte
	types        []datatype.Definition
	capabilities []capability.Definition
	nodes        []NodeDraft
}

// Manifest is immutable and canonical. Executable bytes stay outside JSON;
// path, size, media type, and digest lock every payload.
type Manifest struct{ state *state }

func Seal(draft Draft) (Manifest, error) {
	namespace, err := validateNamespace(draft.PublisherNamespace)
	if err != nil {
		return Manifest{}, err
	}
	if err := validateOwnedURI(namespace, draft.PackageID, "package ID"); err != nil {
		return Manifest{}, err
	}
	if !semverPattern.MatchString(draft.PackageVersion) {
		return Manifest{}, errors.New("package version must be strict SemVer")
	}
	if err := validateHostAPIRange(draft.HostAPI); err != nil {
		return Manifest{}, err
	}
	total := len(draft.Types) + len(draft.Capabilities) + len(draft.Nodes)
	if total == 0 || total > MaxDefinitions || len(draft.Documentation) > MaxDefinitions {
		return Manifest{}, errors.New("node package contribution count is invalid")
	}
	typeRecords, types, err := normalizeTypes(namespace, draft.Types)
	if err != nil {
		return Manifest{}, err
	}
	capabilityRecords, capabilities, err := normalizeCapabilities(namespace, draft.Capabilities)
	if err != nil {
		return Manifest{}, err
	}
	nodeRecords, nodes, paths, err := normalizeNodes(namespace, draft.Nodes)
	if err != nil {
		return Manifest{}, err
	}
	documentation, err := normalizeDocumentation(draft.Documentation, paths)
	if err != nil {
		return Manifest{}, err
	}
	doc := document{
		Format: Format, Version: Version, PublisherNamespace: draft.PublisherNamespace,
		PackageID: draft.PackageID, PackageVersion: draft.PackageVersion, HostAPI: draft.HostAPI,
		Types: typeRecords, Capabilities: capabilityRecords, Nodes: nodeRecords, Documentation: documentation,
	}
	return sealDocument(doc, types, capabilities, nodes)
}

func Open(raw []byte) (Manifest, error) {
	if len(raw) == 0 || len(raw) > MaxBytes {
		return Manifest{}, errors.New("node package manifest exceeds byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, 128, 1_048_576, 1<<20); err != nil {
		return Manifest{}, fmt.Errorf("inspect node package manifest: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return Manifest{}, errors.New("node package manifest is not canonical JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var decoded document
	if err := decoder.Decode(&decoded); err != nil {
		return Manifest{}, fmt.Errorf("decode node package manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Manifest{}, errors.New("node package manifest contains trailing JSON values")
	}
	if decoded.Format != Format || decoded.Version != Version {
		return Manifest{}, errors.New("unsupported node package manifest format")
	}
	types := make([]datatype.Definition, 0, len(decoded.Types))
	for _, record := range decoded.Types {
		definition, err := datatype.OpenSemanticDefinition(record.TypeRef, record.Semantic)
		if err != nil {
			return Manifest{}, fmt.Errorf("open package data type %q: %w", record.TypeRef.TypeID, err)
		}
		types = append(types, definition)
	}
	capabilities := make([]capability.Definition, 0, len(decoded.Capabilities))
	for _, record := range decoded.Capabilities {
		definition, err := capability.OpenSemanticDefinition(record.Ref, record.Semantic)
		if err != nil {
			return Manifest{}, fmt.Errorf("open package capability %q: %w", record.Ref.CapabilityID, err)
		}
		capabilities = append(capabilities, definition)
	}
	nodes := make([]NodeDraft, 0, len(decoded.Nodes))
	for _, record := range decoded.Nodes {
		contract, err := nodecontract.OpenSemantic(record.NodeRef, record.Semantic)
		if err != nil {
			return Manifest{}, fmt.Errorf("open package node %q: %w", record.NodeRef.NodeTypeID, err)
		}
		nodes = append(nodes, NodeDraft{Contract: contract, Implementation: record.Implementation})
	}
	sealed, err := Seal(Draft{
		PublisherNamespace: decoded.PublisherNamespace, PackageID: decoded.PackageID,
		PackageVersion: decoded.PackageVersion, HostAPI: decoded.HostAPI,
		Types: types, Capabilities: capabilities, Nodes: nodes, Documentation: decoded.Documentation,
	})
	if err != nil {
		return Manifest{}, err
	}
	if !bytes.Equal(sealed.Bytes(), raw) {
		return Manifest{}, errors.New("node package manifest is not normalized")
	}
	return sealed, nil
}

func sealDocument(doc document, types []datatype.Definition, capabilities []capability.Definition, nodes []NodeDraft) (Manifest, error) {
	raw, err := artifact.Marshal(doc)
	if err != nil {
		return Manifest{}, err
	}
	if len(raw) > MaxBytes {
		return Manifest{}, errors.New("node package manifest exceeds byte budget")
	}
	digest, err := artifact.Sum(manifestDigestDomain, raw)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{state: &state{
		document: doc, digest: digest, bytes: raw,
		types:        append([]datatype.Definition(nil), types...),
		capabilities: append([]capability.Definition(nil), capabilities...),
		nodes:        cloneNodes(nodes),
	}}, nil
}

func (m Manifest) Valid() bool { return m.state != nil && m.state.digest.Valid() }
func (m Manifest) Digest() artifact.Digest {
	if !m.Valid() {
		return ""
	}
	return m.state.digest
}
func (m Manifest) Bytes() []byte {
	if !m.Valid() {
		return nil
	}
	return append([]byte(nil), m.state.bytes...)
}
func (m Manifest) PackageID() string {
	if !m.Valid() {
		return ""
	}
	return m.state.document.PackageID
}
func (m Manifest) PackageVersion() string {
	if !m.Valid() {
		return ""
	}
	return m.state.document.PackageVersion
}
func (m Manifest) HostAPI() HostAPIRange {
	if !m.Valid() {
		return HostAPIRange{}
	}
	return m.state.document.HostAPI
}

func (m Manifest) SupportsHostAPI(generation string) bool {
	if !m.Valid() {
		return false
	}
	candidate, ok := parseHostAPI(generation)
	if !ok {
		return false
	}
	minimum, _ := parseHostAPI(m.state.document.HostAPI.Min)
	maximum, _ := parseHostAPI(m.state.document.HostAPI.MaxExclusive)
	return compareHostAPI(candidate, minimum) >= 0 && compareHostAPI(candidate, maximum) < 0
}
func (m Manifest) Types() []datatype.Definition {
	if !m.Valid() {
		return nil
	}
	return append([]datatype.Definition(nil), m.state.types...)
}
func (m Manifest) Capabilities() []capability.Definition {
	if !m.Valid() {
		return nil
	}
	return append([]capability.Definition(nil), m.state.capabilities...)
}
func (m Manifest) Nodes() []NodeDraft {
	if !m.Valid() {
		return nil
	}
	return cloneNodes(m.state.nodes)
}
func (m Manifest) Documentation() []Payload {
	if !m.Valid() {
		return nil
	}
	return append([]Payload(nil), m.state.document.Documentation...)
}

func normalizeTypes(namespace *url.URL, source []datatype.Definition) ([]typeRecord, []datatype.Definition, error) {
	ordered := append([]datatype.Definition(nil), source...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].TypeRef().TypeID < ordered[j].TypeRef().TypeID })
	records := make([]typeRecord, 0, len(ordered))
	previous := ""
	for _, definition := range ordered {
		if !definition.Valid() {
			return nil, nil, errors.New("node package contains an invalid data type")
		}
		ref := definition.TypeRef()
		if ref.TypeID == previous {
			return nil, nil, fmt.Errorf("duplicate package data type %q", ref.TypeID)
		}
		if err := validateOwnedURI(namespace, ref.TypeID, "data type ID"); err != nil {
			return nil, nil, err
		}
		previous = ref.TypeID
		records = append(records, typeRecord{TypeRef: ref, Semantic: definition.SemanticBytes()})
	}
	return records, ordered, nil
}

func normalizeCapabilities(namespace *url.URL, source []capability.Definition) ([]capabilityRecord, []capability.Definition, error) {
	ordered := append([]capability.Definition(nil), source...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Ref().CapabilityID < ordered[j].Ref().CapabilityID })
	records := make([]capabilityRecord, 0, len(ordered))
	previous := ""
	for _, definition := range ordered {
		if !definition.Valid() {
			return nil, nil, errors.New("node package contains an invalid capability")
		}
		ref := definition.Ref()
		if ref.CapabilityID == previous {
			return nil, nil, fmt.Errorf("duplicate package capability %q", ref.CapabilityID)
		}
		if err := validateOwnedURI(namespace, ref.CapabilityID, "capability ID"); err != nil {
			return nil, nil, err
		}
		previous = ref.CapabilityID
		records = append(records, capabilityRecord{Ref: ref, Semantic: definition.SemanticBytes()})
	}
	return records, ordered, nil
}

func normalizeNodes(namespace *url.URL, source []NodeDraft) ([]nodeRecord, []NodeDraft, map[string]Payload, error) {
	ordered := append([]NodeDraft(nil), source...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Contract.NodeRef().NodeTypeID < ordered[j].Contract.NodeRef().NodeTypeID
	})
	records := make([]nodeRecord, 0, len(ordered))
	paths := make(map[string]Payload)
	previous := ""
	for index := range ordered {
		node := &ordered[index]
		if !node.Contract.Valid() {
			return nil, nil, nil, errors.New("node package contains an invalid node contract")
		}
		ref := node.Contract.NodeRef()
		if ref.NodeTypeID == previous {
			return nil, nil, nil, fmt.Errorf("duplicate package node %q", ref.NodeTypeID)
		}
		if err := validateOwnedURI(namespace, ref.NodeTypeID, "node type ID"); err != nil {
			return nil, nil, nil, err
		}
		implementation, err := normalizeImplementation(node.Contract, node.Implementation)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("node %q: %w", ref.NodeTypeID, err)
		}
		if err := mergePayload(paths, implementation.Payload); err != nil {
			return nil, nil, nil, err
		}
		node.Implementation = implementation
		previous = ref.NodeTypeID
		records = append(records, nodeRecord{NodeRef: ref, Semantic: node.Contract.SemanticBytes(), Implementation: implementation})
	}
	return records, ordered, paths, nil
}

func normalizeImplementation(contract nodecontract.Contract, source Implementation) (Implementation, error) {
	if source.ABI.Kind != nodecontract.ABIWIT && source.ABI.Kind != nodecontract.ABIProcess {
		return Implementation{}, errors.New("third-party node implementation must use wit or process ABI")
	}
	allowed := false
	for _, requirement := range contract.Machine().ImplementationABI {
		if requirement == source.ABI {
			allowed = true
			break
		}
	}
	if !allowed {
		return Implementation{}, errors.New("implementation ABI is not allowed by the node contract")
	}
	if !entrypointPattern.MatchString(source.Entrypoint) {
		return Implementation{}, errors.New("implementation entrypoint is invalid")
	}
	payload, err := normalizePayload(source.Payload)
	if err != nil {
		return Implementation{}, err
	}
	platforms, err := normalizePlatforms(source.Platforms)
	if err != nil {
		return Implementation{}, err
	}
	return Implementation{ABI: source.ABI, Entrypoint: source.Entrypoint, Payload: payload, Platforms: platforms}, nil
}

func normalizeDocumentation(source []Payload, paths map[string]Payload) ([]Payload, error) {
	ordered := append([]Payload(nil), source...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for index := range ordered {
		payload, err := normalizePayload(ordered[index])
		if err != nil {
			return nil, fmt.Errorf("documentation: %w", err)
		}
		if !strings.HasPrefix(payload.MediaType, "text/") && payload.MediaType != "application/json" {
			return nil, errors.New("documentation payload has an unsupported media type")
		}
		if index > 0 && payload.Path == ordered[index-1].Path {
			return nil, errors.New("package contains a duplicate documentation path")
		}
		if err := mergePayload(paths, payload); err != nil {
			return nil, err
		}
		ordered[index] = payload
	}
	return ordered, nil
}

func normalizePayload(source Payload) (Payload, error) {
	if strings.EqualFold(source.Path, ArchiveManifestPath) || !validPortablePath(source.Path) ||
		path.Clean(source.Path) != source.Path || strings.HasPrefix(source.Path, "/") || strings.HasPrefix(source.Path, "../") {
		return Payload{}, errors.New("package payload path must be a clean relative slash path")
	}
	if !source.Digest.Valid() || source.Size <= 0 || source.Size > MaxPayloadSize || !mediaTypePattern.MatchString(source.MediaType) {
		return Payload{}, errors.New("package payload metadata is invalid")
	}
	return source, nil
}

func validPortablePath(value string) bool {
	if value == "" || value == "." || len(value) > 512 || strings.ContainsAny(value, "\\:") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if !pathSegmentPattern.MatchString(segment) || strings.HasSuffix(segment, ".") {
			return false
		}
		base := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" ||
			len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
			return false
		}
	}
	return true
}

func cloneNodes(source []NodeDraft) []NodeDraft {
	cloned := append([]NodeDraft(nil), source...)
	for index := range cloned {
		platforms := cloned[index].Implementation.Platforms
		platforms.OperatingSystems = append([]string(nil), platforms.OperatingSystems...)
		platforms.Architectures = append([]string(nil), platforms.Architectures...)
		cloned[index].Implementation.Platforms = platforms
	}
	return cloned
}

func mergePayload(paths map[string]Payload, payload Payload) error {
	key := strings.ToLower(payload.Path)
	if existing, found := paths[key]; found {
		if existing.Path != payload.Path {
			return fmt.Errorf("package payload paths %q and %q collide on case-insensitive filesystems", existing.Path, payload.Path)
		}
		if existing != payload {
			return fmt.Errorf("package payload path %q has conflicting identities", payload.Path)
		}
	}
	paths[key] = payload
	return nil
}

func normalizePlatforms(source PlatformSupport) (PlatformSupport, error) {
	operatingSystems, err := normalizeIdentities(source.OperatingSystems)
	if err != nil {
		return PlatformSupport{}, fmt.Errorf("operating systems: %w", err)
	}
	architectures, err := normalizeIdentities(source.Architectures)
	if err != nil {
		return PlatformSupport{}, fmt.Errorf("architectures: %w", err)
	}
	return PlatformSupport{OperatingSystems: operatingSystems, Architectures: architectures}, nil
}

func normalizeIdentities(source []string) ([]string, error) {
	if len(source) == 0 || len(source) > 32 {
		return nil, errors.New("platform identity set is empty or too large")
	}
	ordered := append([]string(nil), source...)
	sort.Strings(ordered)
	for index, value := range ordered {
		if !identityPattern.MatchString(value) || (index > 0 && value == ordered[index-1]) {
			return nil, errors.New("platform identity set is invalid")
		}
	}
	return ordered, nil
}

func validateNamespace(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" || parsed.Path == "" ||
		parsed.Host != strings.ToLower(parsed.Host) || path.Clean(parsed.Path) != parsed.Path || strings.HasSuffix(parsed.Path, "/") {
		return nil, errors.New("publisher namespace must be an HTTPS URI with a non-root path and no trailing slash")
	}
	return parsed, nil
}

func validateOwnedURI(namespace *url.URL, value, label string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != namespace.Scheme || parsed.Host != namespace.Host || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" || path.Clean(parsed.Path) != parsed.Path ||
		!versionedURIPattern.MatchString(parsed.Path) {
		return fmt.Errorf("%s is not a versioned URI in the publisher namespace", label)
	}
	if !strings.HasPrefix(parsed.EscapedPath(), namespace.EscapedPath()+"/") {
		return fmt.Errorf("%s is outside the publisher namespace", label)
	}
	return nil
}

func validateHostAPIRange(value HostAPIRange) error {
	minimum, ok := parseHostAPI(value.Min)
	if !ok {
		return errors.New("minimum host API generation is invalid")
	}
	maximum, ok := parseHostAPI(value.MaxExclusive)
	if !ok || compareHostAPI(minimum, maximum) >= 0 {
		return errors.New("maximum host API generation must be greater than the minimum")
	}
	return nil
}

func parseHostAPI(value string) ([2]uint64, bool) {
	if !hostAPIPattern.MatchString(value) {
		return [2]uint64{}, false
	}
	parts := strings.Split(value, ".")
	major, errMajor := strconv.ParseUint(parts[0], 10, 64)
	minor, errMinor := strconv.ParseUint(parts[1], 10, 64)
	return [2]uint64{major, minor}, errMajor == nil && errMinor == nil
}

func compareHostAPI(left, right [2]uint64) int {
	if left[0] < right[0] || left[0] == right[0] && left[1] < right[1] {
		return -1
	}
	if left == right {
		return 0
	}
	return 1
}
