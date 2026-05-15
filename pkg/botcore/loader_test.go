package botcore

import (
	"testing"
	"testing/fstest"
)

type sampleCfg struct {
	A int `yaml:"a"`
	B int `yaml:"b"`
	C int `yaml:"c"`
}

func TestLoadYAMLWithOverride_EmbedOnly(t *testing.T) {
	embed := fstest.MapFS{
		"cfg.yaml": &fstest.MapFile{
			Data: []byte("a: 1\nb: 2\nc: 3\n"),
		},
	}
	var cfg sampleCfg
	if err := LoadYAMLWithOverride(embed, "cfg.yaml", "no-such-override.yaml", &cfg); err != nil {
		t.Fatalf("err=%v", err)
	}
	if cfg.A != 1 || cfg.B != 2 || cfg.C != 3 {
		t.Errorf("got %+v, want {1,2,3}", cfg)
	}
}

func TestLoadYAMLWithOverride_EmbedMissing(t *testing.T) {
	embed := fstest.MapFS{}
	var cfg sampleCfg
	err := LoadYAMLWithOverride(embed, "no-such.yaml", "no-such-override.yaml", &cfg)
	if err == nil {
		t.Fatal("expected error for missing embed")
	}
}

func TestLoadYAMLWithOverride_EmbedParseFail(t *testing.T) {
	embed := fstest.MapFS{
		"cfg.yaml": &fstest.MapFile{
			Data: []byte("a: not-an-int\n"),
		},
	}
	var cfg sampleCfg
	err := LoadYAMLWithOverride(embed, "cfg.yaml", "no-such-override.yaml", &cfg)
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}
