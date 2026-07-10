package ai

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/llm"
)

type outputDecl struct{ Name, Type string }

// parseOutputDecls 读 config.Outputs[](merged 进 inputs 的 "Outputs" key)→ {Name,Type} 列表。
func parseOutputDecls(in node.Inputs) []outputDecl {
	raw, _ := in.Raw("Outputs").([]any)
	var decls []outputDecl
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["Name"].(string)
		if name == "" {
			continue
		}
		typ, _ := m["Type"].(string)
		decls = append(decls, outputDecl{Name: name, Type: typ})
	}
	return decls
}

func buildSchema(decls []outputDecl) llm.JSONSchema {
	fields := make([]llm.SchemaField, 0, len(decls))
	for _, d := range decls {
		fields = append(fields, llm.SchemaField{Name: d.Name, JSONType: jsonTypeFor(d.Type)})
	}
	return llm.JSONSchema{Fields: fields}
}

// jsonTypeFor 节点类型 → JSON schema 类型(单一映射表, §5)。
func jsonTypeFor(t string) string {
	switch t {
	case "Number":
		return "number"
	case "Integer":
		return "integer"
	case "Bool":
		return "boolean"
	case "JSON":
		return "object"
	case "List":
		return "array"
	default: // String + 兜底
		return "string"
	}
}

// coerceOutput 把解析出的 JSON 值按声明类型转成节点值。Integer 非整/超界 → error。
func coerceOutput(v any, t string) (any, error) {
	switch t {
	case "Integer":
		f, ok := toFloat(v)
		if !ok {
			return nil, fmt.Errorf("期望 integer, 实得 %T", v)
		}
		if f != math.Trunc(f) || f > math.MaxInt64 || f < math.MinInt64 {
			return nil, fmt.Errorf("值 %v 不是整数", v)
		}
		return int(f), nil
	case "Number":
		f, ok := toFloat(v)
		if !ok {
			return nil, fmt.Errorf("期望 number, 实得 %T", v)
		}
		return f, nil
	case "Bool":
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("期望 bool, 实得 %T", v)
		}
		return b, nil
	case "String":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("期望 string, 实得 %T", v)
		}
		return s, nil
	default: // JSON / List — 原样透传
		return v, nil
	}
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	}
	return 0, false
}
