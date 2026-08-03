package macro

import (
	"bytes"
	"testing"
)

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
