package node

import "testing"

func TestSchemaHelpers(t *testing.T) {
	s := ObjSchema(
		Field("HMin", NumberSchema(), true),
		Field("Mode", EnumSchema(EnumOption{Value: "hsv"}, EnumOption{Value: "rgb"}), false),
	)
	if s.Type != "object" || len(s.Fields) != 2 || !s.Fields[0].Required {
		t.Fatalf("ObjSchema 构造错: %+v", s)
	}
	g := GeometrySchema()
	if g.Widget != "geometry" || g.Type != "object" {
		t.Fatalf("GeometrySchema 应 Type=object Widget=geometry, got %+v", g)
	}
}
