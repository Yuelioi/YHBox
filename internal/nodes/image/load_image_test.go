package image

import (
	"bytes"
	"context"
	imagepkg "image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func runLoadImage(t *testing.T, path string) node.RunResult {
	t.Helper()
	registry := node.NewRegistry()
	registry.Register(&LoadImage{})
	rn, _ := registry.Get("LoadImage")
	return node.RunNode(context.Background(), rn, nil,
		map[string]any{"Path": path}, nil, node.StubServices(), false)
}

func writeDataFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadImage_PNG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YOTTA_DATA_DIR", dir)
	pngBytes := tinyPNG(t)
	writeDataFile(t, dir, "pic.png", pngBytes)

	res := runLoadImage(t, "pic.png")
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	img := res.OutputData["Image"].(node.Image)
	if img.Format != "png" || !bytes.Equal(img.Data, pngBytes) {
		t.Errorf("format=%q dataEqual=%v", img.Format, bytes.Equal(img.Data, pngBytes))
	}
}

func TestLoadImage_JPEG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YOTTA_DATA_DIR", dir)
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, imagepkg.NewRGBA(imagepkg.Rect(0, 0, 2, 2)), nil)
	writeDataFile(t, dir, "pic.jpg", buf.Bytes())

	res := runLoadImage(t, "pic.jpg")
	if res.Error != nil || res.OutputData["Image"].(node.Image).Format != "jpeg" {
		t.Fatalf("jpeg load: exit=%q err=%v", res.ExitName, res.Error)
	}
}

func TestLoadImage_NonImage_Fails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YOTTA_DATA_DIR", dir)
	writeDataFile(t, dir, "notes.txt", []byte("just text, not an image"))
	if res := runLoadImage(t, "notes.txt"); res.Error == nil {
		t.Fatal("非 PNG/JPEG 文件应失败")
	}
}

func TestLoadImage_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(abs, tinyPNG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	res := runLoadImage(t, abs) // 绝对路径, 任意本地图
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("绝对路径应能加载: exit=%q err=%v", res.ExitName, res.Error)
	}
	if res.OutputData["Image"].(node.Image).Format != "png" {
		t.Error("绝对路径加载的应是 png")
	}
}

func TestLoadImage_Missing_Fails(t *testing.T) {
	t.Setenv("YOTTA_DATA_DIR", t.TempDir())
	if res := runLoadImage(t, "nope.png"); res.Error == nil {
		t.Fatal("缺文件应失败")
	}
}
