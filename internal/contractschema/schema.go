// Package contractschema validates bounded, offline JSON Schema 2020-12 bundles.
package contractschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxResources     = 256
	MaxResourceBytes = 256 << 10
	MaxBundleBytes   = 1 << 20
	MaxDepth         = 64
	MaxNodes         = 65_536
	MaxReferences    = 1_024
)

type Resource struct {
	ID     string          `json:"id"`
	Schema json.RawMessage `json:"schema"`
}

func Normalize(dialect, root string, source []Resource) ([]Resource, error) {
	if root != "" {
		if err := validateAbsoluteURI(root); err != nil {
			return nil, fmt.Errorf("invalid schema bundle root: %w", err)
		}
	}
	if len(source) == 0 || len(source) > MaxResources {
		return nil, errors.New("schema bundle exceeds resource budget")
	}
	seenRoots := make(map[string]bool, len(source))
	totalBytes := 0
	for _, resource := range source {
		if err := validateAbsoluteURI(resource.ID); err != nil {
			return nil, fmt.Errorf("invalid schema resource id: %w", err)
		}
		if seenRoots[resource.ID] {
			return nil, fmt.Errorf("duplicate schema resource %q", resource.ID)
		}
		seenRoots[resource.ID] = true
		if len(resource.Schema) == 0 || len(resource.Schema) > MaxResourceBytes || totalBytes > MaxBundleBytes-len(resource.Schema) {
			return nil, errors.New("schema bundle exceeds byte budget")
		}
		totalBytes += len(resource.Schema)
		if err := inspectJSON(resource.Schema); err != nil {
			return nil, fmt.Errorf("schema resource %q exceeds structural budget: %w", resource.ID, err)
		}
	}
	if root != "" && !seenRoots[root] {
		return nil, errors.New("schema bundle root is not in bundle")
	}

	result := make([]Resource, len(source))
	parsed := make([]any, len(source))
	for i, resource := range source {
		canonical, err := artifact.Canonicalize(resource.Schema)
		if err != nil {
			return nil, fmt.Errorf("canonicalize schema resource %q: %w", resource.ID, err)
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(canonical, &object); err != nil || object == nil {
			return nil, fmt.Errorf("schema resource %q must be a JSON object", resource.ID)
		}
		var schemaID, schemaDialect string
		if json.Unmarshal(object["$id"], &schemaID) != nil || schemaID != resource.ID {
			return nil, fmt.Errorf("schema resource %q has mismatched $id", resource.ID)
		}
		if json.Unmarshal(object["$schema"], &schemaDialect) != nil || schemaDialect != dialect {
			return nil, fmt.Errorf("schema resource %q has mismatched $schema", resource.ID)
		}
		decoded, err := decode(canonical)
		if err != nil {
			return nil, fmt.Errorf("decode schema resource %q: %w", resource.ID, err)
		}
		parsed[i] = decoded
		result[i] = Resource{ID: resource.ID, Schema: append(json.RawMessage(nil), canonical...)}
	}

	allResourceIDs := make(map[string]bool, len(seenRoots))
	for id := range seenRoots {
		allResourceIDs[id] = true
	}
	for i, resource := range source {
		base, _ := url.Parse(resource.ID)
		if err := collectResourceIDs(parsed[i], base, allResourceIDs, 0, true); err != nil {
			return nil, fmt.Errorf("schema resource %q: %w", resource.ID, err)
		}
	}
	referenceCount := 0
	for i, resource := range source {
		base, _ := url.Parse(resource.ID)
		if err := validateReferences(parsed[i], base, allResourceIDs, 0, &referenceCount); err != nil {
			return nil, fmt.Errorf("schema resource %q: %w", resource.ID, err)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func inspectJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	depth, nodes := 0, 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			if depth != 0 {
				return errors.New("unbalanced JSON containers")
			}
			return nil
		}
		if err != nil {
			return err
		}
		nodes++
		if nodes > MaxNodes {
			return errors.New("JSON node budget exceeded")
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{', '[':
				depth++
				if depth > MaxDepth {
					return errors.New("JSON depth budget exceeded")
				}
			case '}', ']':
				depth--
			}
		case string:
			if len(value) > MaxResourceBytes {
				return errors.New("JSON string budget exceeded")
			}
		}
	}
}

func decode(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func collectResourceIDs(value any, base *url.URL, bundled map[string]bool, depth int, root bool) error {
	if depth > MaxDepth {
		return errors.New("schema identifier walk exceeds depth budget")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	localBase, err := objectBase(object, base)
	if err != nil {
		return err
	}
	if localBase.String() != base.String() || !root {
		if _, hasID := object["$id"]; hasID {
			if bundled[localBase.String()] {
				return fmt.Errorf("duplicate bundled schema id %q", localBase.String())
			}
			bundled[localBase.String()] = true
		}
	}
	return forEachSubschema(object, func(subschema any) error {
		return collectResourceIDs(subschema, localBase, bundled, depth+1, false)
	})
}

func validateReferences(value any, base *url.URL, bundled map[string]bool, depth int, count *int) error {
	if depth > MaxDepth {
		return errors.New("schema reference walk exceeds depth budget")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	localBase, err := objectBase(object, base)
	if err != nil {
		return err
	}
	for _, key := range []string{"$ref", "$dynamicRef"} {
		item, exists := object[key]
		if !exists {
			continue
		}
		*count++
		if *count > MaxReferences {
			return errors.New("schema reference budget exceeded")
		}
		reference, ok := item.(string)
		if !ok {
			return fmt.Errorf("%s must be a string", key)
		}
		parsed, err := url.Parse(reference)
		if err != nil {
			return fmt.Errorf("invalid %s %q", key, reference)
		}
		resolved := localBase.ResolveReference(parsed)
		resolved.Fragment = ""
		if !bundled[resolved.String()] {
			return fmt.Errorf("%s %q is not in the offline schema bundle", key, reference)
		}
	}
	return forEachSubschema(object, func(subschema any) error {
		return validateReferences(subschema, localBase, bundled, depth+1, count)
	})
}

func objectBase(object map[string]any, base *url.URL) (*url.URL, error) {
	rawID, ok := object["$id"]
	if !ok {
		return base, nil
	}
	id, ok := rawID.(string)
	if !ok || id == "" {
		return nil, errors.New("$id must be a non-empty string")
	}
	parsed, err := url.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid $id %q", id)
	}
	resolved := base.ResolveReference(parsed)
	if resolved.Fragment != "" {
		return nil, fmt.Errorf("$id %q must not contain a fragment", id)
	}
	return resolved, nil
}

var singleSchemaKeywords = []string{
	"additionalProperties", "contains", "contentSchema", "else", "if", "items", "not",
	"propertyNames", "then", "unevaluatedItems", "unevaluatedProperties",
}

var schemaArrayKeywords = []string{"allOf", "anyOf", "oneOf", "prefixItems"}
var schemaMapKeywords = []string{"$defs", "dependentSchemas", "patternProperties", "properties"}

func forEachSubschema(object map[string]any, visit func(any) error) error {
	for _, keyword := range singleSchemaKeywords {
		value, ok := object[keyword]
		if !ok {
			continue
		}
		if !isSchema(value) {
			return fmt.Errorf("schema keyword %q must contain a schema", keyword)
		}
		if err := visit(value); err != nil {
			return err
		}
	}
	for _, keyword := range schemaArrayKeywords {
		value, ok := object[keyword]
		if !ok {
			continue
		}
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("schema keyword %q must contain a schema array", keyword)
		}
		for _, item := range items {
			if !isSchema(item) {
				return fmt.Errorf("schema keyword %q must contain only schemas", keyword)
			}
			if err := visit(item); err != nil {
				return err
			}
		}
	}
	for _, keyword := range schemaMapKeywords {
		value, ok := object[keyword]
		if !ok {
			continue
		}
		entries, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("schema keyword %q must contain a schema map", keyword)
		}
		keys := make([]string, 0, len(entries))
		for key := range entries {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !isSchema(entries[key]) {
				return fmt.Errorf("schema keyword %q must contain only schemas", keyword)
			}
			if err := visit(entries[key]); err != nil {
				return err
			}
		}
	}
	return nil
}

func isSchema(value any) bool {
	switch value.(type) {
	case bool, map[string]any:
		return true
	default:
		return false
	}
}

func validateAbsoluteURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Fragment != "" {
		return fmt.Errorf("%q must be an absolute URI without a fragment", value)
	}
	return nil
}
