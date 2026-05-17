package template

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTemplateMeta_JSONRoundTrip(t *testing.T) {
	src := TemplateMeta{
		Name:               "技能1 冷却完成",
		Description:        "右下技能1冰红光",
		RecordedResolution: [2]int{1920, 1080},
		SHA256:             "abc123",
		Width:              64,
		Height:             32,
		Region:             [4]float32{0.85, 0.92, 0.05, 0.04},
		CreatedAt:          mustTime("2026-05-15T05:00:00Z"),
		Origin:             TemplateOrigin{Kind: OriginKindScreenshot},
	}
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got TemplateMeta
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Compare field-by-field since Tags slice makes == comparison fail
	if got.Name != src.Name || got.Description != src.Description ||
		got.RecordedResolution != src.RecordedResolution ||
		got.SHA256 != src.SHA256 || got.Width != src.Width ||
		got.Height != src.Height || got.Region != src.Region ||
		got.CreatedAt != src.CreatedAt || got.Origin != src.Origin {
		t.Errorf("round-trip mismatch:\n  want %#v\n  got  %#v", src, got)
	}
}

func TestTemplateIndex_EmptyDefault(t *testing.T) {
	var idx TemplateIndex
	b, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// 允许 null 或 {} — Go json 库对 nil map 输出 null
	if string(b) != `{"schemaVersion":0,"templates":null}` && string(b) != `{"schemaVersion":0,"templates":{}}` {
		t.Errorf("empty index marshal got: %s", b)
	}
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
