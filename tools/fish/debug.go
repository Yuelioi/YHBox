package fish

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

var debugDir string

func SetDebugDir(dir string) {
	debugDir = dir
}

func saveDebugFrame(name string, img *image.RGBA) string {
	if debugDir == "" {
		debugDir = "logs"
	}
	_ = os.MkdirAll(debugDir, 0o755)
	ts := time.Now().Format("20060102_150405.000")
	filename := fmt.Sprintf("%s_%s.png", ts, name)
	path := filepath.Join(debugDir, filename)
	f, err := os.Create(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	_ = png.Encode(f, img)
	return filename
}
