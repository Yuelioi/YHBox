package tools

import (
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestToolServiceStableEarlyFailures(t *testing.T) {
	service := NewService(nil, nil)
	tests := []struct {
		call func() error
		id   string
	}{
		{func() error { _, err := service.PixelAt("target"); return err }, "tools.target_unavailable"},
		{func() error { return service.OpenScreenPicker("invalid", "request", "target", "", "") }, "tools.picker.invalid"},
		{func() error { return service.OpenScreenPicker("point", "", "target", "", "") }, "tools.picker.invalid"},
		{func() error { return service.OpenScreenPicker("point", "request", "target", "", "") }, "tools.target_unavailable"},
		{func() error { _, err := service.ExtractColorRange(nil, "rgb"); return err }, "tools.color_range.invalid"},
	}
	for _, test := range tests {
		if got := apperr.From(test.call()); got.ID != test.id {
			t.Fatalf("problem = %#v, want %s", got, test.id)
		}
	}
	if err := service.ClosePicker("missing"); err != nil {
		t.Fatal(err)
	}
	if err := service.SetLauncherAlwaysOnTop(true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetLauncherSize(100, 100); err != nil {
		t.Fatal(err)
	}
	withTarget := NewService(fakeTargetResolver{}, nil)
	if _, err := withTarget.MousePos("target"); err != nil {
		t.Fatal(err)
	}
	if _, err := withTarget.MousePos("target"); err != nil {
		t.Fatal(err)
	}
}
