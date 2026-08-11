package macro

import (
	"bytes"
	"testing"
)

const legacyV1Carrier = `{"actions":[{"durationUs":20000,"id":"sleep","kind":"sleep"}],"baseResolution":[800,600],"schemaVersion":1}`

func TestCodecRoundTrip(t *testing.T) {
	document := Document{SchemaVersion: SchemaVersion, BaseResolution: [2]int{800, 600}, Meta: DefaultMeta(), Actions: []Action{
		{ID: "down", Kind: ActionMouseDown, Button: "left", Point: &Point{X: 0.25, Y: 0.75, Unit: "ratio"}},
		{ID: "sleep", Kind: ActionSleep, DurationUs: 20_000},
		{ID: "up", Kind: ActionMouseUp, Button: "left", Point: &Point{X: 0.25, Y: 0.75, Unit: "ratio"}},
	}}
	var encoded bytes.Buffer
	if err := Encode(&encoded, document); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Actions) != 3 || decoded.Actions[1].DurationUs != 20_000 || decoded.Meta.AutoMove.Mode != document.Meta.AutoMove.Mode {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestDecodeRejectsNonCanonicalAndUnknownFields(t *testing.T) {
	for _, source := range []string{
		`{ "schemaVersion":1,"baseResolution":[100,100],"actions":[]}`,
		`{"actions":[],"baseResolution":[100,100],"extra":true,"schemaVersion":1}`,
	} {
		if _, err := Decode(bytes.NewBufferString(source)); err == nil {
			t.Fatalf("Decode(%s) succeeded", source)
		}
	}
}

func TestDecodeMigratesCanonicalVersion1(t *testing.T) {
	document, err := Decode(bytes.NewBufferString(legacyV1Carrier))
	if err != nil {
		t.Fatal(err)
	}
	if document.SchemaVersion != SchemaVersion {
		t.Fatalf("schemaVersion = %d, want %d", document.SchemaVersion, SchemaVersion)
	}
	if document.Meta != legacyV1Meta() {
		t.Fatalf("meta = %#v, want %#v", document.Meta, legacyV1Meta())
	}
	if len(document.Actions) != 1 || document.Actions[0].Kind != ActionSleep || document.Actions[0].DurationUs != 20_000 {
		t.Fatalf("actions = %#v", document.Actions)
	}
}
