package rhythm

import (
	"embed"
	"fmt"
	"image"

	"yhbox/pkg/botcore"
	"yhbox/pkg/locale"
)

//go:embed configs
var configsFS embed.FS

// yamlConfig 对应 configs/<locale>/config.yaml。顶层分成 capture/matching/timing 三组，
// 让所有 bot 用同一种分组习惯（即使某组当前为空），方便将来 GUI 调参面板做通用 form。
//   - capture: 截图来源相关（rhythm 直接全屏抓，本组当前空）
//   - matching: 4 轨 ROI 像素坐标（亮度阈值/retrigger 间隔走包级常量不外置）
//   - timing:  时序参数（rhythm 留 Go const，本组当前空）
type yamlConfig struct {
	Version  int            `yaml:"version"`
	Capture  map[string]any `yaml:"capture"`
	Matching yamlMatching   `yaml:"matching"`
	Timing   map[string]any `yaml:"timing"`
}

type yamlMatching struct {
	Resolutions map[string]yamlResConfig `yaml:"resolutions"`
}

type yamlResConfig struct {
	ROIs [4][4]int `yaml:"rois"` // 每行 [x1, y1, x2, y2]
}

// LoadConfig 启动期 main.go 调一次。失败 → bot 不可用。
//
// 流程：
//  1. loadManifest: 读 configs/<loc>/manifest.yaml 看该 locale 是否实装；
//     若 manifest 写了 locale_alias，effective 就是 alias 指向的 locale
//     （rhythm 跨语言无差异，en 的 manifest alias 到 zh）
//  2. loadConfig:   用 effective locale 拼 config.yaml 路径，embed 默认值 +
//     exe 同目录 override merge → 反射进包级 Configs / HSVRanges
//  3. Validate:     启动期 fail-fast 检查 ROI 在分辨率范围内、HSV 合法、阈值有序
//
// rhythm 没有视觉模板，省略加载 templates 那一步。
func LoadConfig(loc locale.Locale) error {
	effective, _, err := loadManifest(loc)
	if err != nil {
		return err
	}
	if err := loadConfig(effective); err != nil {
		return fmt.Errorf("rhythm.LoadConfig(%s): %w", loc, err)
	}
	if err := Validate(); err != nil {
		return fmt.Errorf("rhythm.LoadConfig(%s) validate: %w", loc, err)
	}
	return nil
}

// loadManifest 读 configs/<loc>/manifest.yaml；如果该 locale 用 locale_alias
// 引用了另一份 locale（如 en → zh），返回 alias 指向的 locale，后续步骤拿这个
// effective locale 去读 config.yaml / templates.
func loadManifest(loc locale.Locale) (locale.Locale, botcore.Manifest, error) {
	eff, m, err := botcore.ResolveLocale(configsFS, string(loc), func(l string) string {
		return fmt.Sprintf("configs/%s/manifest.yaml", l)
	})
	return locale.Locale(eff), m, err
}

// loadConfig embed + exe-dir override merge → 反射进包级 state。
func loadConfig(loc locale.Locale) error {
	var doc yamlConfig
	embedPath := fmt.Sprintf("configs/%s/config.yaml", loc)
	overrideRelPath := fmt.Sprintf("configs/rhythm/%s/config.yaml", loc)
	if err := botcore.LoadYAMLWithOverride(configsFS, embedPath, overrideRelPath, &doc); err != nil {
		return err
	}
	return applyYAMLConfig(&doc)
}

// applyYAMLConfig 反序列化结构 → 包级 Configs。
// 亮度阈值 / retrigger / KeyHold 走包级常量不外置。
func applyYAMLConfig(doc *yamlConfig) error {
	newConfigs := make(map[ResolutionKey]ResConfig, len(doc.Matching.Resolutions))
	for keyStr, rc := range doc.Matching.Resolutions {
		var w, h int
		if _, err := fmt.Sscanf(keyStr, "%dx%d", &w, &h); err != nil {
			return fmt.Errorf("invalid resolution key %q (want like \"1920x1080\"): %w", keyStr, err)
		}
		var rois [4]image.Rectangle
		for i, r := range rc.ROIs {
			rois[i] = image.Rect(r[0], r[1], r[2], r[3])
		}
		newConfigs[ResolutionKey{W: w, H: h}] = ResConfig{ROIs: rois}
	}
	Configs = newConfigs
	return nil
}

// Validate 启动期 fail-fast 检查 ROI 在分辨率范围内。
func Validate() error {
	if len(Configs) == 0 {
		return fmt.Errorf("no resolutions configured")
	}
	for key, cfg := range Configs {
		for i, r := range cfg.ROIs {
			if r.Empty() {
				return fmt.Errorf("res %v: ROI[%d] is empty", key, i)
			}
			if r.Min.X < 0 || r.Min.Y < 0 || r.Max.X > key.W || r.Max.Y > key.H {
				return fmt.Errorf("res %v: ROI[%d]=%v 越出 %dx%d", key, i, r, key.W, key.H)
			}
		}
	}
	return nil
}

// Assets 返回 rhythm 的 RuntimeAssets。rhythm 没有视觉模板，两个字段都是 nil/空。
// caller 想读 rhythm 的强类型配置走包级 Configs / HSVRanges 即可，那些不在 RuntimeAssets 里。
//
// 现在 rhythm 用不上 Assets()，留这个函数是为了跟 fish/cook/battle 的接口对齐——未来 rhythm
// 加 OCR / locale 资产之类才需要从这里返回。
func Assets() botcore.RuntimeAssets {
	return botcore.RuntimeAssets{
		Templates:     nil,
		TemplatesTOML: nil,
	}
}
