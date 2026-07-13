package node

import "testing"

func TestBindableFieldsForNode_MergesConfigOutputs(t *testing.T) {
	spec := Spec{
		DynamicPorts: []DynamicPortSpec{{Role: DynamicPortOutputData, ConfigKey: "ResultFields", Shape: DynamicPortNameTypeRecords, ParentOutput: "Done", MaxItems: 8}},
		Outputs: []OutputSpec{
			{Name: "Done", Type: TypeExec, Data: []DataField{{Name: "Text", Type: "String"}}},
		},
	}
	config := map[string]any{"ResultFields": []any{
		map[string]any{"Name": "Count", "Type": "Integer"},
		map[string]any{"Name": "Text", "Type": "String"}, // 与静态 Text 重名 → 去重
	}}
	got := BindableFieldsForNode(&spec, config)
	if len(got) != 2 || got[0] != "Text" || got[1] != "Count" {
		t.Fatalf("BindableFieldsForNode = %v, want [Text Count]", got)
	}
}

func TestBindableFieldsForNode_NonDynamicIgnoresConfig(t *testing.T) {
	spec := Spec{Outputs: []OutputSpec{
		{Name: "Done", Type: TypeExec, Data: []DataField{{Name: "X", Type: "String"}}},
	}}
	config := map[string]any{"Outputs": []any{map[string]any{"Name": "Y", "Type": "String"}}}
	got := BindableFieldsForNode(&spec, config)
	if len(got) != 1 || got[0] != "X" {
		t.Errorf("node without outputData descriptor must ignore config.Outputs, got %v", got)
	}
}
