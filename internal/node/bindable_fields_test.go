package node

import "testing"

func TestBindableFields(t *testing.T) {
	// 非纯数据节点: 取所有 exec 出口的 Data 字段名, 去重, 保留首现顺序.
	spec := Spec{
		Outputs: []OutputSpec{
			{Name: "Found", Type: TypeExec, Data: []DataField{
				{Name: "Count", Type: "Number"},
				{Name: "Center", Type: "Point"},
			}},
			{Name: "NotFound", Type: TypeExec, Data: []DataField{
				{Name: "Count", Type: "Number"}, // 重复 → 去重
			}},
			{Name: "Value", Type: "Number"}, // 非 exec 出口 → 不算可绑
		},
	}
	got := BindableFields(&spec)
	if len(got) != 2 || got[0] != "Count" || got[1] != "Center" {
		t.Errorf("BindableFields = %v, want [Count Center]", got)
	}
}

func TestBindableFields_PureDataNil(t *testing.T) {
	// 纯数据节点无可绑字段 (输出是连线源)。
	pd := Spec{IsPureData: true, Outputs: []OutputSpec{{Name: "Value", Type: "Number"}}}
	if got := BindableFields(&pd); got != nil {
		t.Errorf("pure-data BindableFields = %v, want nil", got)
	}
	if got := BindableFields(nil); got != nil {
		t.Errorf("nil spec BindableFields = %v, want nil", got)
	}
}
