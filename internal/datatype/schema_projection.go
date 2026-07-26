package datatype

// TuneTypeExpressionDefinitions replaces reflection's structurally loose view
// with the same discriminated Type Expression union used by every generated
// current contract surface.
func TuneTypeExpressionDefinitions(definitions map[string]any) {
	// Some reflection roots only encounter TypeRef through TypeExpression's
	// pointer fields. Once TypeExpression is replaced by the tagged union the
	// reflector's loose, inlined definition may disappear, so install the
	// shared reference shape explicitly as well.
	definitions["TypeRef"] = map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"typeId": map[string]any{"type": "string", "format": "uri"},
			"semanticDigest": map[string]any{
				"type": "string", "pattern": "^sha256:[a-f0-9]{64}$",
			},
		},
		"required": []string{"typeId", "semanticDigest"},
	}
	definitions["TypeExpression"] = map[string]any{
		"title": "TypeExpression",
		"oneOf": []any{
			typeExpressionVariant("ref", map[string]any{
				"ref": map[string]any{"$ref": "#/$defs/TypeRef"},
			}, "ref"),
			typeExpressionVariant("list", map[string]any{
				"element": map[string]any{"$ref": "#/$defs/TypeExpression"},
			}, "element"),
			typeExpressionVariant("union", map[string]any{
				"members": map[string]any{
					"type": "array", "minItems": 2, "maxItems": MaxUnionMembers,
					"items": map[string]any{"$ref": "#/$defs/TypeExpression"},
				},
			}, "members"),
			typeExpressionVariant("variable", map[string]any{
				"variable": map[string]any{"type": "string", "minLength": 1, "maxLength": MaxTypeStringBytes},
				"constraints": map[string]any{
					"type": "array", "maxItems": MaxTypeConstraints,
					"items": map[string]any{"type": "string", "minLength": 1, "maxLength": MaxTypeStringBytes},
				},
			}, "variable"),
		},
	}
}

func typeExpressionVariant(kind string, properties map[string]any, required ...string) map[string]any {
	properties["kind"] = map[string]any{"const": kind, "type": "string"}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             append([]string{"kind"}, required...),
	}
}
