// internal/node/schema.go
package node

// FieldSchema 结构化输入的递归数据 schema. 挂 InputSpec.Schema, 经 Spec RPC 到 FE,
// 前端 StructuredInput 递归渲染. 数据类型 (Type) 与 UI widget (Widget) 分离:
// geometry = Type:"object" + Widget:"geometry", 不污染 Type 系统.
// Spec 1 只承载渲染 + 基础校验 (FieldEntry.Required); 复杂业务校验 (min/max/pattern) 不进 schema.
type FieldSchema struct {
	Type    string       `json:"type"`              // object | number | string | bool | enum
	Widget  string       `json:"widget,omitempty"`  // "" 默认控件 | "geometry"
	Fields  []FieldEntry `json:"fields,omitempty"`  // Type=object
	Options []EnumOption `json:"options,omitempty"` // Type=enum (只填 Value, label 走 i18n)
}

// FieldEntry object 的有序子字段. label/hint 走自动 fieldPath i18n key (不在此存显式 key).
type FieldEntry struct {
	Key      string       `json:"key"`
	Schema   *FieldSchema `json:"schema"`
	Required bool         `json:"required,omitempty"`
}

func ObjSchema(fields ...FieldEntry) *FieldSchema { return &FieldSchema{Type: "object", Fields: fields} }
func Field(key string, s *FieldSchema, required bool) FieldEntry {
	return FieldEntry{Key: key, Schema: s, Required: required}
}
func NumberSchema() *FieldSchema { return &FieldSchema{Type: "number"} }
func StringSchema() *FieldSchema { return &FieldSchema{Type: "string"} }
func BoolSchema() *FieldSchema   { return &FieldSchema{Type: "bool"} }
func EnumSchema(opts ...EnumOption) *FieldSchema { return &FieldSchema{Type: "enum", Options: opts} }
func GeometrySchema() *FieldSchema { return &FieldSchema{Type: "object", Widget: "geometry"} }
