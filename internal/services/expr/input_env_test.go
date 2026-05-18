package expr

import "testing"

// TestInputEnv_BareIdent: v4 Expr node uses bare identifiers without $ prefix.
// Tests the parser + eval + InputEnv integration for plain identifiers.
func TestInputEnv_BareIdent(t *testing.T) {
	n, err := Parse("i + 1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	v, err := Eval(n, InputEnv{"i": float64(5)})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	got, _ := AsNumber(v)
	if got != 6.0 {
		t.Fatalf("i+1 with i=5: want 6, got %v", v)
	}
}

// TestInputEnv_BoolExpr: `s == "FISHING" && hp > 0.5`.
func TestInputEnv_BoolExpr(t *testing.T) {
	n, err := Parse(`s == "FISHING" && hp > 0.5`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	v, err := Eval(n, InputEnv{"s": "FISHING", "hp": float64(0.8)})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if v != true {
		t.Fatalf("want true, got %v", v)
	}

	// false branch
	v, _ = Eval(n, InputEnv{"s": "FISHING", "hp": float64(0.3)})
	if v != false {
		t.Fatalf("hp<0.5: want false, got %v", v)
	}
}

// TestInputEnv_UnknownIdentReturnsNil: missing input → nil (no error).
func TestInputEnv_UnknownIdentReturnsNil(t *testing.T) {
	n, _ := Parse("missing")
	v, err := Eval(n, InputEnv{})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if v != nil {
		t.Fatalf("missing input: want nil, got %v", v)
	}
}

// TestInputEnv_DoesNotExpose_DollarPaths: $vars in Expr context is independent of InputEnv.
// (The tokenizer creates a tkVarPath node — InputEnv just doesn't have $-keys; returns nil.)
func TestInputEnv_DoesNotExpose_DollarPaths(t *testing.T) {
	n, _ := Parse("$vars.x")
	v, _ := Eval(n, InputEnv{"x": float64(99)})
	// $vars.x looked up in InputEnv as "$vars.x" key → not present → nil
	if v != nil {
		t.Fatalf("$vars.x in Expr context: want nil (input isolated from sys vars), got %v", v)
	}
}

// v3 back-compat: bare ident with MapEnv still works through nIdent path.
func TestBareIdent_WithMapEnv(t *testing.T) {
	n, _ := Parse("foo")
	// MapEnv keyed by raw identifier (no $ prefix)
	v, _ := Eval(n, MapEnv{"foo": "bar"})
	if v != "bar" {
		t.Fatalf("want 'bar', got %v", v)
	}
}
