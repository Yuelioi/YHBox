package releasecompat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
)

const (
	BuiltinCatalogReleaseFormat  = "yotta.builtin-catalog-release"
	BuiltinCatalogReleaseVersion = 1
	builtinCatalogReleaseFile    = "builtin-catalog-refs.json"
)

// BuiltinCatalogRelease freezes the type and capability identities that may
// be referenced by durable workflows and run artifacts. Unlike NodeRefs,
// these identities have no rewrite registry: released refs must remain in the
// Catalog exactly, while new versioned IDs may be added alongside them.
type BuiltinCatalogRelease struct {
	Format         string             `json:"format"`
	Version        int                `json:"version"`
	ProductVersion string             `json:"productVersion"`
	TypeRefs       []datatype.TypeRef `json:"typeRefs"`
	CapabilityRefs []capability.Ref   `json:"capabilityRefs"`
}

type CatalogReleases struct {
	Root string
}

func (r CatalogReleases) Freeze(
	productVersion string,
	typeRefs []datatype.TypeRef,
	capabilityRefs []capability.Ref,
) error {
	document, raw, err := buildBuiltinCatalogRelease(productVersion, typeRefs, capabilityRefs)
	if err != nil {
		return err
	}
	if strings.TrimSpace(r.Root) == "" {
		return errors.New("catalog release root is required")
	}
	path := filepath.Join(r.Root, document.ProductVersion, builtinCatalogReleaseFile)
	existing, readErr := os.ReadFile(path)
	if readErr == nil {
		if bytes.Equal(existing, raw) {
			return nil
		}
		return fmt.Errorf("released Catalog snapshot %q is immutable", path)
	}
	if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		_ = file.Close()
		if !committed {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(raw); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r CatalogReleases) Check(
	currentProductVersion string,
	typeRefs []datatype.TypeRef,
	capabilityRefs []capability.Ref,
	requireCurrent bool,
) (int, error) {
	if strings.TrimSpace(r.Root) == "" {
		return 0, errors.New("catalog release root is required")
	}
	if !productVersionPattern.MatchString(currentProductVersion) {
		return 0, fmt.Errorf("current product version %q is invalid", currentProductVersion)
	}
	currentTypes, currentCapabilities, err := indexCatalogRefs(typeRefs, capabilityRefs)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(r.Root)
	if err != nil {
		return 0, fmt.Errorf("read Catalog release snapshots: %w", err)
	}
	checked := 0
	foundCurrent := false
	for _, entry := range entries {
		if !entry.IsDir() || !productVersionPattern.MatchString(entry.Name()) {
			return 0, fmt.Errorf("unexpected Catalog release entry %q", entry.Name())
		}
		path := filepath.Join(r.Root, entry.Name(), builtinCatalogReleaseFile)
		document, err := readBuiltinCatalogRelease(path)
		if err != nil {
			return 0, err
		}
		if document.ProductVersion != entry.Name() {
			return 0, fmt.Errorf("catalog release snapshot %q has product version %q", path, document.ProductVersion)
		}
		for _, released := range document.TypeRefs {
			current, ok := currentTypes[released.TypeID]
			if !ok {
				return 0, fmt.Errorf("release %s data type %q was removed", document.ProductVersion, released.TypeID)
			}
			if current != released {
				return 0, fmt.Errorf("release %s data type %q changed semantic digest; retain the released ref and add a new versioned type ID", document.ProductVersion, released.TypeID)
			}
		}
		for _, released := range document.CapabilityRefs {
			current, ok := currentCapabilities[released.CapabilityID]
			if !ok {
				return 0, fmt.Errorf("release %s capability %q was removed", document.ProductVersion, released.CapabilityID)
			}
			if current != released {
				return 0, fmt.Errorf("release %s capability %q changed semantic digest; retain the released ref and add a new versioned capability ID", document.ProductVersion, released.CapabilityID)
			}
		}
		checked++
		foundCurrent = foundCurrent || document.ProductVersion == currentProductVersion
	}
	if checked == 0 {
		return 0, errors.New("no released Catalog compatibility snapshot exists")
	}
	if requireCurrent && !foundCurrent {
		return 0, fmt.Errorf("product %s has no frozen Catalog compatibility snapshot", currentProductVersion)
	}
	return checked, nil
}

func buildBuiltinCatalogRelease(
	productVersion string,
	typeRefs []datatype.TypeRef,
	capabilityRefs []capability.Ref,
) (BuiltinCatalogRelease, []byte, error) {
	if !productVersionPattern.MatchString(productVersion) {
		return BuiltinCatalogRelease{}, nil, fmt.Errorf("product version %q is invalid", productVersion)
	}
	types := append([]datatype.TypeRef(nil), typeRefs...)
	capabilities := append([]capability.Ref(nil), capabilityRefs...)
	sort.Slice(types, func(i, j int) bool { return types[i].TypeID < types[j].TypeID })
	sort.Slice(capabilities, func(i, j int) bool { return capabilities[i].CapabilityID < capabilities[j].CapabilityID })
	document := BuiltinCatalogRelease{
		Format: BuiltinCatalogReleaseFormat, Version: BuiltinCatalogReleaseVersion,
		ProductVersion: productVersion, TypeRefs: types, CapabilityRefs: capabilities,
	}
	if err := validateBuiltinCatalogRelease(document); err != nil {
		return BuiltinCatalogRelease{}, nil, err
	}
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return BuiltinCatalogRelease{}, nil, err
	}
	return document, append(raw, '\n'), nil
}

func indexCatalogRefs(
	typeRefs []datatype.TypeRef,
	capabilityRefs []capability.Ref,
) (map[string]datatype.TypeRef, map[string]capability.Ref, error) {
	if len(typeRefs) == 0 || len(capabilityRefs) == 0 {
		return nil, nil, errors.New("current Catalog refs are incomplete")
	}
	types := make(map[string]datatype.TypeRef, len(typeRefs))
	for _, ref := range typeRefs {
		if err := ref.Validate(); err != nil {
			return nil, nil, fmt.Errorf("current data type ref: %w", err)
		}
		if _, duplicate := types[ref.TypeID]; duplicate {
			return nil, nil, fmt.Errorf("current data type %q is duplicated", ref.TypeID)
		}
		types[ref.TypeID] = ref
	}
	capabilities := make(map[string]capability.Ref, len(capabilityRefs))
	for _, ref := range capabilityRefs {
		if err := ref.Validate(); err != nil {
			return nil, nil, fmt.Errorf("current capability ref: %w", err)
		}
		if _, duplicate := capabilities[ref.CapabilityID]; duplicate {
			return nil, nil, fmt.Errorf("current capability %q is duplicated", ref.CapabilityID)
		}
		capabilities[ref.CapabilityID] = ref
	}
	return types, capabilities, nil
}

func readBuiltinCatalogRelease(path string) (BuiltinCatalogRelease, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return BuiltinCatalogRelease{}, fmt.Errorf("read Catalog release snapshot %q: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document BuiltinCatalogRelease
	if err := decoder.Decode(&document); err != nil {
		return BuiltinCatalogRelease{}, fmt.Errorf("decode Catalog release snapshot %q: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return BuiltinCatalogRelease{}, fmt.Errorf("catalog release snapshot %q must contain exactly one JSON value", path)
	}
	if err := validateBuiltinCatalogRelease(document); err != nil {
		return BuiltinCatalogRelease{}, fmt.Errorf("catalog release snapshot %q: %w", path, err)
	}
	return document, nil
}

func validateBuiltinCatalogRelease(document BuiltinCatalogRelease) error {
	if document.Format != BuiltinCatalogReleaseFormat || document.Version != BuiltinCatalogReleaseVersion ||
		!productVersionPattern.MatchString(document.ProductVersion) || len(document.TypeRefs) == 0 || len(document.CapabilityRefs) == 0 {
		return errors.New("catalog release identity is invalid")
	}
	previous := ""
	for _, ref := range document.TypeRefs {
		if ref.TypeID <= previous || ref.Validate() != nil {
			return errors.New("catalog release data types are invalid or not uniquely sorted")
		}
		previous = ref.TypeID
	}
	previous = ""
	for _, ref := range document.CapabilityRefs {
		if ref.CapabilityID <= previous || ref.Validate() != nil {
			return errors.New("catalog release capabilities are invalid or not uniquely sorted")
		}
		previous = ref.CapabilityID
	}
	return nil
}
