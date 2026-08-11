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
)

const (
	VersionDomainReleaseFormat  = "yotta.version-domain-release"
	VersionDomainReleaseVersion = 1
	versionDomainReleaseFile    = "version-domains.json"
)

// CurrentVersionDomain is the code-owned compatibility declaration for one
// independently versioned domain. ReadableVersions must include CurrentVersion
// and every version frozen by a supported public release.
type CurrentVersionDomain struct {
	Name             string
	CurrentVersion   string
	Class            string
	ReadableVersions []string
}

type ReleasedVersionDomain struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Class   string `json:"class"`
}

type VersionDomainRelease struct {
	Format         string                  `json:"format"`
	Version        int                     `json:"version"`
	ProductVersion string                  `json:"productVersion"`
	Domains        []ReleasedVersionDomain `json:"domains"`
}

type VersionDomainReleases struct {
	Root string
}

func (r VersionDomainReleases) Freeze(productVersion string, current []CurrentVersionDomain) error {
	document, raw, err := buildVersionDomainRelease(productVersion, current)
	if err != nil {
		return err
	}
	if strings.TrimSpace(r.Root) == "" {
		return errors.New("version-domain release root is required")
	}
	path := filepath.Join(r.Root, document.ProductVersion, versionDomainReleaseFile)
	existing, readErr := os.ReadFile(path)
	if readErr == nil {
		if bytes.Equal(existing, raw) {
			return nil
		}
		return fmt.Errorf("released version-domain snapshot %q is immutable", path)
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

func (r VersionDomainReleases) Check(
	currentProductVersion string,
	current []CurrentVersionDomain,
	requireCurrent bool,
) (int, error) {
	if strings.TrimSpace(r.Root) == "" {
		return 0, errors.New("version-domain release root is required")
	}
	if !productVersionPattern.MatchString(currentProductVersion) {
		return 0, fmt.Errorf("current product version %q is invalid", currentProductVersion)
	}
	currentByName, err := indexCurrentVersionDomains(current)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(r.Root)
	if err != nil {
		return 0, fmt.Errorf("read version-domain release snapshots: %w", err)
	}
	checked := 0
	foundCurrent := false
	for _, entry := range entries {
		if !entry.IsDir() || !productVersionPattern.MatchString(entry.Name()) {
			return 0, fmt.Errorf("unexpected version-domain release entry %q", entry.Name())
		}
		path := filepath.Join(r.Root, entry.Name(), versionDomainReleaseFile)
		document, err := readVersionDomainRelease(path)
		if err != nil {
			return 0, err
		}
		if document.ProductVersion != entry.Name() {
			return 0, fmt.Errorf("version-domain release snapshot %q has product version %q", path, document.ProductVersion)
		}
		for _, released := range document.Domains {
			now, ok := currentByName[released.Name]
			if !ok {
				return 0, fmt.Errorf("release %s domain %q was removed", document.ProductVersion, released.Name)
			}
			if now.Class != released.Class {
				return 0, fmt.Errorf("release %s domain %q changed class from %q to %q", document.ProductVersion, released.Name, released.Class, now.Class)
			}
			if !containsVersion(now.ReadableVersions, released.Version) {
				return 0, fmt.Errorf(
					"release %s domain %q version %q is not declared readable by current version %q",
					document.ProductVersion, released.Name, released.Version, now.CurrentVersion,
				)
			}
		}
		checked++
		foundCurrent = foundCurrent || document.ProductVersion == currentProductVersion
	}
	if checked == 0 {
		return 0, errors.New("no released version-domain compatibility snapshot exists")
	}
	if requireCurrent && !foundCurrent {
		return 0, fmt.Errorf("product %s has no frozen version-domain compatibility snapshot", currentProductVersion)
	}
	return checked, nil
}

func buildVersionDomainRelease(
	productVersion string,
	current []CurrentVersionDomain,
) (VersionDomainRelease, []byte, error) {
	if !productVersionPattern.MatchString(productVersion) {
		return VersionDomainRelease{}, nil, fmt.Errorf("product version %q is invalid", productVersion)
	}
	indexed, err := indexCurrentVersionDomains(current)
	if err != nil {
		return VersionDomainRelease{}, nil, err
	}
	domains := make([]ReleasedVersionDomain, 0, len(indexed))
	for _, domain := range indexed {
		domains = append(domains, ReleasedVersionDomain{
			Name: domain.Name, Version: domain.CurrentVersion, Class: domain.Class,
		})
	}
	sort.Slice(domains, func(i, j int) bool { return domains[i].Name < domains[j].Name })
	document := VersionDomainRelease{
		Format: VersionDomainReleaseFormat, Version: VersionDomainReleaseVersion,
		ProductVersion: productVersion, Domains: domains,
	}
	if err := validateVersionDomainRelease(document); err != nil {
		return VersionDomainRelease{}, nil, err
	}
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return VersionDomainRelease{}, nil, err
	}
	return document, append(raw, '\n'), nil
}

func indexCurrentVersionDomains(current []CurrentVersionDomain) (map[string]CurrentVersionDomain, error) {
	if len(current) == 0 {
		return nil, errors.New("current version-domain inventory is empty")
	}
	result := make(map[string]CurrentVersionDomain, len(current))
	for _, domain := range current {
		if strings.TrimSpace(domain.Name) != domain.Name || domain.Name == "" ||
			strings.TrimSpace(domain.CurrentVersion) != domain.CurrentVersion || domain.CurrentVersion == "" ||
			strings.TrimSpace(domain.Class) != domain.Class || domain.Class == "" || len(domain.ReadableVersions) == 0 {
			return nil, fmt.Errorf("current version domain %q is invalid", domain.Name)
		}
		if _, duplicate := result[domain.Name]; duplicate {
			return nil, fmt.Errorf("current version domain %q is duplicated", domain.Name)
		}
		readable := append([]string(nil), domain.ReadableVersions...)
		sort.Strings(readable)
		for index, version := range readable {
			if strings.TrimSpace(version) != version || version == "" || index > 0 && readable[index-1] == version {
				return nil, fmt.Errorf("current version domain %q has invalid readable versions", domain.Name)
			}
		}
		if !containsVersion(readable, domain.CurrentVersion) {
			return nil, fmt.Errorf("current version domain %q does not read its writer version %q", domain.Name, domain.CurrentVersion)
		}
		domain.ReadableVersions = readable
		result[domain.Name] = domain
	}
	return result, nil
}

func readVersionDomainRelease(path string) (VersionDomainRelease, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return VersionDomainRelease{}, fmt.Errorf("read version-domain release snapshot %q: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document VersionDomainRelease
	if err := decoder.Decode(&document); err != nil {
		return VersionDomainRelease{}, fmt.Errorf("decode version-domain release snapshot %q: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return VersionDomainRelease{}, fmt.Errorf("version-domain release snapshot %q must contain exactly one JSON value", path)
	}
	if err := validateVersionDomainRelease(document); err != nil {
		return VersionDomainRelease{}, fmt.Errorf("version-domain release snapshot %q: %w", path, err)
	}
	return document, nil
}

func validateVersionDomainRelease(document VersionDomainRelease) error {
	if document.Format != VersionDomainReleaseFormat || document.Version != VersionDomainReleaseVersion ||
		!productVersionPattern.MatchString(document.ProductVersion) || len(document.Domains) == 0 {
		return errors.New("version-domain release identity is invalid")
	}
	previous := ""
	for _, domain := range document.Domains {
		if domain.Name <= previous || strings.TrimSpace(domain.Version) != domain.Version || domain.Version == "" ||
			strings.TrimSpace(domain.Class) != domain.Class || domain.Class == "" {
			return errors.New("version-domain release entries are invalid or not uniquely sorted")
		}
		previous = domain.Name
	}
	return nil
}

func containsVersion(versions []string, target string) bool {
	for _, version := range versions {
		if version == target {
			return true
		}
	}
	return false
}
