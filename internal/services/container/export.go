package container

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/container/dependency"
)

// ExportPackageZip writes a portable .yotta-container.zip bundle.
// It excludes installation.json because installation is local machine state.
func (s *Store) ExportPackageZip(id, destPath string) error {
	if err := validateID(id); err != nil {
		return err
	}
	// Save/Delete hold the write lock across disk mutation. Keep the matching read
	// lock until every authoritative file has been copied so an export cannot mix
	// files from two committed generations.
	s.mu.RLock()
	defer s.mu.RUnlock()
	cached, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("container %q not found", id)
	}
	c, err := cloneContainer(cached)
	if err != nil {
		return fmt.Errorf("snapshot container %q: %w", id, err)
	}
	dir := filepath.Join(s.root, id)
	manifestBytes, err := os.ReadFile(filepath.Join(dir, packageFile))
	if err != nil {
		return fmt.Errorf("read %s: %w", packageFile, err)
	}
	var manifest PackageManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("decode %s: %w", packageFile, err)
	}
	graphBytes, err := os.ReadFile(filepath.Join(dir, graphFile))
	if err != nil {
		return fmt.Errorf("read %s: %w", graphFile, err)
	}
	var graph Graph
	if err := json.Unmarshal(graphBytes, &graph); err != nil {
		return fmt.Errorf("decode %s: %w", graphFile, err)
	}
	lockBytes, err := os.ReadFile(filepath.Join(dir, lockFile))
	if err != nil {
		return fmt.Errorf("read %s: %w", lockFile, err)
	}
	var lock YottaLock
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		return fmt.Errorf("decode %s: %w", lockFile, err)
	}
	subgraphs := s.subgraphsFor(&c)
	closure, err := dependencyClosure(graph, subgraphs)
	if err != nil {
		return fmt.Errorf("resolve dependency closure: %w", err)
	}
	if err := validatePackageLock(manifest, graph, subgraphs, lock); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, file := range []struct {
		name string
		data []byte
	}{
		{name: packageFile, data: manifestBytes},
		{name: graphFile, data: graphBytes},
	} {
		writer, err := zw.Create(file.name)
		if err != nil {
			return err
		}
		if _, err := writer.Write(file.data); err != nil {
			return err
		}
	}
	portableLock := lock
	portableLock.InstallationHash = ""
	portableLockBytes, err := json.MarshalIndent(portableLock, "", "  ")
	if err != nil {
		return err
	}
	lockWriter, err := zw.Create(lockFile)
	if err != nil {
		return err
	}
	if _, err := lockWriter.Write(portableLockBytes); err != nil {
		return err
	}
	for _, sg := range subgraphs {
		b, err := json.MarshalIndent(sg, "", "  ")
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(filepath.Join("subgraphs", sg.ID+".json")))
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	if err := s.addAssetClosureToZip(zw, closure); err != nil {
		return err
	}
	return nil
}

func validatePackageLock(manifest PackageManifest, graph Graph, subgraphs []Subgraph, lock YottaLock) error {
	expected, err := buildContainerLock(manifest, graph, subgraphs, lock.GeneratedAt)
	if err != nil {
		return err
	}
	if lock.SchemaVersion != expected.SchemaVersion ||
		lock.PackageID != expected.PackageID ||
		lock.PackageName != expected.PackageName ||
		lock.Version != expected.Version ||
		lock.ManifestHash != expected.ManifestHash ||
		lock.GraphHash != expected.GraphHash ||
		lock.ClosureHash != expected.ClosureHash ||
		!lockDependenciesEqual(lock.Dependencies, expected.Dependencies) {
		return fmt.Errorf("yotta-lock.json is stale; save the container before export")
	}
	return nil
}

func lockDependenciesEqual(a, b LockDependencies) bool {
	return stringListEqual(a.Templates, b.Templates) &&
		stringListEqual(a.Clips, b.Clips) &&
		stringListEqual(a.Subgraphs, b.Subgraphs) &&
		stringListEqual(a.Assets, b.Assets) &&
		stringListEqual(a.AISlots, b.AISlots) &&
		stringListEqual(a.TargetSlots, b.TargetSlots)
}

func stringListEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func dependencyClosure(graph Graph, subgraphs []Subgraph) (dependency.ClosureResult, error) {
	byID := make(map[string]Subgraph, len(subgraphs))
	for _, sg := range subgraphs {
		byID[sg.ID] = sg
	}
	return dependency.Closure(depNodeInfos(graph.Nodes), func(sgID string) ([]dependency.NodeInfo, error) {
		sg, ok := byID[sgID]
		if !ok {
			return nil, nil
		}
		return depNodeInfos(sg.Graph.Nodes), nil
	})
}

func (s *Store) addAssetClosureToZip(zw *zip.Writer, closure dependency.ClosureResult) error {
	if len(closure.Templates) == 0 && len(closure.Clips) == 0 {
		return nil
	}
	if s.assetStore == nil {
		return fmt.Errorf("asset store not configured for package export")
	}
	templateBlobs := map[string]bool{}
	for _, guid := range sortedStrings(closure.Templates) {
		rec, ok := s.assetStore.Get(guid)
		if !ok {
			return fmt.Errorf("template asset %q not found", guid)
		}
		if rec.Kind != asset.KindTemplate {
			return fmt.Errorf("asset %q kind=%q, want %q", guid, rec.Kind, asset.KindTemplate)
		}
		if err := addZipJSON(zw, filepath.ToSlash(filepath.Join("assets", "records", guid+".json")), rec); err != nil {
			return err
		}
		for _, v := range rec.Variants {
			if v.Blob != "" {
				templateBlobs[v.Blob] = true
			}
		}
	}
	for _, sha := range sortedKeys(templateBlobs) {
		if err := addAssetBlob(zw, s.assetStore, sha, filepath.ToSlash(filepath.Join("assets", "blobs", sha))); err != nil {
			return err
		}
	}

	clipBlobs := map[string]bool{}
	for _, guid := range sortedStrings(closure.Clips) {
		rec, ok := s.assetStore.Get(guid)
		if !ok {
			return fmt.Errorf("clip asset %q not found", guid)
		}
		if rec.Kind != asset.KindClip {
			return fmt.Errorf("asset %q kind=%q, want %q", guid, rec.Kind, asset.KindClip)
		}
		if err := addZipJSON(zw, filepath.ToSlash(filepath.Join("clips", guid+".json")), rec); err != nil {
			return err
		}
		if rec.Blob != "" {
			clipBlobs[rec.Blob] = true
		}
	}
	for _, sha := range sortedKeys(clipBlobs) {
		if err := addAssetBlob(zw, s.assetStore, sha, filepath.ToSlash(filepath.Join("clips", "blobs", sha))); err != nil {
			return err
		}
	}
	return nil
}

func addZipJSON(zw *zip.Writer, zipName string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return addZipBytes(zw, zipName, b)
}

func addAssetBlob(zw *zip.Writer, store *asset.Store, sha, zipName string) error {
	b, err := store.Blobs().Read(sha)
	if err != nil {
		return err
	}
	return addZipBytes(zw, zipName, b)
}

func addZipBytes(zw *zip.Writer, zipName string, data []byte) error {
	w, err := zw.Create(zipName)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
