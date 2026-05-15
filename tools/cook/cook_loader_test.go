package cook

import (
	"testing"

	"yhbox/pkg/locale"
)

func TestLoadConfigZh(t *testing.T) {
	if err := LoadConfig(locale.Zh); err != nil {
		t.Fatalf("LoadConfig(zh): %v", err)
	}
	if len(Configs) == 0 {
		t.Error("Configs map 加载后为空")
	}
	// 已知支持的分辨率应该都在
	if _, ok := PickHammer(1920, 1080); !ok {
		t.Error("1920x1080 hammer 坐标缺失")
	}
	if _, ok := PickHammer(1280, 720); !ok {
		t.Error("1280x720 hammer 坐标缺失")
	}
}

func TestPickHammer_UnknownResolution(t *testing.T) {
	if err := LoadConfig(locale.Zh); err != nil {
		t.Fatalf("LoadConfig(zh): %v", err)
	}
	if _, ok := PickHammer(3840, 2160); ok {
		t.Error("4K 没配置应该返 ok=false")
	}
}

func TestLoadConfigEn_AliasZh(t *testing.T) {
	if err := LoadConfig(locale.En); err != nil {
		t.Fatalf("LoadConfig(en): %v", err)
	}
	if _, ok := PickHammer(1920, 1080); !ok {
		t.Error("EN alias zh 后 1920x1080 应该可用")
	}
}
