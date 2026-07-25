package resourceauthoring

import (
	"context"
	"testing"
)

func TestServicePromoteEmitsGlobalAssetRevision(t *testing.T) {
	creator, _, _ := newTestCreator(t)
	resource, err := creator.CreateImage(context.Background(), ImageDraft{
		Metadata: Metadata{Name: "Promoted"}, DataURL: pngDataURL(t, 8, 8),
		Resolution: [2]int{8, 8}, Region: [4]float32{0, 0, 1, 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	var eventName string
	var eventData any
	service := NewService(creator, func(name string, data any) {
		eventName, eventData = name, data
	})
	promotion, err := service.Promote(resource)
	if err != nil {
		t.Fatal(err)
	}
	payload, ok := eventData.(map[string]any)
	if !ok || eventName != "asset:changed" || payload["revision"] != promotion.Revision {
		t.Fatalf("event = %q %#v, promotion = %+v", eventName, eventData, promotion)
	}
	guids, ok := payload["guids"].([]string)
	if !ok || len(guids) != 1 || guids[0] != promotion.GUID {
		t.Fatalf("event GUIDs = %#v", payload["guids"])
	}
}
