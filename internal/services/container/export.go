package container

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ExportPackageZip writes a portable .yotta-container.zip bundle.
// It excludes installation.json because installation is local machine state.
func (s *Store) ExportPackageZip(id, destPath string) error {
	if err := validateID(id); err != nil {
		return err
	}
	c, ok := s.Get(id)
	if !ok {
		return fmt.Errorf("container %q not found", id)
	}
	dir := filepath.Join(s.root, id)
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

	for _, name := range []string{packageFile, graphFile, lockFile} {
		if err := addZipFile(zw, filepath.Join(dir, name), name); err != nil {
			return err
		}
	}
	for _, sg := range s.subgraphsFor(&c) {
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
	return nil
}

func addZipFile(zw *zip.Writer, srcPath, zipName string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	w, err := zw.Create(zipName)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}
