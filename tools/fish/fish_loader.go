package fish

import (
	"embed"
	"fmt"
	"io/fs"

	"yhbox/pkg/botcore"
	"yhbox/pkg/locale"
)

//go:embed configs
var configsFS embed.FS

// 运行时填的字段（LoadConfig 后非 nil）
var (
	templatesFS   fs.FS
	templatesTOML []byte
)

// yamlBarROI 是 capture.bar_rois 单条 entry 的反序列化结构。
type yamlBarROI struct {
	Resolution [2]int `yaml:"resolution"`
	BBox       [4]int `yaml:"bbox"`
}

type yamlCapture struct {
	BarROIs []yamlBarROI `yaml:"bar_rois"`
}

// yamlConfig 对应 configs/<locale>/config.yaml。顶层分 capture/matching/timing
// 三组，跟所有 bot 用同一种分组（即使某组当前为空）。fish 数据全在 capture.bar_rois。
type yamlConfig struct {
	Version  int            `yaml:"version"`
	Capture  yamlCapture    `yaml:"capture"`
	Matching map[string]any `yaml:"matching"`
	Timing   map[string]any `yaml:"timing"`
}

// LoadConfig 启动期 main.go 调一次。失败 → bot 不可用。
//
// 流程：
//  1. loadManifest:  configs/<loc>/manifest.yaml；如果该 locale 走 locale_alias
//     引用别的 locale，effective 就是 alias 指向的那个
//  2. loadConfig:    effective locale 的 config.yaml，embed + exe-dir override merge
//                    → 反射进包级 BarROIs
//  3. loadTemplates: effective locale 的 templates.toml + templates/ 子树挂到包级
//                    templatesTOML + templatesFS（detect.go 用 fish.Assets() 取）
//  4. Validate:      启动期 fail-fast 检查 bar_rois 不为空 / bbox 合法 / 模板可加载
func LoadConfig(loc locale.Locale) error {
	effective, _, err := loadManifest(loc)
	if err != nil {
		return err
	}
	if err := loadConfig(effective); err != nil {
		return fmt.Errorf("fish.LoadConfig(%s): %w", loc, err)
	}
	if err := loadTemplates(effective); err != nil {
		return fmt.Errorf("fish.LoadConfig(%s): %w", loc, err)
	}
	return Validate()
}

// loadManifest 读 configs/<loc>/manifest.yaml；如果该 locale 用 locale_alias
// 引用了另一份 locale，返回 alias 指向的 locale，后续步骤拿这个 effective
// locale 去拼 config.yaml / templates 路径。
func loadManifest(loc locale.Locale) (locale.Locale, botcore.Manifest, error) {
	eff, m, err := botcore.ResolveLocale(configsFS, string(loc), func(l string) string {
		return fmt.Sprintf("configs/%s/manifest.yaml", l)
	})
	return locale.Locale(eff), m, err
}

func loadConfig(loc locale.Locale) error {
	var doc yamlConfig
	embedPath := fmt.Sprintf("configs/%s/config.yaml", loc)
	overrideRelPath := fmt.Sprintf("configs/fish/%s/config.yaml", loc)
	if err := botcore.LoadYAMLWithOverride(configsFS, embedPath, overrideRelPath, &doc); err != nil {
		return err
	}
	BarROIs = make([]BarROI, len(doc.Capture.BarROIs))
	for i, r := range doc.Capture.BarROIs {
		BarROIs[i] = BarROI{Resolution: r.Resolution, BBox: r.BBox}
	}
	return nil
}

func loadTemplates(loc locale.Locale) error {
	tomlBytes, err := fs.ReadFile(configsFS, fmt.Sprintf("configs/%s/templates.toml", loc))
	if err != nil {
		return fmt.Errorf("read templates.toml: %w", err)
	}
	templatesTOML = tomlBytes
	templatesSub, err := fs.Sub(configsFS, fmt.Sprintf("configs/%s/templates", loc))
	if err != nil {
		return fmt.Errorf("sub templates/: %w", err)
	}
	templatesFS = templatesSub
	return nil
}

// Validate 启动期 fail-fast 检查 fish 配置 + 模板可加载性。
func Validate() error {
	if len(BarROIs) == 0 {
		return fmt.Errorf("no bar_rois configured")
	}
	for i, r := range BarROIs {
		if r.BBox[0] >= r.BBox[2] || r.BBox[1] >= r.BBox[3] {
			return fmt.Errorf("bar_rois[%d] bbox %v 无效", i, r.BBox)
		}
		if r.BBox[2] > r.Resolution[0] || r.BBox[3] > r.Resolution[1] {
			return fmt.Errorf("bar_rois[%d] bbox %v 越出分辨率 %v", i, r.BBox, r.Resolution)
		}
	}
	if len(templatesTOML) == 0 {
		return fmt.Errorf("templates.toml 为空")
	}
	if templatesFS == nil {
		return fmt.Errorf("templatesFS 为 nil")
	}
	return nil
}

// Assets 返回 fish 当前 locale 的运行时视觉资源（templates.toml bytes + templates/ 子树）。
// detect.go 用这俩去构造 Detector。fish 的强类型配置（BarROIs）不在这里——直接读包级 BarROIs。
func Assets() botcore.RuntimeAssets {
	return botcore.RuntimeAssets{
		Templates:     templatesFS,
		TemplatesTOML: templatesTOML,
	}
}
