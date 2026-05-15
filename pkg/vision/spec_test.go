package vision

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestValidateSpec(t *testing.T) {
	cases := []struct {
		name    string
		spec    TemplateSpec
		wantErr bool
	}{
		{"valid", TemplateSpec{
			File: "fish/x.png", Resolution: [2]int{1920, 1080},
			BBox: [4]int{100, 100, 200, 200},
		}, false},
		{"empty file", TemplateSpec{
			Resolution: [2]int{1920, 1080}, BBox: [4]int{0, 0, 10, 10},
		}, true},
		{"zero resolution", TemplateSpec{
			File: "f.png", Resolution: [2]int{0, 1080}, BBox: [4]int{0, 0, 10, 10},
		}, true},
		{"bbox not rect (x2<=x1)", TemplateSpec{
			File: "f.png", Resolution: [2]int{1920, 1080}, BBox: [4]int{100, 100, 100, 200},
		}, true},
		{"bbox out of bounds", TemplateSpec{
			File: "f.png", Resolution: [2]int{1920, 1080}, BBox: [4]int{0, 0, 1921, 100},
		}, true},
		{"negative bbox", TemplateSpec{
			File: "f.png", Resolution: [2]int{1920, 1080}, BBox: [4]int{-1, 0, 10, 10},
		}, true},
	}
	for _, c := range cases {
		err := validateSpec(c.spec)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err=%v wantErr=%v", c.name, err, c.wantErr)
		}
	}
}

func TestLoadTemplatesConfig_Valid(t *testing.T) {
	data := []byte(`
[[fish.hook_icon]]
file = "fish/hook_icon_1080.png"
resolution = [1920, 1080]
bbox = [1752, 944, 1797, 998]
note = "暗态"

[[fish.hook_icon]]
file = "fish/hook_icon_720.png"
resolution = [1280, 720]
bbox = [1168, 628, 1197, 666]

[[fish.hook_text]]
file = "fish/hook_text_1080.png"
resolution = [1920, 1080]
bbox = [776, 245, 1172, 282]

[[cook.hammer]]
file = "cook/hammer_1080.png"
resolution = [1920, 1080]
bbox = [58, 406, 148, 509]
`)
	cfg, warns, err := LoadTemplatesConfig(data)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(warns) != 0 {
		t.Errorf("expect 0 warnings, got %v", warns)
	}
	if len(cfg["fish"]["hook_icon"]) != 2 {
		t.Errorf("fish.hook_icon: want 2, got %d", len(cfg["fish"]["hook_icon"]))
	}
	if len(cfg["fish"]["hook_text"]) != 1 {
		t.Errorf("fish.hook_text: want 1, got %d", len(cfg["fish"]["hook_text"]))
	}
	if len(cfg["cook"]["hammer"]) != 1 {
		t.Errorf("cook.hammer: want 1, got %d", len(cfg["cook"]["hammer"]))
	}
	// Source 应该被设为 SourceEmbed
	if cfg["fish"]["hook_icon"][0].Source != SourceEmbed {
		t.Error("expect SourceEmbed for embed-loaded specs")
	}
}

func TestLoadTemplatesConfig_BadEntrySkipped(t *testing.T) {
	data := []byte(`
[[fish.hook_text]]
file = "fish/good.png"
resolution = [1920, 1080]
bbox = [100, 100, 200, 200]

[[fish.hook_text]]
file = ""
resolution = [1920, 1080]
bbox = [0, 0, 10, 10]

[[fish.hook_text]]
file = "fish/bad_bbox.png"
resolution = [1920, 1080]
bbox = [0, 0, 9999, 100]
`)
	cfg, warns, err := LoadTemplatesConfig(data)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(cfg["fish"]["hook_text"]) != 1 {
		t.Errorf("only the valid one should survive, got %d", len(cfg["fish"]["hook_text"]))
	}
	if len(warns) != 2 {
		t.Errorf("expect 2 warnings, got %d: %v", len(warns), warns)
	}
}

func TestLoadTemplatesConfig_ParseError(t *testing.T) {
	_, _, err := LoadTemplatesConfig([]byte("this is not toml ===="))
	if err == nil {
		t.Error("expect parse error")
	}
}

func TestMergeUserOverrides_AddsNewResolution(t *testing.T) {
	cfg := TemplatesConfig{
		"fish": {
			"hook_text": []TemplateSpec{
				{File: "fish/hook_text_1080.png", Resolution: [2]int{1920, 1080},
					BBox: [4]int{100, 100, 200, 200}, Source: SourceEmbed},
			},
		},
	}
	user := []byte(`
[[fish.hook_text]]
file = "fish/my_1440.png"
resolution = [2560, 1440]
bbox = [200, 200, 300, 300]
`)
	warns, err := cfg.MergeUserOverrides("fish", user)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(warns) != 0 {
		t.Errorf("expect no warnings, got %v", warns)
	}
	if len(cfg["fish"]["hook_text"]) != 2 {
		t.Errorf("expect 2 specs after merge, got %d", len(cfg["fish"]["hook_text"]))
	}
	// 新加的应该是 SourceUser
	added := cfg["fish"]["hook_text"][1]
	if added.Source != SourceUser {
		t.Errorf("expect SourceUser, got %d", added.Source)
	}
	if added.Resolution != [2]int{2560, 1440} {
		t.Errorf("expect 1440p added")
	}
}

func TestMergeUserOverrides_DuplicateResolutionSkipped(t *testing.T) {
	cfg := TemplatesConfig{
		"fish": {
			"hook_text": []TemplateSpec{
				{File: "fish/hook_text_1080.png", Resolution: [2]int{1920, 1080},
					BBox: [4]int{100, 100, 200, 200}, Source: SourceEmbed},
			},
		},
	}
	user := []byte(`
[[fish.hook_text]]
file = "fish/my_1080.png"
resolution = [1920, 1080]
bbox = [50, 50, 150, 150]
`)
	warns, err := cfg.MergeUserOverrides("fish", user)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(warns) == 0 {
		t.Error("expect warning about duplicate resolution")
	}
	if len(cfg["fish"]["hook_text"]) != 1 {
		t.Errorf("expect duplicate skipped, still 1 spec")
	}
}

func TestMergeUserOverrides_ParseError(t *testing.T) {
	cfg := TemplatesConfig{}
	_, err := cfg.MergeUserOverrides("fish", []byte("garbage =="))
	if err == nil {
		t.Error("expect parse error")
	}
}

func TestMergeUserOverrides_NewSlot(t *testing.T) {
	// embed 没有的 slot，用户加了——应该被接受
	cfg := TemplatesConfig{"fish": map[string][]TemplateSpec{}}
	user := []byte(`
[[fish.brand_new_slot]]
file = "fish/x.png"
resolution = [1920, 1080]
bbox = [0, 0, 100, 100]
`)
	_, err := cfg.MergeUserOverrides("fish", user)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(cfg["fish"]["brand_new_slot"]) != 1 {
		t.Error("new slot from user should be accepted")
	}
}

func TestLoadPNGForSpec_UserSourceReadsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "x.png")
	pngBytes := []byte{0x89, 'P', 'N', 'G'}
	if err := os.WriteFile(fpath, pngBytes, 0644); err != nil {
		t.Fatal(err)
	}
	spec := TemplateSpec{File: fpath, Source: SourceUser}
	got, err := LoadPNGForSpec(nil, spec) // fsys ignored for SourceUser
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if string(got) != string(pngBytes) {
		t.Errorf("content mismatch")
	}
}

func TestLoadPNGForSpec_UnknownSource(t *testing.T) {
	spec := TemplateSpec{File: "x.png", Source: SourceType(99)}
	_, err := LoadPNGForSpec(nil, spec)
	if err == nil {
		t.Error("expect error for unknown source")
	}
}

func TestLoadPNGForSpec_EmbedReadsFromFsys(t *testing.T) {
	fsys := fstest.MapFS{
		"hook_icon_1080.png": &fstest.MapFile{Data: []byte{0x89, 'P', 'N', 'G'}},
	}
	spec := TemplateSpec{File: "hook_icon_1080.png", Source: SourceEmbed}
	got, err := LoadPNGForSpec(fsys, spec)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if string(got) != string([]byte{0x89, 'P', 'N', 'G'}) {
		t.Errorf("content mismatch")
	}
}

func TestLoadPNGForSpec_EmbedNilFsysFails(t *testing.T) {
	spec := TemplateSpec{File: "x.png", Source: SourceEmbed}
	_, err := LoadPNGForSpec(nil, spec)
	if err == nil {
		t.Error("expect error when fsys is nil for SourceEmbed")
	}
}
