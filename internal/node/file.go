package node

import (
	"mime"
	"os"
	"path/filepath"
)

// FileFromPath snapshots local file metadata.
func FileFromPath(path string) (File, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return File{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return File{}, err
	}
	ext := filepath.Ext(abs)
	return File{
		Path:      abs,
		Name:      filepath.Base(abs),
		Ext:       ext,
		MIME:      mime.TypeByExtension(ext),
		Size:      info.Size(),
		ModTimeMs: info.ModTime().UnixMilli(),
		IsDir:     info.IsDir(),
	}, nil
}
