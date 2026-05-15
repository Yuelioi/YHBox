package battle

import (
	"testing"

	"yhbox/pkg/locale"
)

func TestLoadConfigZh(t *testing.T) {
	if err := LoadConfig(locale.Zh); err != nil {
		t.Fatalf("LoadConfig(zh): %v", err)
	}
	if Assets().Templates == nil {
		t.Error("Templates fs.FS is nil after LoadConfig")
	}
	if len(Assets().TemplatesTOML) == 0 {
		t.Error("TemplatesTOML is empty after LoadConfig")
	}
}
