package runtime

import (
	"testing"
)

// resolveVK 是 KeyHoldStart/Stop 共用 helper: 接受 string keyname (passthrough) 或
// numeric vk code (转 string 给 Backend). Backend.KeyDown/KeyUp 都吃 vk string,
// 所以输出永远是 string.

func TestResolveVKStringPassthrough(t *testing.T) {
	got, err := resolveVK("A")
	if err != nil {
		t.Fatalf("resolveVK(\"A\"): %v", err)
	}
	if got != "A" {
		t.Fatalf("expected \"A\", got %q", got)
	}

	got, err = resolveVK("space")
	if err != nil {
		t.Fatalf("resolveVK(\"space\"): %v", err)
	}
	if got != "space" {
		t.Fatalf("expected \"space\", got %q", got)
	}

	got, err = resolveVK("F9")
	if err != nil {
		t.Fatalf("resolveVK(\"F9\"): %v", err)
	}
	if got != "F9" {
		t.Fatalf("expected \"F9\", got %q", got)
	}
}

func TestResolveVKNumericLetter(t *testing.T) {
	// 65 == 'A' == VK_A
	got, err := resolveVK(float64(65))
	if err != nil {
		t.Fatalf("resolveVK(65): %v", err)
	}
	if got != "A" {
		t.Fatalf("expected \"A\" for 65, got %q", got)
	}
}

func TestResolveVKNumericDigit(t *testing.T) {
	// 0x31 == '1'
	got, err := resolveVK(float64(0x31))
	if err != nil {
		t.Fatalf("resolveVK(0x31): %v", err)
	}
	if got != "1" {
		t.Fatalf("expected \"1\" for 0x31, got %q", got)
	}
}

func TestResolveVKNumericFunctionKey(t *testing.T) {
	// 0x78 == VK_F9
	got, err := resolveVK(float64(0x78))
	if err != nil {
		t.Fatalf("resolveVK(0x78): %v", err)
	}
	if got != "F9" {
		t.Fatalf("expected \"F9\" for 0x78, got %q", got)
	}
}

func TestResolveVKNumericSpace(t *testing.T) {
	// 0x20 == VK_SPACE
	got, err := resolveVK(float64(0x20))
	if err != nil {
		t.Fatalf("resolveVK(0x20): %v", err)
	}
	if got != "space" {
		t.Fatalf("expected \"space\" for 0x20, got %q", got)
	}
}

func TestResolveVKIntType(t *testing.T) {
	// 也支持 int (Go 字面量场景, 虽然 JSON 反序列化通常给 float64)
	got, err := resolveVK(65)
	if err != nil {
		t.Fatalf("resolveVK(int 65): %v", err)
	}
	if got != "A" {
		t.Fatalf("expected \"A\" for int 65, got %q", got)
	}
}

func TestResolveVKEmptyString(t *testing.T) {
	_, err := resolveVK("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestResolveVKNil(t *testing.T) {
	_, err := resolveVK(nil)
	if err == nil {
		t.Fatal("expected error for nil")
	}
}

func TestResolveVKUnknownString(t *testing.T) {
	_, err := resolveVK("not-a-real-key")
	if err == nil {
		t.Fatal("expected error for unknown keyname")
	}
}

func TestResolveVKOutOfRange(t *testing.T) {
	_, err := resolveVK(float64(0))
	if err == nil {
		t.Fatal("expected error for 0")
	}
	_, err = resolveVK(float64(300))
	if err == nil {
		t.Fatal("expected error for 300")
	}
}

func TestResolveVKUnsupportedType(t *testing.T) {
	_, err := resolveVK([]int{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}
