package template

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTemplateOriginKinds(t *testing.T) {
	cases := []TemplateOrigin{
		{Kind: OriginKindScreenshot, SourceID: ""},
		{Kind: OriginKindLibrary, SourceID: "library-tpl-key"},
		{Kind: OriginKindImported, SourceID: "sg-uuid"},
		{Kind: OriginKindEmbedded, SourceID: "sg-uuid"},
	}
	for _, o := range cases {
		b, err := json.Marshal(o)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var back TemplateOrigin
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if back != o {
			t.Errorf("round-trip: got %+v want %+v", back, o)
		}
	}
}

func TestTemplateMetaHasOriginAndTags(t *testing.T) {
	m := TemplateMeta{
		Name:   "x",
		Width:  10,
		Height: 10,
		Tags:   []string{"上钩", "钓鱼"},
		Origin: TemplateOrigin{Kind: OriginKindLibrary, SourceID: "lib-key"},
	}
	b, _ := json.Marshal(m)
	if !strings.Contains(string(b), `"origin"`) {
		t.Errorf("origin missing: %s", string(b))
	}
	if !strings.Contains(string(b), `"tags"`) {
		t.Errorf("tags missing: %s", string(b))
	}
}
