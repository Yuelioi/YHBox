package schema

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseSource31PreservesExplicitBindingStateAndChannels(t *testing.T) {
	raw := fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1",
		"workflow":{"id":"wf-1","name":"Concat"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"concat-1","nodeRef":{"nodeTypeId":"https://schemas.yotta.dev/nodes/text/concat/v1","semanticDigest":"sha256:%s"},
			"position":{"x":0,"y":0},"config":{},
			"bindings":{"a":{"kind":"value","value":"hello"},"b":{"kind":"default"}}
		}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, strings.Repeat("1", 64))
	source, diagnostics := ParseSource([]byte(raw))
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	node := source.Graphs[0].Nodes[0]
	if node.NodeRef.NodeTypeID == "" || node.Bindings["a"].Kind != BindingValue || string(node.Bindings["a"].Value) != `"hello"` || node.Bindings["b"].Kind != BindingDefault {
		t.Fatalf("parsed node = %#v", node)
	}
}

func TestParseSource31RejectsLegacyKindAndImplicitEdgeChannel(t *testing.T) {
	legacy := `{"format":"yotta.workflow","version":3,"workflow":{"id":"wf","name":"x"},"revision":0,"entryGraph":"main","graphs":[],"variables":[],"secretRefs":[]}`
	if _, diagnostics := ParseSource([]byte(legacy)); len(diagnostics) == 0 || diagnostics[0].Code != CodeUnsupportedWorkflowFormat {
		t.Fatalf("legacy diagnostics = %#v", diagnostics)
	}
}

func TestParseSource31RejectsUnknownDuplicateAndMalformedBindingState(t *testing.T) {
	valid := validSource31ForTest()
	tests := []struct {
		name string
		raw  string
		code string
	}{
		{"unknown", strings.Replace(valid, `"revision":0`, `"revision":0,"legacy":true`, 1), CodeUnknownField},
		{"duplicate", strings.Replace(valid, `"revision":0`, `"revision":0,"revision":1`, 1), CodeDuplicateField},
		{"value absent", strings.Replace(valid, `{"kind":"value","value":"hello"}`, `{"kind":"value"}`, 1), CodeInvalidField},
		{"default has value", strings.Replace(valid, `{"kind":"default"}`, `{"kind":"default","value":"x"}`, 1), CodeInvalidField},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, diagnostics := ParseSource([]byte(test.raw))
			if len(diagnostics) == 0 || diagnostics[0].Code != test.code {
				t.Fatalf("diagnostics = %#v", diagnostics)
			}
		})
	}
}

func TestParseSource31RequiresExplicitEdgeChannel(t *testing.T) {
	raw := strings.Replace(validSource31ForTest(), `"edges":[]`, `"edges":[{"from":{"nodeId":"a","portId":"result"},"to":{"nodeId":"b","portId":"a"}}]`, 1)
	_, diagnostics := ParseSource([]byte(raw))
	if len(diagnostics) == 0 || diagnostics[0].Code != CodeMissingRequiredField {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestParseSource31ValidatesNestedTypeExpressionsAndWholeDocumentBudget(t *testing.T) {
	invalidType := strings.Replace(validSource31ForTest(), `"variables":[]`, fmt.Sprintf(`"variables":[{"name":"bad","type":{"kind":"ref","ref":{"typeId":"https://schemas.yotta.dev/types/bad","semanticDigest":"sha256:%s"}}}]`, strings.Repeat("2", 64)), 1)
	if _, diagnostics := ParseSource([]byte(invalidType)); len(diagnostics) == 0 || diagnostics[0].Code != CodeInvalidField {
		t.Fatalf("invalid TypeExpression diagnostics = %#v", diagnostics)
	}
	deep := strings.Replace(validSource31ForTest(), `"config":{}`, `"config":{"deep":`+strings.Repeat("[", 129)+`0`+strings.Repeat("]", 129)+`}`, 1)
	if _, diagnostics := ParseSource([]byte(deep)); len(diagnostics) == 0 || diagnostics[0].Code != CodeInvalidWorkflowJSON {
		t.Fatalf("deep source diagnostics = %#v", diagnostics)
	}
}

func validSource31ForTest() string {
	return fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1",
		"workflow":{"id":"wf-1","name":"Concat"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"concat-1","nodeRef":{"nodeTypeId":"https://schemas.yotta.dev/nodes/text/concat/v1","semanticDigest":"sha256:%s"},
			"position":{"x":0,"y":0},"config":{},
			"bindings":{"a":{"kind":"value","value":"hello"},"b":{"kind":"default"}}
		}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, strings.Repeat("1", 64))
}
