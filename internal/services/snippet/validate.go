package snippet

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var forbiddenPayloadKeys = map[string]struct{}{
	"grant": {}, "grants": {}, "secret": {}, "secrets": {}, "credential": {}, "credentials": {},
	"token": {}, "tokens": {}, "handle": {}, "handles": {}, "runtimehandle": {}, "capabilitygrant": {},
}

func validate(value Snippet) error {
	if value.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported snippet schema version %q", value.SchemaVersion)
	}
	if !idPattern.MatchString(value.ID) {
		return errors.New("invalid snippet id")
	}
	if name := strings.TrimSpace(value.Name); name == "" || len([]rune(name)) > 80 {
		return errors.New("snippet name must contain 1 to 80 characters")
	}
	if len([]rune(value.Description)) > 1000 {
		return errors.New("snippet description exceeds 1000 characters")
	}
	if len([]rune(value.Category)) > 80 {
		return errors.New("snippet category exceeds 80 characters")
	}
	if len(value.Tags) > 32 {
		return errors.New("snippet has more than 32 tags")
	}
	if value.UsageCount < 0 {
		return errors.New("snippet usage count cannot be negative")
	}
	for _, tag := range value.Tags {
		if strings.TrimSpace(tag) == "" || len([]rune(tag)) > 64 {
			return errors.New("snippet tag must contain 1 to 64 characters")
		}
	}
	if strings.TrimSpace(value.Payload.NodeRef.NodeTypeID) == "" ||
		strings.TrimSpace(value.Payload.NodeRef.Version) == "" ||
		strings.TrimSpace(string(value.Payload.NodeRef.SemanticDigest)) == "" {
		return errors.New("snippet node reference is incomplete")
	}
	if value.Payload.Config == nil || value.Payload.Bindings == nil {
		return errors.New("snippet payload requires config and bindings")
	}
	if key, found := findForbiddenKey(value.Payload.Config); found {
		return fmt.Errorf("snippet payload cannot persist sensitive runtime field %q", key)
	}
	for portID, binding := range value.Payload.Bindings {
		if len(binding.Value) == 0 {
			continue
		}
		var literal any
		if err := json.Unmarshal(binding.Value, &literal); err != nil {
			return fmt.Errorf("snippet binding %q has invalid JSON value: %w", portID, err)
		}
		if key, found := findForbiddenKey(literal); found {
			return fmt.Errorf("snippet binding %q cannot persist sensitive runtime field %q", portID, key)
		}
	}
	return nil
}

func findForbiddenKey(value any) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.ToLower(strings.NewReplacer("-", "", "_", "", ".", "").Replace(key))
			if _, forbidden := forbiddenPayloadKeys[normalized]; forbidden {
				return key, true
			}
			if nested, found := findForbiddenKey(child); found {
				return nested, true
			}
		}
	case []any:
		for _, child := range typed {
			if nested, found := findForbiddenKey(child); found {
				return nested, true
			}
		}
	}
	return "", false
}
