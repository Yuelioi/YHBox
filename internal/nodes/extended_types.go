package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/datatype"
)

const (
	BreakPointNodeID  = "https://schemas.yotta.dev/nodes/structure/break-point"
	BreakRegionNodeID = "https://schemas.yotta.dev/nodes/structure/break-region"
)

func sealExtendedTypes(numberRef datatype.TypeRef) (datatype.Definition, datatype.Definition, datatype.Definition, datatype.Definition, error) {
	jsonType, err := sealStructuredType(
		JSONTypeID,
		json.RawMessage(fmt.Sprintf(`{"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema"}`, JSONTypeID+"/schema")),
		datatype.Authoring{TitleKey: "type.core.json.title", DescriptionKey: "type.core.json.description", Color: "#64748b", Icon: "braces"},
	)
	if err != nil {
		return datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, err
	}
	pointUnit, err := sealStructuredType(
		PointUnitTypeID,
		json.RawMessage(fmt.Sprintf(`{"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string","enum":["ratio","px"]}`, PointUnitTypeID+"/schema")),
		datatype.Authoring{TitleKey: "type.geometry.point_unit.title", DescriptionKey: "type.geometry.point_unit.description", Color: "#14b8a6", Icon: "ruler"},
	)
	if err != nil {
		return datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, err
	}
	point, err := sealStructuredTypeWithStructure(
		PointTypeID,
		json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"x":{"type":"number"},"y":{"type":"number"},"unit":{"type":"string","enum":["ratio","px"]}},
			"required":["x","y","unit"],"additionalProperties":false
		}`, PointTypeID+"/schema")),
		datatype.Authoring{
			TitleKey: "type.geometry.point.title", DescriptionKey: "type.geometry.point.description", Color: "#22c55e", Icon: "map-pin", EditorAdapter: "point",
			Importance: "primary", InlinePriority: 100,
			Examples:      []json.RawMessage{json.RawMessage(`{"x":0.5,"y":0.5,"unit":"ratio"}`), json.RawMessage(`{"x":640,"y":360,"unit":"px"}`)},
			BreakTitleKey: "node.structure.breakPoint.title", BreakDescriptionKey: "node.structure.breakPoint.description",
		},
		&datatype.StructureSpec{BreakNodeTypeID: BreakPointNodeID, Fields: []datatype.StructureField{
			{ID: "unit", Type: datatype.RefExpression(pointUnit.TypeRef())},
			{ID: "x", Type: datatype.RefExpression(numberRef)},
			{ID: "y", Type: datatype.RefExpression(numberRef)},
		}},
	)
	if err != nil {
		return datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, err
	}
	region, err := sealStructuredTypeWithStructure(
		RegionTypeID,
		json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",
			"properties":{"x":{"type":"number"},"y":{"type":"number"},"width":{"type":"number","minimum":0},"height":{"type":"number","minimum":0},"unit":{"type":"string","enum":["ratio","px"]}},
			"required":["x","y","width","height","unit"],"additionalProperties":false
		}`, RegionTypeID+"/schema")),
		datatype.Authoring{
			TitleKey: "type.geometry.region.title", DescriptionKey: "type.geometry.region.description", Color: "#84cc16", Icon: "crop", EditorAdapter: "region",
			Importance: "primary", InlinePriority: 80,
			Examples:      []json.RawMessage{json.RawMessage(`{"x":0.25,"y":0.25,"width":0.5,"height":0.5,"unit":"ratio"}`)},
			BreakTitleKey: "node.structure.breakRegion.title", BreakDescriptionKey: "node.structure.breakRegion.description",
		},
		&datatype.StructureSpec{BreakNodeTypeID: BreakRegionNodeID, Fields: []datatype.StructureField{
			{ID: "height", Type: datatype.RefExpression(numberRef)},
			{ID: "unit", Type: datatype.RefExpression(pointUnit.TypeRef())},
			{ID: "width", Type: datatype.RefExpression(numberRef)},
			{ID: "x", Type: datatype.RefExpression(numberRef)},
			{ID: "y", Type: datatype.RefExpression(numberRef)},
		}},
	)
	if err != nil {
		return datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, datatype.Definition{}, err
	}
	return jsonType, pointUnit, point, region, nil
}

func sealStructuredType(typeID string, schema json.RawMessage, authoring datatype.Authoring) (datatype.Definition, error) {
	return sealStructuredTypeWithStructure(typeID, schema, authoring, nil)
}

func sealStructuredTypeWithStructure(typeID string, schema json.RawMessage, authoring datatype.Authoring, structure *datatype.StructureSpec) (datatype.Definition, error) {
	return datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: typeID, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: typeID + "/schema",
		SchemaBundle:    []datatype.SchemaResource{{ID: typeID + "/schema", Schema: schema}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}},
		Traits: []datatype.Trait{
			datatype.TraitDurable,
			datatype.TraitEquatable,
			datatype.TraitObservable,
		},
		Authoring: authoring,
		Structure: structure,
	})
}
