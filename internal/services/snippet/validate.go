package snippet

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var functionKeyPattern = regexp.MustCompile(`^F([1-9]|1[0-2])$`)

var reservedShortcuts = map[string]struct{}{
	"Ctrl+A": {}, "Ctrl+C": {}, "Ctrl+D": {}, "Ctrl+F": {}, "Ctrl+S": {},
	"Ctrl+V": {}, "Ctrl+X": {}, "Ctrl+Y": {}, "Ctrl+Z": {}, "Ctrl+Shift+Z": {},
}

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
	if _, err := normalizeShortcut(value.Shortcut); err != nil {
		return err
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

func normalizeShortcut(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len([]rune(value)) > 64 {
		return "", errors.New("snippet shortcut exceeds 64 characters")
	}
	modifiers := map[string]bool{}
	key := ""
	for _, raw := range strings.Split(value, "+") {
		part := strings.TrimSpace(raw)
		switch strings.ToLower(part) {
		case "ctrl", "control":
			modifiers["Ctrl"] = true
		case "shift":
			modifiers["Shift"] = true
		case "alt":
			modifiers["Alt"] = true
		case "meta", "cmd", "command":
			modifiers["Meta"] = true
		default:
			if part == "" || key != "" {
				return "", errors.New("snippet shortcut must contain exactly one key")
			}
			key = canonicalShortcutKey(part)
		}
	}
	if key == "" {
		return "", errors.New("snippet shortcut requires a key")
	}
	if !validShortcutKey(key) {
		return "", fmt.Errorf("snippet shortcut key %q is unsupported", key)
	}
	if len(modifiers) == 0 && !functionKeyPattern.MatchString(key) {
		return "", errors.New("snippet shortcut requires a modifier or an F1-F12 key")
	}
	parts := make([]string, 0, len(modifiers)+1)
	for _, modifier := range []string{"Ctrl", "Shift", "Alt", "Meta"} {
		if modifiers[modifier] {
			parts = append(parts, modifier)
		}
	}
	parts = append(parts, key)
	result := strings.Join(parts, "+")
	if _, reserved := reservedShortcuts[result]; reserved {
		return "", fmt.Errorf("snippet shortcut %q is reserved by the editor", result)
	}
	return result, nil
}

func validShortcutKey(value string) bool {
	if len(value) == 1 {
		return (value[0] >= 'A' && value[0] <= 'Z') ||
			(value[0] >= '0' && value[0] <= '9') || value == "." || value == ","
	}
	if functionKeyPattern.MatchString(value) {
		return true
	}
	switch value {
	case "Space", "Enter", "Tab", "Delete", "Insert", "Home", "End", "PgUp", "PgDn", "Up", "Down", "Left", "Right":
		return true
	default:
		return false
	}
}

func canonicalShortcutKey(value string) string {
	upper := strings.ToUpper(strings.TrimSpace(value))
	if len([]rune(upper)) == 1 || functionKeyPattern.MatchString(upper) {
		return upper
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pgup", "pageup":
		return "PgUp"
	case "pgdn", "pagedown":
		return "PgDn"
	case "space":
		return "Space"
	case "enter":
		return "Enter"
	case "tab":
		return "Tab"
	case "delete":
		return "Delete"
	case "insert":
		return "Insert"
	case "home":
		return "Home"
	case "end":
		return "End"
	case "up", "down", "left", "right":
		return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
	default:
		return strings.TrimSpace(value)
	}
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
