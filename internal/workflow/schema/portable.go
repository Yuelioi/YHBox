package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	runtimejsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/contractschema"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	MacroResourceMediaType     = "application/vnd.yotta.macro+json"
	InputClipResourceMediaType = "application/vnd.yotta.input-clip"
)

var (
	strictSemverPattern  = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	versionedPathPattern = regexp.MustCompile(`/v[1-9][0-9]*$`)
	windowsDrivePath     = regexp.MustCompile(`(?i)^[a-z]:[\\/]`)
	forbiddenTargetKeys  = map[string]struct{}{
		"applicationpath": {}, "executablepath": {}, "hwnd": {}, "pid": {}, "processid": {},
		"deviceserial": {}, "adbserial": {}, "browsertargetid": {}, "browserwebsocketurl": {},
		"credential": {}, "credentialid": {}, "secret": {}, "token": {}, "consent": {}, "providerauthority": {},
	}
)

func validateResources(resources []WorkflowResource) error {
	if len(resources) > MaxResources {
		return fmt.Errorf("resource count exceeds %d", MaxResources)
	}
	previous := ""
	for index, resource := range resources {
		if resource.ID <= previous {
			return errors.New("resources must be uniquely sorted by id")
		}
		if resource.Name == "" || strings.TrimSpace(resource.Name) != resource.Name || !validSortedTextSet(resource.Tags) {
			return fmt.Errorf("resource %d presentation metadata is invalid", index)
		}
		shapeCount := boolCount(resource.Image != nil, resource.Macro != nil, resource.InputClip != nil)
		if shapeCount != 1 {
			return fmt.Errorf("resource %q must contain exactly one kind payload", resource.ID)
		}
		var err error
		switch resource.Kind {
		case ResourceImage:
			if resource.Image == nil {
				err = errors.New("image payload is missing")
			} else {
				err = validateImageResource(*resource.Image)
			}
		case ResourceMacro:
			if resource.Macro == nil {
				err = errors.New("macro payload is missing")
			} else {
				err = validateMacroResource(*resource.Macro)
			}
		case ResourceInputClip:
			if resource.InputClip == nil {
				err = errors.New("input clip payload is missing")
			} else {
				err = validateInputClipResource(*resource.InputClip)
			}
		default:
			err = errors.New("resource kind is invalid")
		}
		if err != nil {
			return fmt.Errorf("resource %q: %w", resource.ID, err)
		}
		previous = resource.ID
	}
	return nil
}

// ValidateWorkflowResource validates one portable resource independently of a
// Source document. Authoring RPCs use it before accepting untrusted resource
// values from the frontend.
func ValidateWorkflowResource(resource WorkflowResource) error {
	return validateResources([]WorkflowResource{resource})
}

func validateImageResource(resource ImageResource) error {
	if len(resource.Variants) == 0 || len(resource.Variants) > 256 {
		return errors.New("image variant count is invalid")
	}
	previous := ""
	for _, variant := range resource.Variants {
		if variant.ID <= previous || !validResolution(variant.Resolution) || !validBBox(variant.BBox, variant.Resolution) ||
			len(variant.Regions) > 256 || variant.Blob.Validate() != nil || !strings.HasPrefix(variant.Blob.MediaType, "image/") {
			return errors.New("image variants must be valid and uniquely sorted by id")
		}
		for _, region := range variant.Regions {
			if !validBBox(region, variant.Resolution) {
				return errors.New("image variant region is outside its resolution")
			}
		}
		previous = variant.ID
	}
	return nil
}

func validateMacroResource(resource MacroResource) error {
	if resource.Blob.Validate() != nil || resource.Blob.MediaType != MacroResourceMediaType || !validResolution(resource.BaseResolution) ||
		resource.ActionCount < 0 || resource.ActionCount > 4096 || resource.DurationUs > 3_600_000_000 {
		return errors.New("macro metadata is invalid")
	}
	return nil
}

func validateInputClipResource(resource InputClipResource) error {
	if resource.Blob.Validate() != nil || resource.Blob.MediaType != InputClipResourceMediaType ||
		resource.DurationUs > 3_600_000_000 || resource.EventCount < 0 || resource.EventCount > 10_000_000 ||
		(resource.RecordingMode != "simple" && resource.RecordingMode != "precise") ||
		(resource.MouseMode != "relative" && resource.MouseMode != "absolute" && resource.MouseMode != "mixed") ||
		!validResolution(resource.BaseResolution) || resource.MouseCounts360 < 0 || resource.MouseCounts360 > 10_000_000 ||
		resource.StopHotkeyVK > 255 {
		return errors.New("input clip metadata is invalid")
	}
	return nil
}

func validateTargetProfileDefinitions(definitions []TargetProfileDefinition) error {
	if len(definitions) > MaxTargetDefaults {
		return fmt.Errorf("target profile definition count exceeds %d", MaxTargetDefaults)
	}
	previous := ""
	for _, definition := range definitions {
		if definition.ID <= previous || !validTargetDefaultName(definition.ID) || definition.Name == "" ||
			strings.TrimSpace(definition.Name) != definition.Name || !validTargetDefaultName(definition.TargetKind) ||
			!validTargetDefaultName(definition.AdapterKind) {
			return errors.New("target profile definitions must be valid and uniquely sorted by id")
		}
		normalized, err := contractschema.Normalize(datatype.JSONSchemaDialect, definition.SettingsSchemaRoot, definition.SettingsSchemaBundle)
		if err != nil || !equalSchemaResources(normalized, definition.SettingsSchemaBundle) {
			return fmt.Errorf("target profile definition %q settings schema is invalid or not normalized", definition.ID)
		}
		if err := validatePortableTargetDefaults(definition.InitialDefaults); err != nil {
			return fmt.Errorf("target profile definition %q defaults: %w", definition.ID, err)
		}
		if err := validateTargetDefaultsSchema(definition.SettingsSchemaRoot, normalized, definition.InitialDefaults); err != nil {
			return fmt.Errorf("target profile definition %q defaults do not satisfy settings schema: %w", definition.ID, err)
		}
		if !validDiscoveryHints(definition.DiscoveryHints) {
			return fmt.Errorf("target profile definition %q discovery hints are invalid", definition.ID)
		}
		previous = definition.ID
	}
	return nil
}

func validateTargetDefaultReferences(defaults []TargetDefault, definitions []TargetProfileDefinition) error {
	known := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		known[definition.ID] = struct{}{}
	}
	for _, candidate := range defaults {
		if _, ok := known[candidate.Slot]; !ok {
			return fmt.Errorf("target default %q references unknown profile definition %q", candidate.Target, candidate.Slot)
		}
	}
	return nil
}

func validateCredentialRequirements(requirements []CredentialRequirement) error {
	if len(requirements) > MaxCredentials {
		return fmt.Errorf("credential requirement count exceeds %d", MaxCredentials)
	}
	previous := ""
	for _, requirement := range requirements {
		if requirement.Slot <= previous || !validTargetDefaultName(requirement.Slot) ||
			validateVersionedHTTPSURI(requirement.Kind) != nil || requirement.Purpose == "" ||
			strings.TrimSpace(requirement.Purpose) != requirement.Purpose {
			return errors.New("credential requirements must be valid and uniquely sorted by slot")
		}
		previous = requirement.Slot
	}
	return nil
}

func validateDependencies(dependencies []NodePackageDependency, used map[nodecontract.NodeRef]struct{}) error {
	if len(dependencies) > MaxDependencies {
		return fmt.Errorf("node package dependency count exceeds %d", MaxDependencies)
	}
	previousPackage := ""
	claimed := make(map[nodecontract.NodeRef]struct{})
	for _, dependency := range dependencies {
		namespace, err := validatePublisherNamespace(dependency.PublisherNamespace)
		if err != nil || dependency.PackageID <= previousPackage || validateOwnedVersionedURI(namespace, dependency.PackageID) != nil ||
			!strictSemverPattern.MatchString(dependency.PackageVersion) || !dependency.ManifestDigest.Valid() || len(dependency.NodeRefs) == 0 {
			return errors.New("node package dependencies must be exact and uniquely sorted by package id")
		}
		previousRef := nodecontract.NodeRef{}
		for index, ref := range dependency.NodeRefs {
			if index > 0 && compareNodeRef(previousRef, ref) >= 0 {
				return fmt.Errorf("dependency %q node refs must be uniquely sorted", dependency.PackageID)
			}
			if _, ok := used[ref]; !ok {
				return fmt.Errorf("dependency %q references a node not used by the workflow", dependency.PackageID)
			}
			if _, duplicate := claimed[ref]; duplicate {
				return fmt.Errorf("node ref %q is claimed by multiple dependencies", ref.NodeTypeID)
			}
			claimed[ref] = struct{}{}
			previousRef = ref
		}
		previousPackage = dependency.PackageID
	}
	return nil
}

// ResolveResourceBinding resolves portable authoring identity to the exact
// immutable BlobRef consumed by compilation. Global Asset records never take
// part in this lookup.
func ResolveResourceBinding(resources []WorkflowResource, binding ResourceBinding) (blob.BlobRef, error) {
	index := resourceIndex(resources)
	resource, ok := index[binding.ResourceID]
	if !ok {
		return blob.BlobRef{}, fmt.Errorf("unknown workflow resource %q", binding.ResourceID)
	}
	switch resource.Kind {
	case ResourceImage:
		if resource.Image == nil || binding.VariantID == "" {
			return blob.BlobRef{}, errors.New("image resource binding requires a variant id")
		}
		for _, variant := range resource.Image.Variants {
			if variant.ID == binding.VariantID {
				return variant.Blob, nil
			}
		}
		return blob.BlobRef{}, fmt.Errorf("unknown image resource variant %q", binding.VariantID)
	case ResourceMacro:
		if resource.Macro == nil || binding.VariantID != "" {
			return blob.BlobRef{}, errors.New("macro resource binding cannot select a variant")
		}
		return resource.Macro.Blob, nil
	case ResourceInputClip:
		if resource.InputClip == nil || binding.VariantID != "" {
			return blob.BlobRef{}, errors.New("input clip resource binding cannot select a variant")
		}
		return resource.InputClip.Blob, nil
	default:
		return blob.BlobRef{}, errors.New("workflow resource kind is invalid")
	}
}

// BlobReferences inventories every immutable payload needed to retain,
// export, or verify a Workflow Source. It hides whether a caller chose a raw
// Blob binding or a domain-rich Workflow Resource.
func BlobReferences(source WorkflowSource) ([]blob.BlobRef, error) {
	byDigest := make(map[artifact.Digest]blob.BlobRef)
	add := func(ref blob.BlobRef) error {
		if err := ref.Validate(); err != nil {
			return err
		}
		if previous, exists := byDigest[ref.Digest]; exists && previous != ref {
			return fmt.Errorf("blob digest %s has conflicting metadata", ref.Digest)
		}
		byDigest[ref.Digest] = ref
		return nil
	}
	for _, resource := range source.Resources {
		switch resource.Kind {
		case ResourceImage:
			if resource.Image != nil {
				for _, variant := range resource.Image.Variants {
					if err := add(variant.Blob); err != nil {
						return nil, err
					}
				}
			}
		case ResourceMacro:
			if resource.Macro != nil {
				if err := add(resource.Macro.Blob); err != nil {
					return nil, err
				}
			}
		case ResourceInputClip:
			if resource.InputClip != nil {
				if err := add(resource.InputClip.Blob); err != nil {
					return nil, err
				}
			}
		}
	}
	collect := func(bindings map[string]InputBinding) error {
		for _, binding := range bindings {
			if binding.Kind == BindingBlob && binding.Blob != nil {
				if err := add(*binding.Blob); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, graph := range source.Graphs {
		for _, node := range graph.Nodes {
			if err := collect(node.Bindings); err != nil {
				return nil, err
			}
		}
		for _, call := range graph.Calls {
			if err := collect(call.Bindings); err != nil {
				return nil, err
			}
		}
	}
	result := make([]blob.BlobRef, 0, len(byDigest))
	for _, ref := range byDigest {
		result = append(result, ref)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Digest < result[j].Digest })
	return result, nil
}

func resourceIndex(resources []WorkflowResource) map[string]WorkflowResource {
	result := make(map[string]WorkflowResource, len(resources))
	for _, resource := range resources {
		result[resource.ID] = resource
	}
	return result
}

func validatePortableTargetDefaults(raw json.RawMessage) error {
	if len(raw) == 0 {
		return errors.New("initial defaults are required")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if _, ok := value.(map[string]any); !ok {
		return errors.New("initial defaults must be a JSON object")
	}
	return inspectPortableTargetValue(value)
}

func validateTargetDefaultsSchema(root string, resources []datatype.SchemaResource, raw json.RawMessage) error {
	compiler := runtimejsonschema.NewCompiler()
	for _, resource := range resources {
		decoder := json.NewDecoder(bytes.NewReader(resource.Schema))
		decoder.UseNumber()
		var document any
		if err := decoder.Decode(&document); err != nil {
			return err
		}
		if err := compiler.AddResource(resource.ID, document); err != nil {
			return err
		}
	}
	validator, err := compiler.Compile(root)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	return validator.Validate(value)
}

func inspectPortableTargetValue(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			canonicalKey := strings.Map(func(r rune) rune {
				if r == '-' || r == '_' || r == '.' {
					return -1
				}
				return unicode.ToLower(r)
			}, key)
			if _, forbidden := forbiddenTargetKeys[canonicalKey]; forbidden {
				return fmt.Errorf("field %q belongs to a local target installation", key)
			}
			if err := inspectPortableTargetValue(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := inspectPortableTargetValue(child); err != nil {
				return err
			}
		}
	case string:
		if windowsDrivePath.MatchString(typed) || strings.HasPrefix(typed, `\\`) {
			return errors.New("local filesystem paths are not portable target defaults")
		}
	}
	return nil
}

func validDiscoveryHints(hints []TargetDiscoveryHint) bool {
	previous := ""
	for _, hint := range hints {
		key := hint.Kind + "\x00" + hint.Value
		if key <= previous || !validDiscoveryHintKind(hint.Kind) || !validPortableHintValue(hint.Value) {
			return false
		}
		previous = key
	}
	return true
}

func validDiscoveryHintKind(kind string) bool {
	switch kind {
	case "application-name", "executable-name", "window-title", "android-package", "device-model", "browser-host":
		return true
	default:
		return false
	}
}

func validPortableHintValue(value string) bool {
	if value == "" || len(value) > 512 || !utf8.ValidString(value) || strings.TrimSpace(value) != value ||
		strings.ContainsAny(value, `\\/`) || windowsDrivePath.MatchString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validatePublisherNamespace(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.Fragment != "" || parsed.RawPath != "" || parsed.Path == "" || parsed.Host != strings.ToLower(parsed.Host) ||
		path.Clean(parsed.Path) != parsed.Path || strings.HasSuffix(parsed.Path, "/") {
		return nil, errors.New("publisher namespace must be a canonical HTTPS URI with a non-root path")
	}
	return parsed, nil
}

func validateOwnedVersionedURI(namespace *url.URL, value string) error {
	if namespace == nil {
		return errors.New("publisher namespace is invalid")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != namespace.Scheme || parsed.Host != namespace.Host || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" || path.Clean(parsed.Path) != parsed.Path ||
		!versionedPathPattern.MatchString(parsed.Path) || !strings.HasPrefix(parsed.EscapedPath(), namespace.EscapedPath()+"/") {
		return errors.New("package id must be a versioned URI inside its publisher namespace")
	}
	return nil
}

func validateVersionedHTTPSURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.Fragment != "" || parsed.RawPath != "" || path.Clean(parsed.Path) != parsed.Path || !versionedPathPattern.MatchString(parsed.Path) {
		return errors.New("value must be a canonical versioned HTTPS URI")
	}
	return nil
}

func compareNodeRef(left, right nodecontract.NodeRef) int {
	if left.NodeTypeID < right.NodeTypeID {
		return -1
	}
	if left.NodeTypeID > right.NodeTypeID {
		return 1
	}
	if left.Version < right.Version {
		return -1
	}
	if left.Version > right.Version {
		return 1
	}
	return strings.Compare(left.SemanticDigest.String(), right.SemanticDigest.String())
}

func equalSchemaResources(left, right []datatype.SchemaResource) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].ID != right[index].ID || !bytes.Equal(left[index].Schema, right[index].Schema) {
			return false
		}
	}
	return true
}

func validResolution(value [2]int) bool {
	return value[0] > 0 && value[0] <= 100_000 && value[1] > 0 && value[1] <= 100_000
}

func validBBox(value [4]int, resolution [2]int) bool {
	return value[0] >= 0 && value[1] >= 0 && value[2] > value[0] && value[3] > value[1] &&
		value[2] <= resolution[0] && value[3] <= resolution[1]
}

func validSortedTextSet(values []string) bool {
	previous := ""
	for _, value := range values {
		if value == "" || strings.TrimSpace(value) != value || value <= previous {
			return false
		}
		previous = value
	}
	return true
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}
