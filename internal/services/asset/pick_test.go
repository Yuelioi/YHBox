// internal/services/asset/pick_test.go
package asset

import "testing"

func TestStore_PickVariant(t *testing.T) {
	s, _ := newTestStore(t)

	// 建 template 记录，含 1920x1080 和 1280x720 两档。
	rec := makeRecord("g", "Pick Test", KindTemplate)
	if err := s.PutRecord(rec); err != nil {
		t.Fatalf("PutRecord: %v", err)
	}

	b1080 := testBlobRef("sha-1080")
	b720 := testBlobRef("sha-720")
	observeTestBlob(t, s, b1080)
	observeTestBlob(t, s, b720)
	if err := s.putVariant("g", [2]int{1920, 1080}, b1080, [4]int{0, 0, 1920, 1080}, nil); err != nil {
		t.Fatalf("PutVariant 1080: %v", err)
	}
	if err := s.putVariant("g", [2]int{1280, 720}, b720, [4]int{0, 0, 1280, 720}, nil); err != nil {
		t.Fatalf("PutVariant 720: %v", err)
	}

	// Case 1: 精确命中 1920x1080 → 返回 b1080。
	got, ok := s.PickVariant("g", 1920, 1080)
	if !ok {
		t.Fatal("case1: exact hit should return ok=true")
	}
	if got.Blob != b1080 {
		t.Errorf("case1: got blob %#v, want %#v", got.Blob, b1080)
	}

	// Case 2: 2560x1440 没有精确档 → 长边比 fallback（2560 vs 1920 长边比=1.33, vs 1280 长边比=2.0）
	// 应命中 1920x1080（更近）, ok=true。
	got2, ok2 := s.PickVariant("g", 2560, 1440)
	if !ok2 {
		t.Fatal("case2: fallback should return ok=true")
	}
	if got2.Resolution != [2]int{1920, 1080} {
		t.Errorf("case2: expected 1920x1080 fallback, got %v", got2.Resolution)
	}

	// Case 3: guid 不存在 → ok=false。
	_, ok3 := s.PickVariant("missing", 1920, 1080)
	if ok3 {
		t.Error("case3: missing guid should return ok=false")
	}

	// Case 4: 记录存在但 0 个变体 → ok=false。
	empty := makeRecord("empty", "Empty", KindTemplate)
	s.PutRecord(empty)
	_, ok4 := s.PickVariant("empty", 1920, 1080)
	if ok4 {
		t.Error("case4: record with 0 variants should return ok=false")
	}
}
