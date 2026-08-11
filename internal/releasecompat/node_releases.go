// Package releasecompat owns immutable release compatibility floors and the
// checks that prove current code can still open them.
package releasecompat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
)

const (
	BuiltinNodeReleaseFormat  = "yotta.builtin-node-release"
	BuiltinNodeReleaseVersion = 1
	builtinNodeReleaseFile    = "builtin-node-refs.json"
)

var productVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type BuiltinNodeRelease struct {
	Format         string                 `json:"format"`
	Version        int                    `json:"version"`
	ProductVersion string                 `json:"productVersion"`
	NodeRefs       []nodecontract.NodeRef `json:"nodeRefs"`
}

type NodeReleases struct {
	Root string
}

// Freeze creates one immutable snapshot for a product release. An existing
// snapshot may be verified again, but is never overwritten.
func (r NodeReleases) Freeze(productVersion string, refs []nodecontract.NodeRef) error {
	document, raw, err := buildBuiltinNodeRelease(productVersion, refs)
	if err != nil {
		return err
	}
	if strings.TrimSpace(r.Root) == "" {
		return errors.New("node release root is required")
	}
	path := filepath.Join(r.Root, document.ProductVersion, builtinNodeReleaseFile)
	existing, readErr := os.ReadFile(path)
	if readErr == nil {
		if bytes.Equal(existing, raw) {
			return nil
		}
		return fmt.Errorf("released node snapshot %q is immutable", path)
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

// Check verifies every frozen release against the current built-in Catalog.
// requireCurrent is reserved for packaging: ordinary development checks can
// evolve toward a future release without prematurely freezing its snapshot.
func (r NodeReleases) Check(
	currentProductVersion string,
	currentRefs []nodecontract.NodeRef,
	requireCurrent bool,
) (int, error) {
	if strings.TrimSpace(r.Root) == "" {
		return 0, errors.New("node release root is required")
	}
	if !productVersionPattern.MatchString(currentProductVersion) {
		return 0, fmt.Errorf("current product version %q is invalid", currentProductVersion)
	}
	entries, err := os.ReadDir(r.Root)
	if err != nil {
		return 0, fmt.Errorf("read node release snapshots: %w", err)
	}
	checked := 0
	foundCurrent := false
	for _, entry := range entries {
		if !entry.IsDir() || !productVersionPattern.MatchString(entry.Name()) {
			return 0, fmt.Errorf("unexpected node release entry %q", entry.Name())
		}
		path := filepath.Join(r.Root, entry.Name(), builtinNodeReleaseFile)
		document, err := readBuiltinNodeRelease(path)
		if err != nil {
			return 0, err
		}
		if document.ProductVersion != entry.Name() {
			return 0, fmt.Errorf("node release snapshot %q has product version %q", path, document.ProductVersion)
		}
		if err := authoring.ValidateReleasedNodeContracts(document.NodeRefs, currentRefs); err != nil {
			return 0, fmt.Errorf("node release %s is not upgradeable: %w", document.ProductVersion, err)
		}
		checked++
		foundCurrent = foundCurrent || document.ProductVersion == currentProductVersion
	}
	if checked == 0 {
		return 0, errors.New("no released node compatibility snapshot exists")
	}
	if requireCurrent && !foundCurrent {
		return 0, fmt.Errorf("product %s has no frozen node compatibility snapshot", currentProductVersion)
	}
	return checked, nil
}

func buildBuiltinNodeRelease(
	productVersion string,
	refs []nodecontract.NodeRef,
) (BuiltinNodeRelease, []byte, error) {
	if !productVersionPattern.MatchString(productVersion) {
		return BuiltinNodeRelease{}, nil, fmt.Errorf("product version %q is invalid", productVersion)
	}
	ordered := append([]nodecontract.NodeRef(nil), refs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].NodeTypeID < ordered[j].NodeTypeID })
	document := BuiltinNodeRelease{
		Format: BuiltinNodeReleaseFormat, Version: BuiltinNodeReleaseVersion,
		ProductVersion: productVersion, NodeRefs: ordered,
	}
	if err := validateBuiltinNodeRelease(document); err != nil {
		return BuiltinNodeRelease{}, nil, err
	}
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return BuiltinNodeRelease{}, nil, err
	}
	return document, append(raw, '\n'), nil
}

func readBuiltinNodeRelease(path string) (BuiltinNodeRelease, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return BuiltinNodeRelease{}, fmt.Errorf("read node release snapshot %q: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document BuiltinNodeRelease
	if err := decoder.Decode(&document); err != nil {
		return BuiltinNodeRelease{}, fmt.Errorf("decode node release snapshot %q: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return BuiltinNodeRelease{}, fmt.Errorf("node release snapshot %q must contain exactly one JSON value", path)
	}
	if err := validateBuiltinNodeRelease(document); err != nil {
		return BuiltinNodeRelease{}, fmt.Errorf("node release snapshot %q: %w", path, err)
	}
	return document, nil
}

func validateBuiltinNodeRelease(document BuiltinNodeRelease) error {
	if document.Format != BuiltinNodeReleaseFormat || document.Version != BuiltinNodeReleaseVersion ||
		!productVersionPattern.MatchString(document.ProductVersion) || len(document.NodeRefs) == 0 {
		return errors.New("node release identity is invalid")
	}
	previous := ""
	for _, ref := range document.NodeRefs {
		if ref.NodeTypeID <= previous {
			return errors.New("node release refs are not uniquely sorted")
		}
		previous = ref.NodeTypeID
	}
	return nil
}
