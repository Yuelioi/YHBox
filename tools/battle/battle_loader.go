package battle

import (
	"embed"
	"fmt"
	"io/fs"

	"yhbox/pkg/botcore"
	"yhbox/pkg/locale"
)

//go:embed configs
var configsFS embed.FS

var (
	templatesFS   fs.FS
	templatesTOML []byte
)

// LoadConfig 启动期 main.go 调一次。失败 → bot 不可用。
//
// battle 没有可调参数（无 config.yaml），所以流程只有：
//  1. loadManifest:  configs/<loc>/manifest.yaml；如果该 locale 走 locale_alias，
//                    effective 是 alias 指向的 locale
//  2. loadTemplates: effective locale 的 templates.toml + templates/ 子树挂到包级，
//                    detect.go 用 battle.Assets() 取
//  3. Validate:      启动期 fail-fast 检查 templates 加载成功
func LoadConfig(loc locale.Locale) error {
	effective, _, err := loadManifest(loc)
	if err != nil {
		return err
	}
	if err := loadTemplates(effective); err != nil {
		return fmt.Errorf("battle.LoadConfig(%s): %w", loc, err)
	}
	return Validate()
}

// loadManifest 读 configs/<loc>/manifest.yaml；如果该 locale 走 locale_alias 引用
// 别的 locale，返回 alias 指向的 effective locale。
func loadManifest(loc locale.Locale) (locale.Locale, botcore.Manifest, error) {
	eff, m, err := botcore.ResolveLocale(configsFS, string(loc), func(l string) string {
		return fmt.Sprintf("configs/%s/manifest.yaml", l)
	})
	return locale.Locale(eff), m, err
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

func Validate() error {
	if len(templatesTOML) == 0 {
		return fmt.Errorf("battle templates.toml 为空")
	}
	if templatesFS == nil {
		return fmt.Errorf("battle templatesFS 为 nil")
	}
	return nil
}

func Assets() botcore.RuntimeAssets {
	return botcore.RuntimeAssets{
		Templates:     templatesFS,
		TemplatesTOML: templatesTOML,
	}
}
