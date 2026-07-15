package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"

	runtimejsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxStructuredSchemaBytes = 64 << 10
	MaxStructuredDepth       = 32
	MaxStructuredNodes       = 8192
	structuredSchemaResource = "https://schemas.yotta.dev/runtime/ai-output/v1"
)

var structuredNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

type StructuredOutputSpec struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
}

func CompileStructuredOutput(name string, raw json.RawMessage) (StructuredOutputSpec, error) {
	if !structuredNamePattern.MatchString(name) || len(raw) == 0 || len(raw) > MaxStructuredSchemaBytes {
		return StructuredOutputSpec{}, errors.New("invalid AI structured output identity or schema budget")
	}
	if err := artifact.InspectJSONBudget(raw, MaxStructuredDepth, MaxStructuredNodes, MaxStructuredSchemaBytes); err != nil {
		return StructuredOutputSpec{}, fmt.Errorf("AI structured output schema exceeds structural budget: %w", err)
	}
	var schema any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&schema); err != nil {
		return StructuredOutputSpec{}, fmt.Errorf("decode AI structured output schema: %w", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		return StructuredOutputSpec{}, errors.New("AI structured output schema root must be an object")
	}
	if err := validateStrictSchemaNode(root, true, 0); err != nil {
		return StructuredOutputSpec{}, err
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return StructuredOutputSpec{}, err
	}
	spec := StructuredOutputSpec{Name: name, Schema: canonical}
	if _, err := spec.validator(); err != nil {
		return StructuredOutputSpec{}, fmt.Errorf("compile AI structured output schema: %w", err)
	}
	return spec, nil
}

func (s StructuredOutputSpec) Validate() error {
	compiled, err := CompileStructuredOutput(s.Name, s.Schema)
	if err != nil {
		return err
	}
	if !bytes.Equal(compiled.Schema, s.Schema) {
		return errors.New("AI structured output schema must be canonical")
	}
	return nil
}

func (s StructuredOutputSpec) ValidateValue(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || len(raw) > MaxPromptBytes {
		return nil, errors.New("AI structured output value exceeds byte budget")
	}
	validator, err := s.validator()
	if err != nil {
		return nil, err
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode AI structured output value: %w", err)
	}
	if err := validator.Validate(value); err != nil {
		return nil, fmt.Errorf("AI structured output violates schema: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func (s StructuredOutputSpec) validator() (*runtimejsonschema.Schema, error) {
	if !structuredNamePattern.MatchString(s.Name) || len(s.Schema) == 0 || len(s.Schema) > MaxStructuredSchemaBytes {
		return nil, errors.New("invalid AI structured output specification")
	}
	var schema any
	decoder := json.NewDecoder(bytes.NewReader(s.Schema))
	decoder.UseNumber()
	if err := decoder.Decode(&schema); err != nil {
		return nil, err
	}
	compiler := runtimejsonschema.NewCompiler()
	if err := compiler.AddResource(structuredSchemaResource, schema); err != nil {
		return nil, err
	}
	return compiler.Compile(structuredSchemaResource)
}

func validateStrictSchemaNode(node map[string]any, root bool, depth int) error {
	if depth > MaxStructuredDepth {
		return errors.New("AI structured output schema exceeds depth budget")
	}
	allowed := map[string]bool{
		"type": true, "description": true, "enum": true, "properties": true, "required": true,
		"additionalProperties": true, "items": true, "anyOf": true,
	}
	for key := range node {
		if !allowed[key] {
			return fmt.Errorf("AI structured output schema keyword %q is not in the strict portable profile", key)
		}
	}
	if alternatives, ok := node["anyOf"]; ok {
		if root {
			return errors.New("AI structured output root must be an object")
		}
		if len(node) != 1 && !(len(node) == 2 && node["description"] != nil) {
			return errors.New("AI structured output anyOf cannot be combined with other validation keywords")
		}
		items, ok := alternatives.([]any)
		if !ok || len(items) < 2 || len(items) > 16 {
			return errors.New("AI structured output anyOf must contain 2 to 16 alternatives")
		}
		for _, item := range items {
			child, ok := item.(map[string]any)
			if !ok {
				return errors.New("AI structured output anyOf alternatives must be schemas")
			}
			if err := validateStrictSchemaNode(child, false, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	typeName, ok := node["type"].(string)
	if !ok {
		return errors.New("AI structured output schema node requires one explicit type")
	}
	switch typeName {
	case "object":
		properties, ok := node["properties"].(map[string]any)
		if !ok || len(properties) > 1024 || node["additionalProperties"] != false {
			return errors.New("AI structured object requires properties and additionalProperties:false")
		}
		requiredRaw, ok := node["required"].([]any)
		if !ok || len(requiredRaw) != len(properties) {
			return errors.New("AI structured object must require every property")
		}
		required := make([]string, 0, len(requiredRaw))
		for _, value := range requiredRaw {
			name, ok := value.(string)
			if !ok || name == "" {
				return errors.New("AI structured object has an invalid required property")
			}
			required = append(required, name)
		}
		sort.Strings(required)
		propertyNames := make([]string, 0, len(properties))
		for name, value := range properties {
			child, ok := value.(map[string]any)
			if !ok || name == "" {
				return errors.New("AI structured object property must contain a schema")
			}
			if err := validateStrictSchemaNode(child, false, depth+1); err != nil {
				return fmt.Errorf("AI structured property %q: %w", name, err)
			}
			propertyNames = append(propertyNames, name)
		}
		sort.Strings(propertyNames)
		for index := range required {
			if required[index] != propertyNames[index] {
				return errors.New("AI structured object required list must exactly match properties")
			}
		}
	case "array":
		child, ok := node["items"].(map[string]any)
		if !ok {
			return errors.New("AI structured array requires one item schema")
		}
		if err := validateStrictSchemaNode(child, false, depth+1); err != nil {
			return err
		}
	case "string", "number", "integer", "boolean", "null":
	default:
		return fmt.Errorf("AI structured output type %q is not supported", typeName)
	}
	if root && typeName != "object" {
		return errors.New("AI structured output root must be an object")
	}
	if enum, ok := node["enum"]; ok {
		values, ok := enum.([]any)
		if !ok || len(values) == 0 || len(values) > 1024 {
			return errors.New("AI structured output enum is invalid")
		}
	}
	return nil
}
