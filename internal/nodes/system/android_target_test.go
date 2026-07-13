package system

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/node"
)

func TestAndroidTarget_SpecHasExecInAndTargetPins(t *testing.T) {
	s := AndroidTarget{}.Spec()
	if s.Kind != "AndroidTarget" {
		t.Fatalf("kind = %q, want AndroidTarget", s.Kind)
	}
	in := map[string]string{}
	for _, p := range s.Inputs {
		in[p.Name] = p.Type
		if p.Name == atInSerial {
			if p.Widget.Kind != "async-dropdown" {
				t.Fatalf("Serial widget = %q, want async-dropdown", p.Widget.Kind)
			}
			applyMeta, ok := p.Widget.Props["applyMeta"].(map[string]any)
			if !ok {
				t.Fatalf("Serial applyMeta = %#v", p.Widget.Props["applyMeta"])
			}
			if applyMeta["width"] != atInWidth || applyMeta["height"] != atInHeight || applyMeta["name"] != atInName {
				t.Fatalf("Serial applyMeta = %#v", applyMeta)
			}
		}
	}
	for name, typ := range map[string]string{
		atInExec:   node.TypeExec,
		atInSerial: "String",
		atInName:   "String",
		atInWidth:  "Number",
		atInHeight: "Number",
	} {
		if in[name] != typ {
			t.Fatalf("input %q type = %q, want %q", name, in[name], typ)
		}
	}
}

func TestAndroidTarget_RunSetsActiveTarget(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&AndroidTarget{})
	rn, _ := registry.Get("AndroidTarget")

	svc := node.StubServices()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			atInSerial: "emulator-5554",
			atInName:   "Pixel 8",
			atInWidth:  1080,
			atInHeight: 2400,
		},
		nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != atOutDone {
		t.Fatalf("exit = %q, want %q", r.ExitName, atOutDone)
	}
	if r.OutputData["TargetID"] != "android:emulator-5554" {
		t.Fatalf("TargetID = %#v", r.OutputData["TargetID"])
	}
	tg, ok := svc.Target.Active()
	if !ok {
		t.Fatal("active target missing")
	}
	if tg.Kind != target.KindAndroidADB || tg.Ref.ADBSerial != "emulator-5554" {
		t.Fatalf("active target = %#v", tg)
	}
	if tg.DisplayName != "Pixel 8" || tg.Resolution.W != 1080 || tg.Resolution.H != 2400 {
		t.Fatalf("active target metadata = %#v", tg)
	}
}

func TestAndroidTarget_DeclarativeConstraintsBlockRun(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&AndroidTarget{})
	rn, _ := registry.Get("AndroidTarget")
	if rn.Validate != nil {
		t.Fatal("AndroidTarget retained executable validation callback")
	}
	services := node.StubServices()
	result := node.RunNode(context.Background(), rn, nil, map[string]any{
		atInSerial: " \t ",
		atInWidth:  0,
		atInHeight: -1,
	}, nil, services, false)
	if len(result.Validation) != 3 {
		t.Fatalf("validation = %#v, want 3 constraints", result.Validation)
	}
	if _, active := services.Target.Active(); active {
		t.Fatal("invalid target changed active runtime target")
	}
}
