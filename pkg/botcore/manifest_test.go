package botcore

import (
	"errors"
	"testing"
	"testing/fstest"
)

func TestLoadManifest_Implemented(t *testing.T) {
	fsys := fstest.MapFS{
		"m.yaml": &fstest.MapFile{Data: []byte("version: 1\nimplemented: true\nreason: \"\"\n")},
	}
	m, err := LoadManifest(fsys, "m.yaml")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !m.Implemented || m.Version != 1 {
		t.Errorf("got %+v", m)
	}
}

func TestLoadManifest_NotImplemented(t *testing.T) {
	fsys := fstest.MapFS{
		"m.yaml": &fstest.MapFile{Data: []byte("version: 1\nimplemented: false\nreason: \"模板未采集\"\n")},
	}
	_, err := LoadManifest(fsys, "m.yaml")
	if !errors.Is(err, ErrLocaleNotImplemented) {
		t.Fatalf("expected ErrLocaleNotImplemented, got %v", err)
	}
}

func TestLoadManifest_Missing(t *testing.T) {
	fsys := fstest.MapFS{}
	_, err := LoadManifest(fsys, "nope.yaml")
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
	if errors.Is(err, ErrLocaleNotImplemented) {
		t.Error("missing file should NOT be ErrLocaleNotImplemented (it's a config error)")
	}
}

// 同 bot 不同 locale 下 manifest 的常见形态，给 ResolveLocale 测试用。
func aliasFS() fstest.MapFS {
	return fstest.MapFS{
		"zh/manifest.yaml": &fstest.MapFile{
			Data: []byte("version: 1\nimplemented: true\nreason: \"\"\n"),
		},
		"en/manifest.yaml": &fstest.MapFile{
			Data: []byte("version: 1\nimplemented: true\nlocale_alias: zh\n"),
		},
		"ja/manifest.yaml": &fstest.MapFile{
			Data: []byte("version: 1\nimplemented: false\nreason: \"未实装\"\n"),
		},
		"ko/manifest.yaml": &fstest.MapFile{
			// 链式 alias —— 不允许
			Data: []byte("version: 1\nimplemented: true\nlocale_alias: en\n"),
		},
		"self/manifest.yaml": &fstest.MapFile{
			// 自指 —— 不允许
			Data: []byte("version: 1\nimplemented: true\nlocale_alias: self\n"),
		},
		"orphan/manifest.yaml": &fstest.MapFile{
			// alias 指向不存在的 locale
			Data: []byte("version: 1\nimplemented: true\nlocale_alias: xx\n"),
		},
		"bad/manifest.yaml": &fstest.MapFile{
			// alias 指向未实装的 locale
			Data: []byte("version: 1\nimplemented: true\nlocale_alias: ja\n"),
		},
	}
}

func pathFor(l string) string { return l + "/manifest.yaml" }

func TestResolveLocale_Direct(t *testing.T) {
	eff, m, err := ResolveLocale(aliasFS(), "zh", pathFor)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if eff != "zh" || !m.Implemented {
		t.Errorf("got eff=%q m=%+v", eff, m)
	}
}

func TestResolveLocale_FollowsAlias(t *testing.T) {
	eff, m, err := ResolveLocale(aliasFS(), "en", pathFor)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if eff != "zh" {
		t.Errorf("expected effective=zh, got %q", eff)
	}
	if !m.Implemented {
		t.Errorf("aliased manifest should be implemented, got %+v", m)
	}
}

func TestResolveLocale_NotImplemented(t *testing.T) {
	_, _, err := ResolveLocale(aliasFS(), "ja", pathFor)
	if !errors.Is(err, ErrLocaleNotImplemented) {
		t.Fatalf("expected ErrLocaleNotImplemented, got %v", err)
	}
}

func TestResolveLocale_AliasChainTooDeep(t *testing.T) {
	_, _, err := ResolveLocale(aliasFS(), "ko", pathFor)
	if err == nil {
		t.Fatal("expected error for two-hop alias chain")
	}
	if errors.Is(err, ErrLocaleNotImplemented) {
		t.Error("chain-too-deep should NOT be ErrLocaleNotImplemented")
	}
}

func TestResolveLocale_SelfAlias(t *testing.T) {
	_, _, err := ResolveLocale(aliasFS(), "self", pathFor)
	if err == nil {
		t.Fatal("expected error for self-alias")
	}
}

func TestResolveLocale_AliasTargetMissing(t *testing.T) {
	_, _, err := ResolveLocale(aliasFS(), "orphan", pathFor)
	if err == nil {
		t.Fatal("expected error when alias target manifest missing")
	}
}

func TestResolveLocale_AliasToUnimplemented(t *testing.T) {
	_, _, err := ResolveLocale(aliasFS(), "bad", pathFor)
	if !errors.Is(err, ErrLocaleNotImplemented) {
		t.Fatalf("alias to unimplemented should surface as ErrLocaleNotImplemented, got %v", err)
	}
}
