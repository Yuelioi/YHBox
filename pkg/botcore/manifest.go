// Package botcore 提供各 bot 共用的 runtime 类型和加载工具：
//   - RuntimeAssets：bot 启动后 locale-bound 的视觉资产句柄（templates fs.FS + toml bytes）
//   - Manifest 解析：configs/<locale>/manifest.yaml 描述该 locale 是否实装、能否
//     alias 别的 locale，让 bot 在某些 locale 复用另一份配置（如 rhythm 跨语言无差异）
//   - LoadYAMLWithOverride：embedded YAML + exe 同目录 override merge
package botcore

import (
	"errors"
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

// ErrLocaleNotImplemented 在 manifest.implemented=false 且无 alias 时返回。
// caller (main.go) 检测到此 error 应 log info 而非 error，并标 bot unavailable。
var ErrLocaleNotImplemented = errors.New("locale not implemented")

// Manifest 是 tools/<bot>/configs/<locale>/manifest.yaml 的反序列化结构。
//
// 三种状态：
//   - Implemented=true, LocaleAlias=""：本 locale 有自己的 config/templates，直接用。
//   - Implemented=任意, LocaleAlias="zh"：本 locale 复用 alias locale 的 config/templates
//     （例如 rhythm 跨语言无差异，en 直接 alias zh）。此时 Implemented 被忽略。
//   - Implemented=false, LocaleAlias=""：未实装，返回 ErrLocaleNotImplemented。
type Manifest struct {
	Version     int    `yaml:"version"`
	Implemented bool   `yaml:"implemented"`
	Reason      string `yaml:"reason"`
	LocaleAlias string `yaml:"locale_alias,omitempty"`
}

// LoadManifest 读 fsys 里 path 处的 manifest.yaml。
// 返回 (manifest, ErrLocaleNotImplemented) 如果 implemented=false 且无 alias。
// 返回 (manifest, nil) 如果 implemented=true 或有 alias（caller 应自己看 LocaleAlias 字段）。
// 返回 (zero, err) 如果文件缺失或解析失败 —— 这是配置错误，不是"未实装"。
//
// **alias 不在这里解析**：调用方负责跟 alias 取 effective locale；上层 helper
// ResolveLocale 包了这层逻辑，bot package 应该调 ResolveLocale 而非直接 LoadManifest。
func LoadManifest(fsys fs.FS, path string) (Manifest, error) {
	m, err := loadManifestRaw(fsys, path)
	if err != nil {
		return Manifest{}, err
	}
	if m.LocaleAlias != "" {
		return m, nil
	}
	if !m.Implemented {
		return m, fmt.Errorf("%w: %s", ErrLocaleNotImplemented, m.Reason)
	}
	return m, nil
}

// loadManifestRaw 纯反序列化，不做 implemented/alias 检查。
// 仅 botcore 内部使用。
func loadManifestRaw(fsys fs.FS, path string) (Manifest, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest %s: %w", path, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	return m, nil
}

// ResolveLocale 解析 loc 的 manifest，跟随 LocaleAlias 重定向（最多一跳），
// 返回最终生效的 locale + 它的 manifest。
//
// pathFor: 给定 locale 字符串返回该 locale 的 manifest 文件路径。
//
// 行为：
//   - 读 primary manifest (path = pathFor(loc))。
//   - 如果 primary.LocaleAlias 非空：读 alias 的 manifest 并校验
//     （implemented 必须 true，不能再次有 alias，不能自指）。
//   - 如果 primary 无 alias：要求 implemented=true，否则 ErrLocaleNotImplemented。
//
// 返回的 effective locale 给 caller 后续拼 config.yaml / templates.toml 路径用。
func ResolveLocale(fsys fs.FS, loc string, pathFor func(string) string) (effective string, m Manifest, err error) {
	primary, err := loadManifestRaw(fsys, pathFor(loc))
	if err != nil {
		return "", Manifest{}, err
	}
	if primary.LocaleAlias == "" {
		if !primary.Implemented {
			return "", primary, fmt.Errorf("%w: %s", ErrLocaleNotImplemented, primary.Reason)
		}
		return loc, primary, nil
	}
	aliasLoc := primary.LocaleAlias
	if aliasLoc == loc {
		return "", Manifest{}, fmt.Errorf("locale %q aliases to itself", loc)
	}
	aliased, err := loadManifestRaw(fsys, pathFor(aliasLoc))
	if err != nil {
		return "", Manifest{}, fmt.Errorf("alias %s -> %s: %w", loc, aliasLoc, err)
	}
	if aliased.LocaleAlias != "" {
		return "", Manifest{}, fmt.Errorf("alias chain too deep: %s -> %s -> %s (alias 只允许一跳)",
			loc, aliasLoc, aliased.LocaleAlias)
	}
	if !aliased.Implemented {
		return "", aliased, fmt.Errorf("%w (via alias %s -> %s): %s",
			ErrLocaleNotImplemented, loc, aliasLoc, aliased.Reason)
	}
	return aliasLoc, aliased, nil
}
