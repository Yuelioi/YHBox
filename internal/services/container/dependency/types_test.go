package dependency

import "testing"

func TestDependency_String(t *testing.T) {
	d := Dependency{Kind: KindTemplate, Key: "fishing.hook_icon"}
	if got := d.String(); got != "template:fishing.hook_icon" {
		t.Errorf("got %q", got)
	}
}
