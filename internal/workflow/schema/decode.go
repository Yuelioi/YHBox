package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

func ParseSource(raw []byte) (WorkflowSource, []Diagnostic) {
	if len(raw) == 0 || !utf8.Valid(raw) || bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": "source must be non-empty UTF-8 without BOM"})
	}
	if duplicate, err := firstDuplicateField(raw); err != nil {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": err.Error()})
	} else if duplicate != nil {
		return WorkflowSource{}, oneDiagnostic(CodeDuplicateField, duplicate, map[string]any{"field": duplicate[len(duplicate)-1]})
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": err.Error()})
	}
	var format string
	var version int
	formatRaw, hasFormat := envelope["format"]
	versionRaw, hasVersion := envelope["version"]
	if !hasFormat || !hasVersion || json.Unmarshal(formatRaw, &format) != nil || json.Unmarshal(versionRaw, &version) != nil || format != Format || version != Version {
		return WorkflowSource{}, oneDiagnostic(CodeUnsupportedWorkflowFormat, nil, map[string]any{"expectedFormat": Format, "expectedVersion": Version})
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var source WorkflowSource
	if err := decoder.Decode(&source); err != nil {
		code := CodeInvalidWorkflowJSON
		var fieldPath []string
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			code = CodeUnknownField
			field := strings.Trim(err.Error()[len("json: unknown field "):], "\"")
			fieldPath = []string{field}
		}
		return WorkflowSource{}, oneDiagnostic(code, fieldPath, map[string]any{"reason": err.Error()})
	}
	if err := requireEOF(decoder); err != nil {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": err.Error()})
	}

	diagnostics := validateSource(source, envelope)
	if hasError(diagnostics) {
		return WorkflowSource{}, diagnostics
	}
	return source, diagnostics
}

func validateSource(source WorkflowSource, envelope map[string]json.RawMessage) []Diagnostic {
	var out []Diagnostic
	requiredTop := []string{"format", "version", "workflow", "revision", "entryGraph", "graphs", "variables", "secretRefs", "requestedCapabilities"}
	for _, field := range requiredTop {
		if _, ok := envelope[field]; !ok {
			out = append(out, diagnostic(CodeMissingRequiredField, []string{field}, map[string]any{"field": field}))
		}
	}
	if source.Workflow.ID == "" {
		out = append(out, diagnostic(CodeMissingRequiredField, []string{"workflow", "id"}, map[string]any{"field": "id"}))
	}
	if source.Workflow.Name == "" {
		out = append(out, diagnostic(CodeMissingRequiredField, []string{"workflow", "name"}, map[string]any{"field": "name"}))
	}
	if source.Revision < 0 {
		out = append(out, diagnostic(CodeInvalidField, []string{"revision"}, map[string]any{"minimum": 0}))
	}
	if source.EntryGraph == "" {
		out = append(out, diagnostic(CodeMissingRequiredField, []string{"entryGraph"}, map[string]any{"field": "entryGraph"}))
	}
	if len(source.Graphs) == 0 {
		out = append(out, diagnostic(CodeMissingRequiredField, []string{"graphs"}, map[string]any{"field": "graphs"}))
	}
	for field, missing := range map[string]bool{"variables": source.Variables == nil, "secretRefs": source.SecretRefs == nil, "requestedCapabilities": source.RequestedCapabilities == nil} {
		if missing {
			out = append(out, diagnostic(CodeMissingRequiredField, []string{field}, map[string]any{"field": field}))
		}
	}

	graphIDs := map[string]bool{}
	graphKinds := map[string]GraphKind{}
	mainGraphs := 0
	for graphIndex, graph := range source.Graphs {
		graphPath := []string{graph.ID}
		base := []string{"graphs", fmt.Sprint(graphIndex)}
		if graph.ID == "" {
			out = append(out, diagnosticAtGraph(CodeMissingRequiredField, graphPath, append(base, "id"), map[string]any{"field": "id"}))
		} else if graphIDs[graph.ID] {
			out = append(out, diagnosticAtGraph(CodeDuplicateID, graphPath, append(base, "id"), map[string]any{"id": graph.ID}))
		}
		graphIDs[graph.ID] = true
		graphKinds[graph.ID] = graph.Kind
		if graph.Kind == GraphKindMain {
			mainGraphs++
		} else if graph.Kind != GraphKindSubgraph {
			out = append(out, diagnosticAtGraph(CodeInvalidField, graphPath, append(base, "kind"), map[string]any{"value": graph.Kind}))
		}
		for field, missing := range map[string]bool{"nodes": graph.Nodes == nil, "edges": graph.Edges == nil, "inputs": graph.Inputs == nil, "outputs": graph.Outputs == nil} {
			if missing {
				out = append(out, diagnosticAtGraph(CodeMissingRequiredField, graphPath, append(base, field), map[string]any{"field": field}))
			}
		}
		nodeIDs := map[string]bool{}
		for nodeIndex, node := range graph.Nodes {
			nodePath := append(append([]string(nil), base...), "nodes", fmt.Sprint(nodeIndex))
			if node.ID == "" || node.Kind == "" || node.Config == nil {
				field := "id"
				if node.ID != "" && node.Kind == "" {
					field = "kind"
				}
				if node.ID != "" && node.Kind != "" && node.Config == nil {
					field = "config"
				}
				out = append(out, Diagnostic{Code: CodeMissingRequiredField, Severity: SeverityError, GraphPath: graphPath, NodeID: node.ID, FieldPath: append(nodePath, field), Params: map[string]any{"field": field}})
			} else if nodeIDs[node.ID] {
				out = append(out, Diagnostic{Code: CodeDuplicateID, Severity: SeverityError, GraphPath: graphPath, NodeID: node.ID, FieldPath: append(nodePath, "id"), Params: map[string]any{"id": node.ID}})
			}
			nodeIDs[node.ID] = true
		}
		validateGraphPorts := func(ports []GraphPort, field string) {
			portIDs := map[string]bool{}
			for portIndex, port := range ports {
				portPath := append(append([]string(nil), base...), field, fmt.Sprint(portIndex))
				if port.ID == "" || port.Name == "" || port.Type == "" || port.NodeID == "" {
					out = append(out, diagnosticAtGraph(CodeMissingRequiredField, graphPath, portPath, map[string]any{"field": field}))
				} else if portIDs[port.ID] {
					out = append(out, diagnosticAtGraph(CodeDuplicateID, graphPath, append(portPath, "id"), map[string]any{"id": port.ID}))
				}
				portIDs[port.ID] = true
			}
		}
		validateGraphPorts(graph.Inputs, "inputs")
		validateGraphPorts(graph.Outputs, "outputs")
	}
	if !graphIDs[source.EntryGraph] || graphKinds[source.EntryGraph] != GraphKindMain || mainGraphs != 1 {
		out = append(out, diagnostic(CodeMissingEntryGraph, []string{"entryGraph"}, map[string]any{"entryGraph": source.EntryGraph, "mainGraphs": mainGraphs}))
	}
	variableNames := map[string]bool{}
	for index, variable := range source.Variables {
		path := []string{"variables", fmt.Sprint(index)}
		if variable.Name == "" || variable.Type == "" {
			out = append(out, diagnostic(CodeMissingRequiredField, path, map[string]any{"field": "variables"}))
		} else if variableNames[variable.Name] {
			out = append(out, diagnostic(CodeDuplicateID, append(path, "name"), map[string]any{"id": variable.Name}))
		}
		variableNames[variable.Name] = true
	}
	secretIDs := map[string]bool{}
	for index, secret := range source.SecretRefs {
		path := []string{"secretRefs", fmt.Sprint(index)}
		if secret.ID == "" || secret.Purpose == "" {
			out = append(out, diagnostic(CodeMissingRequiredField, path, map[string]any{"field": "secretRefs"}))
		} else if secretIDs[secret.ID] {
			out = append(out, diagnostic(CodeDuplicateID, append(path, "id"), map[string]any{"id": secret.ID}))
		}
		secretIDs[secret.ID] = true
	}
	capabilities := map[Capability]bool{}
	for index, capability := range source.RequestedCapabilities {
		path := []string{"requestedCapabilities", fmt.Sprint(index)}
		if capability == "" {
			out = append(out, diagnostic(CodeInvalidField, path, map[string]any{"reason": "capability is empty"}))
		} else if capabilities[capability] {
			out = append(out, diagnostic(CodeDuplicateID, path, map[string]any{"id": capability}))
		}
		capabilities[capability] = true
	}
	sortDiagnostics(out)
	return out
}

func firstDuplicateField(raw []byte) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	duplicate, err := scanJSONValue(decoder, nil)
	if err != nil {
		return nil, err
	}
	if duplicate != nil {
		return duplicate, nil
	}
	if err := requireEOF(decoder); err != nil {
		return nil, err
	}
	return nil, nil
}

func scanJSONValue(decoder *json.Decoder, path []string) ([]string, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil, nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, errors.New("object key is not a string")
			}
			keyPath := append(append([]string(nil), path...), key)
			if seen[key] {
				return keyPath, nil
			}
			seen[key] = true
			if duplicate, err := scanJSONValue(decoder, keyPath); err != nil || duplicate != nil {
				return duplicate, err
			}
		}
		_, err = decoder.Token()
		return nil, err
	case '[':
		index := 0
		for decoder.More() {
			itemPath := append(append([]string(nil), path...), fmt.Sprint(index))
			if duplicate, err := scanJSONValue(decoder, itemPath); err != nil || duplicate != nil {
				return duplicate, err
			}
			index++
		}
		_, err = decoder.Token()
		return nil, err
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

func requireEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func diagnostic(code string, fieldPath []string, params map[string]any) Diagnostic {
	return Diagnostic{Code: code, Severity: SeverityError, FieldPath: fieldPath, Params: params}
}

func diagnosticAtGraph(code string, graphPath, fieldPath []string, params map[string]any) Diagnostic {
	d := diagnostic(code, fieldPath, params)
	d.GraphPath = graphPath
	return d
}

func oneDiagnostic(code string, fieldPath []string, params map[string]any) []Diagnostic {
	return []Diagnostic{diagnostic(code, fieldPath, params)}
}

func hasError(diagnostics []Diagnostic) bool {
	return slices.ContainsFunc(diagnostics, func(d Diagnostic) bool { return d.Severity == SeverityError })
}

func sortDiagnostics(diagnostics []Diagnostic) {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		left, right := diagnostics[i], diagnostics[j]
		for _, pair := range [][2]string{{strings.Join(left.GraphPath, "/"), strings.Join(right.GraphPath, "/")}, {left.NodeID, right.NodeID}, {strings.Join(left.FieldPath, "/"), strings.Join(right.FieldPath, "/")}, {left.Code, right.Code}} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
}
