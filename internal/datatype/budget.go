package datatype

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
)

var typeVersionPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)

func inspectJSONBudget(raw []byte, maxDepth, maxNodes int) error {
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
		if nodes > maxNodes {
			return errors.New("JSON node budget exceeded")
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{', '[':
				depth++
				if depth > maxDepth {
					return errors.New("JSON depth budget exceeded")
				}
			case '}', ']':
				depth--
			}
		case string:
			if len(value) > MaxSchemaResourceBytes {
				return errors.New("JSON string budget exceeded")
			}
		}
	}
}

func decodeJSONValue(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func collectBundledResourceIDs(value any, base *url.URL, bundled map[string]bool, depth int, root bool) error {
	if depth > MaxSchemaDepth {
		return errors.New("schema identifier walk exceeds depth budget")
	}
	switch typed := value.(type) {
	case map[string]any:
		localBase, err := schemaObjectBase(typed, base)
		if err != nil {
			return err
		}
		if localBase.String() != base.String() || !root {
			if _, hasID := typed["$id"]; hasID {
				if bundled[localBase.String()] {
					return fmt.Errorf("duplicate bundled schema id %q", localBase.String())
				}
				bundled[localBase.String()] = true
			}
		}
		return forEachSubschema(typed, func(subschema any) error {
			return collectBundledResourceIDs(subschema, localBase, bundled, depth+1, false)
		})
	}
	return nil
}

func validateBundledReferences(value any, base *url.URL, bundled map[string]bool, depth int, referenceCount *int) error {
	if depth > MaxSchemaDepth {
		return errors.New("schema reference walk exceeds depth budget")
	}
	switch typed := value.(type) {
	case map[string]any:
		localBase, err := schemaObjectBase(typed, base)
		if err != nil {
			return err
		}
		for _, key := range []string{"$ref", "$dynamicRef"} {
			item, exists := typed[key]
			if exists {
				*referenceCount++
				if *referenceCount > MaxSchemaReferences {
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
		}
		return forEachSubschema(typed, func(subschema any) error {
			return validateBundledReferences(subschema, localBase, bundled, depth+1, referenceCount)
		})
	}
	return nil
}

func schemaObjectBase(object map[string]any, base *url.URL) (*url.URL, error) {
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

func sortedObjectKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
		for _, name := range sortedObjectKeys(entries) {
			if !isSchema(entries[name]) {
				return fmt.Errorf("schema keyword %q must contain only schemas", keyword)
			}
			if err := visit(entries[name]); err != nil {
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
