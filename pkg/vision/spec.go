package vision

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// SourceType 标记 NamedTemplate 的来源，决定 PNG 加载时的路径基底。
type SourceType int

const (
	SourceEmbed SourceType = iota // 路径基底 = "assets/"，走 yhbox.AssetBytes
	SourceUser                    // 路径基底 = exe 同目录 "templates/"，走 os.ReadFile
)

// TemplateSpec 是 templates.toml 里 `[[<tool>.<slot>]]` 一项的反序列化结构。
type TemplateSpec struct {
	File          string `toml:"file"`
	Resolution    [2]int `toml:"resolution"`
	BBox          [4]int `toml:"bbox"`
	Note          string `toml:"note,omitempty"`
	ResolutionTag string `toml:"resolution_tag,omitempty"`

	// Source 内部字段（不从 TOML 反序列化），由 LoadTemplatesConfig / MergeUserOverrides 设置。
	Source SourceType `toml:"-"`
}

// TemplatesConfig: tool 名 → slot 名 → []TemplateSpec
// 例：cfg["fish"]["hook_icon"] → []TemplateSpec
type TemplatesConfig map[string]map[string][]TemplateSpec

// validateSpec 校验单条 spec 的合法性。
//   - File 非空
//   - Resolution 两个分量都 > 0
//   - BBox 是合法矩形（x2>x1, y2>y1, 非负）
//   - BBox 在 Resolution 边界内
func validateSpec(s TemplateSpec) error {
	if s.File == "" {
		return fmt.Errorf("缺 file")
	}
	if s.Resolution[0] <= 0 || s.Resolution[1] <= 0 {
		return fmt.Errorf("resolution 非法 %v（要求两个分量都 >0）", s.Resolution)
	}
	if s.BBox[0] < 0 || s.BBox[1] < 0 {
		return fmt.Errorf("bbox 含负数 %v", s.BBox)
	}
	if s.BBox[2] <= s.BBox[0] || s.BBox[3] <= s.BBox[1] {
		return fmt.Errorf("bbox 非法矩形 %v（要求 x2>x1 且 y2>y1）", s.BBox)
	}
	if s.BBox[2] > s.Resolution[0] || s.BBox[3] > s.Resolution[1] {
		return fmt.Errorf("bbox %v 超出 resolution %v 边界", s.BBox, s.Resolution)
	}
	return nil
}

// LoadTemplatesConfig 解析 embed 的 templates.toml 字节流。
//   - 整个文件解析失败 → 返回 err（embed 解析失败应该早炸）
//   - 单条 spec 校验失败 → 跳过 + 加入 warnings
//   - 所有合法 spec 的 Source 设为 SourceEmbed
func LoadTemplatesConfig(data []byte) (TemplatesConfig, []string, error) {
	// 第一步：用 map[string]map[string][]TemplateSpec 反序列化
	var raw TemplatesConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("parse templates.toml: %w", err)
	}

	// 第二步：遍历 + 校验 + 标记 Source
	out := TemplatesConfig{}
	var warnings []string
	for tool, slots := range raw {
		out[tool] = map[string][]TemplateSpec{}
		for slot, specs := range slots {
			for _, s := range specs {
				if err := validateSpec(s); err != nil {
					warnings = append(warnings,
						fmt.Sprintf("[%s.%s] file=%q: %v", tool, slot, s.File, err))
					continue
				}
				s.Source = SourceEmbed
				out[tool][slot] = append(out[tool][slot], s)
			}
		}
	}
	return out, warnings, nil
}

// MergeUserOverrides 把用户外挂 toml 追加到 cfg 里。
//   - tool: 当前文件描述的工具（"fish" / "cook"）— 用户外挂 toml 的顶层 key 必须匹配
//   - userTomlBytes: 用户的 templates/<tool>.toml 内容
//   - 不覆写 embed 同 (slot, resolution) 条目，跳过 + warning
//   - 单条 spec 校验失败跳过 + warning
//   - 整个 toml 解析失败返回 err（让调用方决定 WARN + 跳过外挂层）
//   - 所有合法 spec 的 Source 设为 SourceUser
func (c TemplatesConfig) MergeUserOverrides(tool string, userTomlBytes []byte) ([]string, error) {
	var raw TemplatesConfig
	if err := toml.Unmarshal(userTomlBytes, &raw); err != nil {
		return nil, fmt.Errorf("parse user templates: %w", err)
	}

	// 确保目标 tool 在 cfg 里有 map（即便用户写的 tool key 跟参数不一致，也只处理参数指定的）
	if c[tool] == nil {
		c[tool] = map[string][]TemplateSpec{}
	}

	var warnings []string
	for slot, specs := range raw[tool] {
		for _, s := range specs {
			if err := validateSpec(s); err != nil {
				warnings = append(warnings,
					fmt.Sprintf("[%s.%s] file=%q: %v (跳过)", tool, slot, s.File, err))
				continue
			}
			// 检查 (slot, resolution) 重复
			dup := false
			for _, existing := range c[tool][slot] {
				if existing.Resolution == s.Resolution {
					warnings = append(warnings, fmt.Sprintf(
						"[%s.%s] file=%q resolution=%v: 重复 embed 已有的 file=%q，跳过",
						tool, slot, s.File, s.Resolution, existing.File))
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			s.Source = SourceUser
			c[tool][slot] = append(c[tool][slot], s)
		}
	}
	return warnings, nil
}

// LoadPNGForSpec 从 fsys 读 spec.File 的字节流：
//   - SourceEmbed: fs.ReadFile(fsys, spec.File) —— fsys 由 caller (bot) 提供
//   - SourceUser:  os.ReadFile(spec.File) —— spec.File 是 caller 已解析好的完整路径
//
// caller 拿不到合适的 fsys 但又是 SourceEmbed 类型 → fsys=nil 返回 error。
func LoadPNGForSpec(fsys fs.FS, spec TemplateSpec) ([]byte, error) {
	switch spec.Source {
	case SourceEmbed:
		if fsys == nil {
			return nil, fmt.Errorf("LoadPNGForSpec: fsys 为 nil（caller 应该提供 bot-local FS）")
		}
		return fs.ReadFile(fsys, spec.File)
	case SourceUser:
		return os.ReadFile(spec.File)
	default:
		return nil, fmt.Errorf("unknown source %d", spec.Source)
	}
}
