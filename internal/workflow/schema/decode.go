package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	runtimejsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

var workflowValidator = sync.OnceValues(compileWorkflowValidator)

func ParseSource(raw []byte) (WorkflowSource, []Diagnostic) {
	if len(raw) == 0 || !utf8.Valid(raw) || bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": "source must be non-empty UTF-8 without BOM"})
	}
	if duplicate, err := firstDuplicateField(raw); err != nil {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": err.Error()})
	} else if duplicate != nil {
		return WorkflowSource{}, oneDiagnostic(CodeDuplicateField, duplicate, map[string]any{"field": duplicate[len(duplicate)-1]})
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": err.Error()})
	}
	if err := requireEOF(decoder); err != nil {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": err.Error()})
	}
	envelope, ok := document.(map[string]any)
	if !ok {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": "source must be a JSON object"})
	}
	var format string
	formatValue, hasFormat := envelope["format"]
	versionValue, hasVersion := envelope["version"]
	format, formatOK := formatValue.(string)
	versionNumber, versionOK := versionValue.(json.Number)
	if !hasFormat || !hasVersion || !formatOK || !versionOK || format != Format || !isCurrentVersion(versionNumber) {
		return WorkflowSource{}, oneDiagnostic(CodeUnsupportedWorkflowFormat, nil, map[string]any{"expectedFormat": Format, "expectedVersion": Version})
	}
	if schemaContractViolationBudgetExceeded(document, reflect.TypeFor[WorkflowSource](), MaxDiagnostics) {
		return WorkflowSource{}, oneDiagnostic(CodeDiagnosticBudgetExceeded, nil, map[string]any{"maximum": MaxDiagnostics})
	}

	validator, err := workflowValidator()
	if err != nil {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": "workflow contract is unavailable: " + err.Error()})
	}
	if err := validator.Validate(document); err != nil {
		var validationError *runtimejsonschema.ValidationError
		if errors.As(err, &validationError) {
			return WorkflowSource{}, diagnosticsFromSchemaError(validationError, document)
		}
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": err.Error()})
	}

	var source WorkflowSource
	if err := decodeValidatedSource(document, &source); err != nil {
		return WorkflowSource{}, oneDiagnostic(CodeInvalidWorkflowJSON, nil, map[string]any{"reason": err.Error()})
	}
	diagnostics := validateSource(source)
	if hasError(diagnostics) {
		return WorkflowSource{}, diagnostics
	}
	return source, diagnostics
}

func decodeValidatedSource(document any, source *WorkflowSource) error {
	root := document.(map[string]any)
	canonical := make(map[string]any, len(root))
	for key, value := range root {
		canonical[key] = value
	}
	canonical["version"] = json.Number(strconv.Itoa(Version))
	revision, ok := new(big.Rat).SetString(root["revision"].(json.Number).String())
	if !ok || !revision.IsInt() {
		return errors.New("validated revision is not an integer")
	}
	canonical["revision"] = json.Number(revision.Num().String())
	raw, err := json.Marshal(canonical)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, source)
}

func isCurrentVersion(number json.Number) bool {
	value, ok := new(big.Rat).SetString(number.String())
	return ok && value.Cmp(big.NewRat(Version, 1)) == 0
}

type structuralViolationCounter struct {
	count, limit int
}

func (c *structuralViolationCounter) add() bool {
	c.count++
	return c.count >= c.limit
}

type reflectedFieldContract struct {
	typ       reflect.Type
	required  bool
	minLength bool
	enum      map[string]bool
	maxItems  int
}

type reflectedStructContract struct {
	fields map[string]reflectedFieldContract
}

var reflectedContracts sync.Map

func schemaContractViolationBudgetExceeded(document any, typ reflect.Type, limit int) bool {
	counter := &structuralViolationCounter{limit: limit}
	inspectSchemaValue(document, typ, reflectedFieldContract{}, counter)
	return counter.count >= limit
}

func inspectSchemaValue(value any, typ reflect.Type, field reflectedFieldContract, counter *structuralViolationCounter) {
	if counter.count >= counter.limit {
		return
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Struct:
		object, ok := value.(map[string]any)
		if !ok {
			counter.add()
			return
		}
		contract := reflectedContract(typ)
		for key := range object {
			if _, ok := contract.fields[key]; !ok && counter.add() {
				return
			}
		}
		for name, child := range contract.fields {
			childValue, exists := object[name]
			if !exists {
				if child.required {
					counter.add()
				}
				continue
			}
			inspectSchemaValue(childValue, child.typ, child, counter)
		}
	case reflect.Slice, reflect.Array:
		values, ok := value.([]any)
		if !ok {
			counter.add()
			return
		}
		if field.maxItems > 0 && len(values) > field.maxItems {
			counter.count = counter.limit
			return
		}
		for _, child := range values {
			inspectSchemaValue(child, typ.Elem(), reflectedFieldContract{}, counter)
			if counter.count >= counter.limit {
				return
			}
		}
	case reflect.Map:
		if _, ok := value.(map[string]any); !ok {
			counter.add()
		}
	case reflect.Interface:
		return
	case reflect.String:
		text, ok := value.(string)
		if !ok || (field.minLength || typ == reflect.TypeFor[Capability]()) && text == "" || len(field.enum) > 0 && !field.enum[text] {
			counter.add()
		}
	case reflect.Bool:
		if _, ok := value.(bool); !ok {
			counter.add()
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		if _, ok := value.(json.Number); !ok {
			counter.add()
		}
	}
}

func reflectedContract(typ reflect.Type) reflectedStructContract {
	if cached, ok := reflectedContracts.Load(typ); ok {
		return cached.(reflectedStructContract)
	}
	contract := reflectedStructContract{fields: make(map[string]reflectedFieldContract, typ.NumField())}
	for index := 0; index < typ.NumField(); index++ {
		modelField := typ.Field(index)
		name := strings.Split(modelField.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		field := reflectedFieldContract{typ: modelField.Type, enum: map[string]bool{}}
		for _, option := range strings.Split(modelField.Tag.Get("jsonschema"), ",") {
			switch {
			case option == "required":
				field.required = true
			case option == "minLength=1":
				field.minLength = true
			case strings.HasPrefix(option, "enum="):
				field.enum[strings.TrimPrefix(option, "enum=")] = true
			case strings.HasPrefix(option, "maxItems="):
				field.maxItems, _ = strconv.Atoi(strings.TrimPrefix(option, "maxItems="))
			}
		}
		contract.fields[name] = field
	}
	actual, _ := reflectedContracts.LoadOrStore(typ, contract)
	return actual.(reflectedStructContract)
}

func validateSource(source WorkflowSource) []Diagnostic {
	var out []Diagnostic

	graphIDs := map[string]bool{}
	graphKinds := map[string]GraphKind{}
	mainGraphs := 0
	for graphIndex, graph := range source.Graphs {
		if len(out) >= MaxDiagnostics {
			return sortedDiagnostics(out)
		}
		graphPath := []string{graph.ID}
		base := []string{"graphs", fmt.Sprint(graphIndex)}
		if graphIDs[graph.ID] {
			appendDiagnostic(&out, diagnosticAtGraph(CodeDuplicateID, graphPath, append(base, "id"), map[string]any{"id": graph.ID}))
		}
		graphIDs[graph.ID] = true
		graphKinds[graph.ID] = graph.Kind
		if graph.Kind == GraphKindMain {
			mainGraphs++
		}
		nodeIDs := map[string]bool{}
		for nodeIndex, node := range graph.Nodes {
			if len(out) >= MaxDiagnostics {
				return sortedDiagnostics(out)
			}
			nodePath := append(append([]string(nil), base...), "nodes", fmt.Sprint(nodeIndex))
			if nodeIDs[node.ID] {
				appendDiagnostic(&out, Diagnostic{Code: CodeDuplicateID, Severity: SeverityError, GraphPath: graphPath, NodeID: node.ID, FieldPath: append(nodePath, "id"), Params: map[string]any{"id": node.ID}})
			}
			nodeIDs[node.ID] = true
		}
		validateGraphPorts := func(ports []GraphPort, field string) {
			portIDs := map[string]bool{}
			for portIndex, port := range ports {
				if len(out) >= MaxDiagnostics {
					return
				}
				portPath := append(append([]string(nil), base...), field, fmt.Sprint(portIndex))
				if portIDs[port.ID] {
					appendDiagnostic(&out, diagnosticAtGraph(CodeDuplicateID, graphPath, append(portPath, "id"), map[string]any{"id": port.ID}))
				}
				portIDs[port.ID] = true
			}
		}
		validateGraphPorts(graph.Inputs, "inputs")
		validateGraphPorts(graph.Outputs, "outputs")
		if len(out) >= MaxDiagnostics {
			return sortedDiagnostics(out)
		}
	}
	if !graphIDs[source.EntryGraph] || graphKinds[source.EntryGraph] != GraphKindMain || mainGraphs != 1 {
		appendDiagnostic(&out, diagnostic(CodeMissingEntryGraph, []string{"entryGraph"}, map[string]any{"entryGraph": source.EntryGraph, "mainGraphs": mainGraphs}))
	}
	variableNames := map[string]bool{}
	for index, variable := range source.Variables {
		if len(out) >= MaxDiagnostics {
			return sortedDiagnostics(out)
		}
		path := []string{"variables", fmt.Sprint(index)}
		if variableNames[variable.Name] {
			appendDiagnostic(&out, diagnostic(CodeDuplicateID, append(path, "name"), map[string]any{"id": variable.Name}))
		}
		variableNames[variable.Name] = true
	}
	secretIDs := map[string]bool{}
	for index, secret := range source.SecretRefs {
		if len(out) >= MaxDiagnostics {
			return sortedDiagnostics(out)
		}
		path := []string{"secretRefs", fmt.Sprint(index)}
		if secretIDs[secret.ID] {
			appendDiagnostic(&out, diagnostic(CodeDuplicateID, append(path, "id"), map[string]any{"id": secret.ID}))
		}
		secretIDs[secret.ID] = true
	}
	capabilities := map[Capability]bool{}
	for index, capability := range source.RequestedCapabilities {
		if len(out) >= MaxDiagnostics {
			return sortedDiagnostics(out)
		}
		path := []string{"requestedCapabilities", fmt.Sprint(index)}
		if capability == "" {
			appendDiagnostic(&out, diagnostic(CodeInvalidField, path, map[string]any{"keyword": "minLength"}))
		} else if capabilities[capability] {
			appendDiagnostic(&out, diagnostic(CodeDuplicateID, path, map[string]any{"id": capability}))
		}
		capabilities[capability] = true
	}
	return sortedDiagnostics(out)
}

func sortedDiagnostics(values []Diagnostic) []Diagnostic {
	sortDiagnostics(values)
	return values
}

func compileWorkflowValidator() (*runtimejsonschema.Schema, error) {
	raw, err := GenerateContract("workflow")
	if err != nil {
		return nil, err
	}
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	compiler := runtimejsonschema.NewCompiler()
	if err := compiler.AddResource(workflowSchemaID, document); err != nil {
		return nil, err
	}
	return compiler.Compile(workflowSchemaID)
}

func diagnosticsFromSchemaError(validationError *runtimejsonschema.ValidationError, document any) []Diagnostic {
	var out []Diagnostic
	var visit func(*runtimejsonschema.ValidationError)
	visit = func(current *runtimejsonschema.ValidationError) {
		if len(out) >= MaxDiagnostics {
			return
		}
		if len(current.Causes) > 0 {
			for _, cause := range current.Causes {
				visit(cause)
				if len(out) >= MaxDiagnostics {
					return
				}
			}
			return
		}
		base := append([]string(nil), current.InstanceLocation...)
		keyword := strings.Join(current.ErrorKind.KeywordPath(), "/")
		switch typed := current.ErrorKind.(type) {
		case *kind.Required:
			for _, field := range typed.Missing {
				appendDiagnostic(&out, schemaDiagnostic(CodeMissingRequiredField, appendPath(base, field), keyword, document))
			}
		case *kind.AdditionalProperties:
			for _, field := range typed.Properties {
				appendDiagnostic(&out, schemaDiagnostic(CodeUnknownField, appendPath(base, field), keyword, document))
			}
		default:
			appendDiagnostic(&out, schemaDiagnostic(CodeInvalidField, base, keyword, document))
		}
	}
	visit(validationError)
	sortDiagnostics(out)
	return out
}

func schemaDiagnostic(code string, fieldPath []string, keyword string, document any) Diagnostic {
	d := diagnostic(code, fieldPath, map[string]any{"keyword": keyword})
	d.GraphPath, d.NodeID = locationContext(document, fieldPath)
	return d
}

func locationContext(document any, fieldPath []string) ([]string, string) {
	root, _ := document.(map[string]any)
	graphs, _ := root["graphs"].([]any)
	if len(fieldPath) < 2 || fieldPath[0] != "graphs" {
		return nil, ""
	}
	graphIndex, err := strconv.Atoi(fieldPath[1])
	if err != nil || graphIndex < 0 || graphIndex >= len(graphs) {
		return nil, ""
	}
	graph, _ := graphs[graphIndex].(map[string]any)
	graphID, _ := graph["id"].(string)
	graphPath := []string{graphID}
	if len(fieldPath) < 4 || fieldPath[2] != "nodes" {
		return graphPath, ""
	}
	nodes, _ := graph["nodes"].([]any)
	nodeIndex, err := strconv.Atoi(fieldPath[3])
	if err != nil || nodeIndex < 0 || nodeIndex >= len(nodes) {
		return graphPath, ""
	}
	node, _ := nodes[nodeIndex].(map[string]any)
	nodeID, _ := node["id"].(string)
	return graphPath, nodeID
}

func appendPath(path []string, field string) []string {
	return append(append([]string(nil), path...), field)
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

func appendDiagnostic(out *[]Diagnostic, value Diagnostic) {
	if len(*out) < MaxDiagnostics-1 {
		*out = append(*out, value)
		return
	}
	if len(*out) == MaxDiagnostics-1 {
		*out = append(*out, diagnostic(CodeDiagnosticBudgetExceeded, nil, map[string]any{"maximum": MaxDiagnostics}))
	}
}

func hasError(diagnostics []Diagnostic) bool {
	return slices.ContainsFunc(diagnostics, func(d Diagnostic) bool { return d.Severity == SeverityError })
}

func sortDiagnostics(diagnostics []Diagnostic) {
	sortable := diagnostics
	if len(diagnostics) > 0 && diagnostics[len(diagnostics)-1].Code == CodeDiagnosticBudgetExceeded {
		sortable = diagnostics[:len(diagnostics)-1]
	}
	sort.SliceStable(sortable, func(i, j int) bool {
		left, right := sortable[i], sortable[j]
		for _, pair := range [][2]string{{strings.Join(left.GraphPath, "/"), strings.Join(right.GraphPath, "/")}, {left.NodeID, right.NodeID}, {strings.Join(left.FieldPath, "/"), strings.Join(right.FieldPath, "/")}, {left.Code, right.Code}} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
}
