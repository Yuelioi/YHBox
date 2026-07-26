package migrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func ensureTrustedSubdirectory(root, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("migration directory escaped storage root")
	}
	current := root
	if err := requireTrustedDirectory(current); err != nil {
		return err
	}
	if relative == "." {
		return nil
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			return errors.New("migration directory path is invalid")
		}
		current = filepath.Join(current, component)
		if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if err := requireTrustedDirectory(current); err != nil {
			return err
		}
	}
	return nil
}

func requireTrustedDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("migration path contains an untrusted directory")
	}
	return nil
}
