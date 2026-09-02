package snippet

import (
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const SchemaVersion = "1"

type NodeTemplate struct {
	NodeRef  nodecontract.NodeRef           `json:"nodeRef"`
	Label    string                         `json:"label,omitempty"`
	Config   map[string]any                 `json:"config"`
	Bindings map[string]schema.InputBinding `json:"bindings"`
	Disabled bool                           `json:"disabled,omitempty"`
}

type Snippet struct {
	SchemaVersion string       `json:"schemaVersion"`
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Description   string       `json:"description,omitempty"`
	Category      string       `json:"category,omitempty"`
	Tags          []string     `json:"tags"`
	Shortcut      string       `json:"shortcut,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
	UsageCount    int          `json:"usageCount"`
	LastUsedAt    *time.Time   `json:"lastUsedAt,omitempty"`
	Payload       NodeTemplate `json:"payload"`
}

type Summary struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Category    string     `json:"category,omitempty"`
	Tags        []string   `json:"tags"`
	Shortcut    string     `json:"shortcut,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	UsageCount  int        `json:"usageCount"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	NodeTypeID  string     `json:"nodeTypeId"`
}

type LoadWarning struct {
	File    string           `json:"file"`
	Problem *apperr.Envelope `json:"problem"`
}

type ListResult struct {
	Items    []Summary     `json:"items"`
	Warnings []LoadWarning `json:"warnings"`
}

func summary(value Snippet) Summary {
	return Summary{
		ID: value.ID, Name: value.Name, Description: value.Description, Category: value.Category,
		Tags: append([]string(nil), value.Tags...), Shortcut: value.Shortcut, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
		UsageCount: value.UsageCount, LastUsedAt: cloneTime(value.LastUsedAt),
		NodeTypeID: value.Payload.NodeRef.NodeTypeID,
	}
}

func clone(value Snippet) Snippet {
	result := value
	result.Tags = append([]string(nil), value.Tags...)
	result.LastUsedAt = cloneTime(value.LastUsedAt)
	result.Payload.Config = cloneMap(value.Payload.Config)
	result.Payload.Bindings = make(map[string]schema.InputBinding, len(value.Payload.Bindings))
	for key, binding := range value.Payload.Bindings {
		binding.Value = append([]byte(nil), binding.Value...)
		result.Payload.Bindings[key] = binding
	}
	return result
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		switch typed := value.(type) {
		case map[string]any:
			result[key] = cloneMap(typed)
		case []any:
			result[key] = cloneSlice(typed)
		default:
			result[key] = value
		}
	}
	return result
}

func cloneSlice(source []any) []any {
	result := make([]any, len(source))
	for index, value := range source {
		switch typed := value.(type) {
		case map[string]any:
			result[index] = cloneMap(typed)
		case []any:
			result[index] = cloneSlice(typed)
		default:
			result[index] = value
		}
	}
	return result
}

func normalizeTags(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		tag := strings.TrimSpace(value)
		key := strings.ToLower(tag)
		if key == "" {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	return result
}
