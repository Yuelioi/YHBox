package cook

import (
	"embed"
	"fmt"

	"yhbox/pkg/botcore"
	"yhbox/pkg/locale"
)

//go:embed configs
var configsFS embed.FS

// yamlConfig 对应 configs/<locale>/config.yaml。
// cook 是固定坐标 click，没视觉模板，只需读 hammer 像素坐标。
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
	Hammer [2]int `yaml:"hammer"` // [x, y] 客户区像素坐标
}

// LoadConfig 启动期 main.go 调一次。失败 → bot 不可用。
//
//  1. loadManifest:  configs/<loc>/manifest.yaml；locale_alias 时返回 effective
//  2. loadConfig:    effective locale 的 config.yaml，embed + exe-dir override merge
//                    → 填包级 Configs map
//  3. Validate:      启动期 fail-fast
func LoadConfig(loc locale.Locale) error {
	effective, _, err := loadManifest(loc)
	if err != nil {
		return err
	}
	if err := loadConfig(effective); err != nil {
		return fmt.Errorf("cook.LoadConfig(%s): %w", loc, err)
	}
	return Validate()
}

func loadManifest(loc locale.Locale) (locale.Locale, botcore.Manifest, error) {
	eff, m, err := botcore.ResolveLocale(configsFS, string(loc), func(l string) string {
		return fmt.Sprintf("configs/%s/manifest.yaml", l)
	})
	return locale.Locale(eff), m, err
}

func loadConfig(loc locale.Locale) error {
	var doc yamlConfig
	embedPath := fmt.Sprintf("configs/%s/config.yaml", loc)
	overrideRelPath := fmt.Sprintf("configs/cook/%s/config.yaml", loc)
	if err := botcore.LoadYAMLWithOverride(configsFS, embedPath, overrideRelPath, &doc); err != nil {
		return err
	}
	newConfigs := make(map[ResolutionKey]HammerPoint, len(doc.Matching.Resolutions))
	for keyStr, rc := range doc.Matching.Resolutions {
		var w, h int
		if _, err := fmt.Sscanf(keyStr, "%dx%d", &w, &h); err != nil {
			return fmt.Errorf("invalid resolution key %q: %w", keyStr, err)
		}
		newConfigs[ResolutionKey{W: w, H: h}] = HammerPoint{X: rc.Hammer[0], Y: rc.Hammer[1]}
	}
	Configs = newConfigs
	return nil
}

func Validate() error {
	if len(Configs) == 0 {
		return fmt.Errorf("no hammer coordinates configured")
	}
	for key, p := range Configs {
		if p.X <= 0 || p.X >= key.W || p.Y <= 0 || p.Y >= key.H {
			return fmt.Errorf("res %v: hammer point (%d,%d) 越出客户区", key, p.X, p.Y)
		}
	}
	return nil
}

// Assets 返回空 RuntimeAssets——cook 没有视觉模板。保留接口跟其它 bot 对齐。
func Assets() botcore.RuntimeAssets {
	return botcore.RuntimeAssets{}
}
