// Package yhbox 是 Go 端的根包，承载 embed 模板 PNG。
package yhbox

import (
	"embed"
	"fmt"
	"io/fs"

	"yhbox/pkg/vision"
)

//go:embed assets/templates.toml assets/fish/*.png assets/cook/*.png assets/battle/*.png assets/midi/*.mid
var assetsFS embed.FS

const assetsPrefix = "assets/"

func init() {
	// 把 yhbox.AssetBytes 注入 vision 包，让 vision.LoadPNGForSpec(SourceEmbed)
	// 知道从哪读字节流。放在 yhbox 根包 init 里，确保 fish/cook 任一调用 vision 之前
	// （这俩都 import yhbox，init 必然先跑）已经设好。
	vision.PNGLoader = AssetBytes
}

// AssetBytes 按 path 读取 embed 的资源，path 相对 assets/。
// 如 "templates.toml"、"fish/hook_icon_1080.png"。
func AssetBytes(name string) ([]byte, error) {
	return fs.ReadFile(assetsFS, assetsPrefix+name)
}

// MustAssetBytes 同上，找不到 panic。
func MustAssetBytes(name string) []byte {
	b, err := AssetBytes(name)
	if err != nil {
		panic(fmt.Sprintf("yhbox: 找不到 asset %q: %v", name, err))
	}
	return b
}

// AssetList 列出所有 embed 进来的资源（递归子目录）。
// 返回路径不含 "assets/" 前缀，例：["templates.toml", "fish/hook_icon_1080.png", ...]
func AssetList() []string {
	var out []string
	_ = fs.WalkDir(assetsFS, assetsPrefix[:len(assetsPrefix)-1], func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		// 去掉 "assets/" 前缀
		out = append(out, path[len(assetsPrefix):])
		return nil
	})
	return out
}
