package script

import "testing"

func depKeys(code string) map[string]string {
	out := map[string]string{}
	for _, d := range AssetDeps(code) {
		out[d.Key] = d.Kind
	}
	return out
}

func TestAssetDeps_ConstAndInline(t *testing.T) {
	// 关键场景:GUID 放 const, 调用处引用变量 — pin-keyed 抽会漏, 全文扫覆盖。
	code := `
const RESULT = "3680b3d2-d31d-461c-b697-0d9c3e6a87ed";
const SOUND  = "clip-2ba73f97-2820-4090-958a-c07dd3f8f48c";
while (true) {
  if (CheckTemplate({Templates:[RESULT], Threshold:0.75}).exit === "Found") break;
  ClickTemplate({Templates:["b518a466-e3d4-4b9e-9bb1-895ea5b80b1d"]});
  PlayClip({ClipID: SOUND});
}
`
	got := depKeys(code)
	want := map[string]string{
		"3680b3d2-d31d-461c-b697-0d9c3e6a87ed":      "template", // const-declared
		"b518a466-e3d4-4b9e-9bb1-895ea5b80b1d":      "template", // inline literal
		"clip-2ba73f97-2820-4090-958a-c07dd3f8f48c": "clip",     // clip- prefix
	}
	if len(got) != len(want) {
		t.Fatalf("dep count = %d, want %d: %v", len(got), len(want), got)
	}
	for k, kind := range want {
		if got[k] != kind {
			t.Errorf("dep %q kind = %q, want %q", k, got[k], kind)
		}
	}
}

func TestAssetDeps_ClipPrefixNotDoubleCounted(t *testing.T) {
	// clip-<uuid> 不能既算 clip 又把内层 uuid 当 template。
	deps := AssetDeps(`PlayClip({ClipID:"clip-2ba73f97-2820-4090-958a-c07dd3f8f48c"})`)
	if len(deps) != 1 {
		t.Fatalf("want 1 dep, got %d: %v", len(deps), deps)
	}
	if deps[0].Kind != "clip" || deps[0].Key != "clip-2ba73f97-2820-4090-958a-c07dd3f8f48c" {
		t.Errorf("got %+v", deps[0])
	}
}

func TestAssetDeps_Dedup(t *testing.T) {
	code := `
const A = "3680b3d2-d31d-461c-b697-0d9c3e6a87ed";
CheckTemplate({Templates:[A]});
CheckTemplate({Templates:["3680b3d2-d31d-461c-b697-0d9c3e6a87ed"]});
`
	if n := len(AssetDeps(code)); n != 1 {
		t.Fatalf("want 1 deduped dep, got %d", n)
	}
}

func TestAssetDeps_EmptyAndNoGUID(t *testing.T) {
	for _, code := range []string{"", "log.info($hp); return 1;", "let x = 'not-a-guid-1234';"} {
		if d := AssetDeps(code); len(d) != 0 {
			t.Errorf("code %q: want 0 deps, got %v", code, d)
		}
	}
}

func TestAssetDeps_MultipleInOrder(t *testing.T) {
	code := `"3680b3d2-d31d-461c-b697-0d9c3e6a87ed" "b518a466-e3d4-4b9e-9bb1-895ea5b80b1d"`
	deps := AssetDeps(code)
	if len(deps) != 2 {
		t.Fatalf("want 2, got %d: %v", len(deps), deps)
	}
	if deps[0].Key != "3680b3d2-d31d-461c-b697-0d9c3e6a87ed" || deps[1].Key != "b518a466-e3d4-4b9e-9bb1-895ea5b80b1d" {
		t.Errorf("order not preserved: %+v", deps)
	}
}
