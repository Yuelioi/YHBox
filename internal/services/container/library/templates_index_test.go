package library

import "testing"

func TestLoadTemplatesIndex_18Entries(t *testing.T) {
	idx, err := LoadTemplatesIndex()
	if err != nil {
		t.Fatalf("load err: %v", err)
	}
	if len(idx.Templates) != 18 {
		t.Fatalf("expected 18 templates, got %d", len(idx.Templates))
	}
	expectedSlots := []string{
		"bait_product", "buy_button", "buy_confirm", "buy_max", "buy_success",
		"change_bait_confirm", "fish_escape", "go_fishing", "hook_icon", "hook_text",
		"need_bait", "no_currency", "result", "shop_bag_tab", "shop_confirm_sell",
		"shop_sell_all", "start_fish", "warehouse_full",
	}
	for _, slot := range expectedSlots {
		entry, ok := idx.Templates[slot]
		if !ok {
			t.Errorf("missing slot %q", slot)
			continue
		}
		if entry.File == "" {
			t.Errorf("slot %q: empty file", slot)
		}
		if entry.Resolution != [2]int{1920, 1080} {
			t.Errorf("slot %q: expected 1080p, got %v", slot, entry.Resolution)
		}
		if entry.Bbox[0] >= entry.Bbox[2] || entry.Bbox[1] >= entry.Bbox[3] {
			t.Errorf("slot %q: invalid bbox %v", slot, entry.Bbox)
		}
	}
}
