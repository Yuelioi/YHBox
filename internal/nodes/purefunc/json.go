package purefunc

import (
	"encoding/json"
	stdio "io"
	"strconv"
	"strings"

	"yotta/internal/node"
)

type ParseJSON struct{}

func (ParseJSON) Spec() node.Spec {
	return specBuilder("ParseJSON", []node.InputSpec{
		{Name: "Text", Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
	}, "JSON")
}

func (ParseJSON) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	dec := json.NewDecoder(strings.NewReader(in.String("Text")))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		ctx.Log().Warn("ParseJSON: JSON 解析失败: %v", err)
		return nil, nil
	}
	var extra any
	if err := dec.Decode(&extra); err != stdio.EOF {
		ctx.Log().Warn("ParseJSON: JSON 解析失败: 尾部有多余内容")
		return nil, nil
	}
	return value, nil
}

type ToJSON struct{}

func (ToJSON) Spec() node.Spec {
	return specBuilder("ToJSON", []node.InputSpec{
		{Name: "Value", Type: "*"},
	}, "String")
}

func (ToJSON) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	data, err := json.Marshal(in.Raw("Value"))
	if err != nil {
		ctx.Log().Warn("ToJSON: JSON 序列化失败: %v", err)
		return "", nil
	}
	return string(data), nil
}

type JsonPath struct{}

func (JsonPath) Spec() node.Spec {
	return specBuilder("JsonPath", []node.InputSpec{
		{Name: "JSON", Type: "JSON"},
		{Name: "Path", Type: "String", Default: "$", Widget: node.WidgetSpec{Kind: "text"}},
	}, "*")
}

func (JsonPath) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	steps, ok := parseJSONPath(in.String("Path"))
	if !ok {
		return nil, nil
	}
	return evalJSONPath(in.JSONValue("JSON"), steps), nil
}

type jsonPathStep struct {
	kind  string
	field string
	index int
}

func parseJSONPath(path string) ([]jsonPathStep, bool) {
	if path == "$" {
		return nil, true
	}
	if !strings.HasPrefix(path, "$") {
		return nil, false
	}
	var steps []jsonPathStep
	for i := 1; i < len(path); {
		switch path[i] {
		case '.':
			i++
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			if start == i {
				return nil, false
			}
			steps = append(steps, jsonPathStep{kind: "field", field: path[start:i]})
		case '[':
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				return nil, false
			}
			raw := path[i+1 : i+end]
			if raw == "*" {
				steps = append(steps, jsonPathStep{kind: "wildcard"})
			} else {
				index, err := strconv.Atoi(raw)
				if err != nil || index < 0 {
					return nil, false
				}
				steps = append(steps, jsonPathStep{kind: "index", index: index})
			}
			i += end + 1
		default:
			return nil, false
		}
	}
	return steps, true
}

func evalJSONPath(value any, steps []jsonPathStep) any {
	if len(steps) == 0 {
		return value
	}
	step := steps[0]
	rest := steps[1:]
	switch step.kind {
	case "field":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		next, ok := obj[step.field]
		if !ok {
			return nil
		}
		return evalJSONPath(next, rest)
	case "index":
		arr, ok := value.([]any)
		if !ok || step.index >= len(arr) {
			return nil
		}
		return evalJSONPath(arr[step.index], rest)
	case "wildcard":
		arr, ok := value.([]any)
		if !ok {
			return nil
		}
		out := make([]any, len(arr))
		for i, item := range arr {
			out[i] = evalJSONPath(item, rest)
		}
		return out
	default:
		return nil
	}
}
