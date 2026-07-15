// Package authoring applies the only mutable Workflow 3.1 authoring protocol.
// Callers submit small domain commands against an exact source revision; they
// never replace a complete Workflow Source document.
package authoring

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const MaxCommands = 256

var handlePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

type PatchRequest struct {
	WorkflowID   string    `json:"workflowId" jsonschema:"required,maxLength=128,pattern=^[A-Za-z0-9_][A-Za-z0-9._-]*$"`
	BaseRevision int64     `json:"baseRevision" jsonschema:"required,minimum=0,maximum=9007199254740991"`
	Commands     []Command `json:"commands" jsonschema:"required,minItems=1,maxItems=256"`
}

type CommandKind string

const (
	CommandRenameWorkflow      CommandKind = "rename-workflow"
	CommandAddStateVariable    CommandKind = "add-state-variable"
	CommandRemoveStateVariable CommandKind = "remove-state-variable"
	CommandAddNode             CommandKind = "add-node"
	CommandRemoveNode          CommandKind = "remove-node"
	CommandMoveNode            CommandKind = "move-node"
	CommandSetNodeLabel        CommandKind = "set-node-label"
	CommandSetNodeDisabled     CommandKind = "set-node-disabled"
	CommandSetConfig           CommandKind = "set-config"
	CommandClearConfig         CommandKind = "clear-config"
	CommandBindValue           CommandKind = "bind-value"
	CommandBindDefault         CommandKind = "bind-default"
	CommandBindBlob            CommandKind = "bind-blob"
	CommandClearBinding        CommandKind = "clear-binding"
	CommandConnect             CommandKind = "connect"
	CommandDisconnect          CommandKind = "disconnect"
)

// Command is an explicit tagged union. Exactly one payload must be present and
// it must match Kind; this is checked again inside Engine.Apply for every
// caller, including Wails and MCP decoders.
type Command struct {
	Kind                CommandKind                 `json:"kind"`
	RenameWorkflow      *RenameWorkflowCommand      `json:"renameWorkflow,omitempty"`
	AddStateVariable    *AddStateVariableCommand    `json:"addStateVariable,omitempty"`
	RemoveStateVariable *RemoveStateVariableCommand `json:"removeStateVariable,omitempty"`
	AddNode             *AddNodeCommand             `json:"addNode,omitempty"`
	RemoveNode          *NodeCommand                `json:"removeNode,omitempty"`
	MoveNode            *MoveNodeCommand            `json:"moveNode,omitempty"`
	SetNodeLabel        *SetNodeLabelCommand        `json:"setNodeLabel,omitempty"`
	SetNodeDisabled     *SetNodeDisabledCommand     `json:"setNodeDisabled,omitempty"`
	SetConfig           *SetConfigCommand           `json:"setConfig,omitempty"`
	ClearConfig         *FieldCommand               `json:"clearConfig,omitempty"`
	BindValue           *BindValueCommand           `json:"bindValue,omitempty"`
	BindDefault         *PortCommand                `json:"bindDefault,omitempty"`
	BindBlob            *BindBlobCommand            `json:"bindBlob,omitempty"`
	ClearBinding        *PortCommand                `json:"clearBinding,omitempty"`
	Connect             *EdgeCommand                `json:"connect,omitempty"`
	Disconnect          *EdgeCommand                `json:"disconnect,omitempty"`
}

func (c Command) Validate() error { return validateTaggedCommand(c) }

type RenameWorkflowCommand struct {
	Name string `json:"name"`
}

type AddStateVariableCommand struct {
	Name    string                  `json:"name"`
	Type    datatype.TypeExpression `json:"type"`
	Default any                     `json:"default"`
}

type RemoveStateVariableCommand struct {
	Name string `json:"name"`
}

type AddNodeCommand struct {
	GraphID    string          `json:"graphId"`
	NodeTypeID string          `json:"nodeTypeId"`
	Handle     string          `json:"handle,omitempty"`
	Position   schema.Position `json:"position"`
}

type NodeCommand struct {
	GraphID string `json:"graphId"`
	NodeID  string `json:"nodeId"`
}

type MoveNodeCommand struct {
	GraphID  string          `json:"graphId"`
	NodeID   string          `json:"nodeId"`
	Position schema.Position `json:"position"`
}

type SetNodeLabelCommand struct {
	GraphID string `json:"graphId"`
	NodeID  string `json:"nodeId"`
	Label   string `json:"label"`
}

type SetNodeDisabledCommand struct {
	GraphID  string `json:"graphId"`
	NodeID   string `json:"nodeId"`
	Disabled bool   `json:"disabled"`
}

type SetConfigCommand struct {
	GraphID string `json:"graphId"`
	NodeID  string `json:"nodeId"`
	FieldID string `json:"fieldId"`
	Value   any    `json:"value"`
}

type FieldCommand struct {
	GraphID string `json:"graphId"`
	NodeID  string `json:"nodeId"`
	FieldID string `json:"fieldId"`
}

type BindValueCommand struct {
	GraphID string `json:"graphId"`
	NodeID  string `json:"nodeId"`
	PortID  string `json:"portId"`
	Value   any    `json:"value"`
}

type PortCommand struct {
	GraphID string `json:"graphId"`
	NodeID  string `json:"nodeId"`
	PortID  string `json:"portId"`
}

type BindBlobCommand struct {
	GraphID string       `json:"graphId"`
	NodeID  string       `json:"nodeId"`
	PortID  string       `json:"portId"`
	Blob    blob.BlobRef `json:"blob"`
}

type EdgeCommand struct {
	GraphID string      `json:"graphId"`
	Edge    schema.Edge `json:"edge"`
}

type GeneratedNode struct {
	CommandIndex int    `json:"commandIndex"`
	Handle       string `json:"handle,omitempty"`
	NodeID       string `json:"nodeId"`
}

type Result struct {
	Source         schema.WorkflowSource
	Artifact       []byte
	GeneratedNodes []GeneratedNode
}

type PatchError struct {
	CommandIndex int    `json:"commandIndex"`
	Code         string `json:"code"`
	Message      string `json:"message"`
}

func (e *PatchError) Error() string {
	if e.CommandIndex < 0 {
		return e.Code + ": " + e.Message
	}
	return fmt.Sprintf("command %d: %s: %s", e.CommandIndex, e.Code, e.Message)
}

type IDFactory func() string

type Engine struct {
	catalog    nodecatalog.Snapshot
	projection nodeauthoring.Snapshot
	newID      IDFactory
}

func New(catalog nodecatalog.Snapshot, projection nodeauthoring.Snapshot, newID IDFactory) (*Engine, error) {
	if !catalog.Valid() || !projection.Valid() || projection.CatalogHash() != catalog.Hash() {
		return nil, errors.New("authoring engine requires matching trusted Catalog and projection")
	}
	if newID == nil {
		newID = uuid.NewString
	}
	return &Engine{catalog: catalog, projection: projection, newID: newID}, nil
}

func (e *Engine) Apply(source schema.WorkflowSource, commands []Command) (Result, error) {
	if len(commands) == 0 || len(commands) > MaxCommands {
		return Result{}, patchError(-1, "COMMAND_BUDGET", "patch must contain between 1 and 256 commands")
	}
	working, err := cloneSource(source)
	if err != nil {
		return Result{}, err
	}
	handles := make(map[string]string)
	generated := make([]GeneratedNode, 0)
	for index := range commands {
		if err := e.applyCommand(&working, commands[index], index, handles, &generated); err != nil {
			return Result{}, err
		}
	}
	if working.Revision >= schema.MaxRevision {
		return Result{}, patchError(-1, "REVISION_EXHAUSTED", "workflow revision reached its maximum")
	}
	working.Revision++
	raw, err := artifact.Marshal(working)
	if err != nil {
		return Result{}, fmt.Errorf("encode patched Workflow Source: %w", err)
	}
	canonical, diagnostics := schema.ParseSource(raw)
	if len(diagnostics) != 0 {
		return Result{}, patchError(-1, "INVALID_RESULT", diagnostics[0].Code)
	}
	raw, err = artifact.Marshal(canonical)
	if err != nil {
		return Result{}, fmt.Errorf("encode canonical patched Workflow Source: %w", err)
	}
	return Result{Source: canonical, Artifact: raw, GeneratedNodes: generated}, nil
}

func (e *Engine) applyCommand(source *schema.WorkflowSource, command Command, index int, handles map[string]string, generated *[]GeneratedNode) error {
	if err := validateTaggedCommand(command); err != nil {
		return patchError(index, "INVALID_COMMAND", err.Error())
	}
	switch command.Kind {
	case CommandRenameWorkflow:
		name := strings.TrimSpace(command.RenameWorkflow.Name)
		if name == "" || len(name) > 256 {
			return patchError(index, "INVALID_NAME", "workflow name must contain 1 to 256 bytes")
		}
		source.Workflow.Name = name
	case CommandAddStateVariable:
		payload := command.AddStateVariable
		if hasStateVariable(*source, payload.Name) {
			return patchError(index, "DUPLICATE_STATE", "state variable already exists")
		}
		if err := payload.Type.Validate(); err != nil {
			return patchError(index, "INVALID_STATE_TYPE", err.Error())
		}
		value, err := rawValue(payload.Default)
		if err != nil {
			return patchError(index, "INVALID_STATE_DEFAULT", err.Error())
		}
		source.Variables = append(source.Variables, schema.Variable{Name: payload.Name, Type: payload.Type, Default: value})
	case CommandRemoveStateVariable:
		name := command.RemoveStateVariable.Name
		if !hasStateVariable(*source, name) {
			return patchError(index, "UNKNOWN_STATE", "state variable does not exist")
		}
		if e.stateVariableReferenced(*source, name) {
			return patchError(index, "REFERENCE_IN_USE", "state variable is still referenced by a node")
		}
		for variableIndex := range source.Variables {
			if source.Variables[variableIndex].Name == name {
				source.Variables = append(source.Variables[:variableIndex], source.Variables[variableIndex+1:]...)
				break
			}
		}
	case CommandAddNode:
		payload := command.AddNode
		graph, err := graphByID(source, payload.GraphID)
		if err != nil {
			return patchError(index, "UNKNOWN_GRAPH", err.Error())
		}
		projection, ok := e.projection.Node(payload.NodeTypeID)
		if !ok {
			return patchError(index, "UNKNOWN_NODE_TYPE", "node type is not in the admitted Catalog")
		}
		if !finitePosition(payload.Position) {
			return patchError(index, "INVALID_POSITION", "node position must be finite")
		}
		if payload.Handle != "" {
			if !handlePattern.MatchString(payload.Handle) {
				return patchError(index, "INVALID_HANDLE", "node handle must match the authoring handle grammar")
			}
			if _, exists := handles[payload.Handle]; exists {
				return patchError(index, "DUPLICATE_HANDLE", "node handle already exists in this patch")
			}
		}
		nodeID := e.newID()
		if nodeID == "" || nodeByID(graph, nodeID) != nil {
			return patchError(index, "INVALID_GENERATED_ID", "host generated an invalid or duplicate node ID")
		}
		config := make(map[string]any)
		for _, field := range projection.ConfigFields {
			if field.HasDefault {
				value, decodeErr := decodeRaw(field.Default)
				if decodeErr != nil {
					return patchError(index, "INVALID_CATALOG_DEFAULT", decodeErr.Error())
				}
				config[field.ID] = value
			}
		}
		bindings := make(map[string]schema.InputBinding)
		for _, port := range projection.DataInputs {
			if port.HasDefault {
				bindings[port.ID] = schema.InputBinding{Kind: schema.BindingDefault}
			}
		}
		graph.Nodes = append(graph.Nodes, schema.Node{
			ID: nodeID, NodeRef: projection.NodeRef, Position: payload.Position, Config: config, Bindings: bindings,
		})
		if payload.Handle != "" {
			handles[payload.Handle] = nodeID
		}
		*generated = append(*generated, GeneratedNode{CommandIndex: index, Handle: payload.Handle, NodeID: nodeID})
	case CommandRemoveNode:
		graph, node, err := resolveNode(source, command.RemoveNode.GraphID, command.RemoveNode.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		removeNode(graph, node.ID)
	case CommandMoveNode:
		if !finitePosition(command.MoveNode.Position) {
			return patchError(index, "INVALID_POSITION", "node position must be finite")
		}
		_, node, err := resolveNode(source, command.MoveNode.GraphID, command.MoveNode.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		node.Position = command.MoveNode.Position
	case CommandSetNodeLabel:
		_, node, err := resolveNode(source, command.SetNodeLabel.GraphID, command.SetNodeLabel.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		if len(command.SetNodeLabel.Label) > 1024 {
			return patchError(index, "INVALID_LABEL", "node label exceeds 1024 bytes")
		}
		node.Label = command.SetNodeLabel.Label
	case CommandSetNodeDisabled:
		_, node, err := resolveNode(source, command.SetNodeDisabled.GraphID, command.SetNodeDisabled.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		node.Disabled = command.SetNodeDisabled.Disabled
	case CommandSetConfig:
		_, node, err := resolveNode(source, command.SetConfig.GraphID, command.SetConfig.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		if _, ok := configField(e.projection, *node, command.SetConfig.FieldID); !ok {
			return patchError(index, "UNKNOWN_CONFIG_FIELD", "config field is not declared by the Node Contract")
		}
		value, err := cloneValue(command.SetConfig.Value)
		if err != nil {
			return patchError(index, "INVALID_CONFIG_VALUE", err.Error())
		}
		node.Config[command.SetConfig.FieldID] = value
	case CommandClearConfig:
		_, node, err := resolveNode(source, command.ClearConfig.GraphID, command.ClearConfig.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		field, ok := configField(e.projection, *node, command.ClearConfig.FieldID)
		if !ok {
			return patchError(index, "UNKNOWN_CONFIG_FIELD", "config field is not declared by the Node Contract")
		}
		if field.Required && !field.HasDefault {
			return patchError(index, "REQUIRED_CONFIG_FIELD", "required config field cannot be cleared")
		}
		if field.HasDefault {
			value, decodeErr := decodeRaw(field.Default)
			if decodeErr != nil {
				return patchError(index, "INVALID_CATALOG_DEFAULT", decodeErr.Error())
			}
			node.Config[field.ID] = value
		} else {
			delete(node.Config, field.ID)
		}
	case CommandBindValue:
		graph, node, err := resolveNode(source, command.BindValue.GraphID, command.BindValue.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		if _, ok := dataInput(e.projection, *node, command.BindValue.PortID); !ok {
			return patchError(index, "UNKNOWN_DATA_INPUT", "data input is not declared by the Node Contract")
		}
		value, err := rawValue(command.BindValue.Value)
		if err != nil {
			return patchError(index, "INVALID_BINDING_VALUE", err.Error())
		}
		removeIncomingData(graph, node.ID, command.BindValue.PortID)
		node.Bindings[command.BindValue.PortID] = schema.InputBinding{Kind: schema.BindingValue, Value: value}
	case CommandBindDefault:
		graph, node, err := resolveNode(source, command.BindDefault.GraphID, command.BindDefault.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		port, ok := dataInput(e.projection, *node, command.BindDefault.PortID)
		if !ok || !port.HasDefault {
			return patchError(index, "DEFAULT_UNAVAILABLE", "data input has no declared default")
		}
		removeIncomingData(graph, node.ID, command.BindDefault.PortID)
		node.Bindings[command.BindDefault.PortID] = schema.InputBinding{Kind: schema.BindingDefault}
	case CommandBindBlob:
		graph, node, err := resolveNode(source, command.BindBlob.GraphID, command.BindBlob.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		if _, ok := dataInput(e.projection, *node, command.BindBlob.PortID); !ok {
			return patchError(index, "UNKNOWN_DATA_INPUT", "data input is not declared by the Node Contract")
		}
		if err := command.BindBlob.Blob.Validate(); err != nil {
			return patchError(index, "INVALID_BLOB_REF", err.Error())
		}
		blobRef := command.BindBlob.Blob
		removeIncomingData(graph, node.ID, command.BindBlob.PortID)
		node.Bindings[command.BindBlob.PortID] = schema.InputBinding{Kind: schema.BindingBlob, Blob: &blobRef}
	case CommandClearBinding:
		_, node, err := resolveNode(source, command.ClearBinding.GraphID, command.ClearBinding.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_NODE", err.Error())
		}
		if _, ok := dataInput(e.projection, *node, command.ClearBinding.PortID); !ok {
			return patchError(index, "UNKNOWN_DATA_INPUT", "data input is not declared by the Node Contract")
		}
		delete(node.Bindings, command.ClearBinding.PortID)
	case CommandConnect:
		payload := command.Connect
		graph, err := graphByID(source, payload.GraphID)
		if err != nil {
			return patchError(index, "UNKNOWN_GRAPH", err.Error())
		}
		edge := payload.Edge
		edge.From.NodeID, err = resolveNodeReference(edge.From.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_HANDLE", err.Error())
		}
		edge.To.NodeID, err = resolveNodeReference(edge.To.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_HANDLE", err.Error())
		}
		from, to := nodeByID(graph, edge.From.NodeID), nodeByID(graph, edge.To.NodeID)
		if from == nil || to == nil {
			return patchError(index, "UNKNOWN_NODE", "edge endpoint node does not exist")
		}
		if !edgePortsMatch(e.projection, *from, *to, edge) {
			return patchError(index, "INVALID_EDGE", "edge channel, direction, or port does not match the Node Contracts")
		}
		if edgeExists(*graph, edge) {
			return patchError(index, "DUPLICATE_EDGE", "edge already exists")
		}
		if edge.Channel == schema.EdgeData {
			delete(to.Bindings, edge.To.PortID)
			removeIncomingData(graph, edge.To.NodeID, edge.To.PortID)
		}
		graph.Edges = append(graph.Edges, edge)
	case CommandDisconnect:
		payload := command.Disconnect
		graph, err := graphByID(source, payload.GraphID)
		if err != nil {
			return patchError(index, "UNKNOWN_GRAPH", err.Error())
		}
		edge := payload.Edge
		edge.From.NodeID, err = resolveNodeReference(edge.From.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_HANDLE", err.Error())
		}
		edge.To.NodeID, err = resolveNodeReference(edge.To.NodeID, handles)
		if err != nil {
			return patchError(index, "UNKNOWN_HANDLE", err.Error())
		}
		found := false
		for edgeIndex := range graph.Edges {
			if graph.Edges[edgeIndex] == edge {
				graph.Edges = append(graph.Edges[:edgeIndex], graph.Edges[edgeIndex+1:]...)
				found = true
				break
			}
		}
		if !found {
			return patchError(index, "UNKNOWN_EDGE", "edge does not exist")
		}
	default:
		return patchError(index, "INVALID_COMMAND", "unknown command kind")
	}
	return nil
}

func validateTaggedCommand(command Command) error {
	payloads := []bool{
		command.RenameWorkflow != nil, command.AddStateVariable != nil, command.RemoveStateVariable != nil,
		command.AddNode != nil, command.RemoveNode != nil, command.MoveNode != nil, command.SetNodeLabel != nil,
		command.SetNodeDisabled != nil, command.SetConfig != nil, command.ClearConfig != nil,
		command.BindValue != nil, command.BindDefault != nil, command.BindBlob != nil, command.ClearBinding != nil,
		command.Connect != nil, command.Disconnect != nil,
	}
	count := 0
	for _, present := range payloads {
		if present {
			count++
		}
	}
	if count != 1 {
		return errors.New("command must contain exactly one payload")
	}
	matches := map[CommandKind]bool{
		CommandRenameWorkflow: command.RenameWorkflow != nil, CommandAddStateVariable: command.AddStateVariable != nil,
		CommandRemoveStateVariable: command.RemoveStateVariable != nil, CommandAddNode: command.AddNode != nil,
		CommandRemoveNode: command.RemoveNode != nil, CommandMoveNode: command.MoveNode != nil,
		CommandSetNodeLabel: command.SetNodeLabel != nil, CommandSetNodeDisabled: command.SetNodeDisabled != nil,
		CommandSetConfig: command.SetConfig != nil, CommandClearConfig: command.ClearConfig != nil,
		CommandBindValue: command.BindValue != nil, CommandBindDefault: command.BindDefault != nil,
		CommandBindBlob: command.BindBlob != nil, CommandClearBinding: command.ClearBinding != nil,
		CommandConnect: command.Connect != nil, CommandDisconnect: command.Disconnect != nil,
	}
	if !matches[command.Kind] {
		return errors.New("command kind does not match its payload")
	}
	return nil
}

func cloneSource(source schema.WorkflowSource) (schema.WorkflowSource, error) {
	raw, err := artifact.Marshal(source)
	if err != nil {
		return schema.WorkflowSource{}, err
	}
	clone, diagnostics := schema.ParseSource(raw)
	if len(diagnostics) != 0 {
		return schema.WorkflowSource{}, patchError(-1, "INVALID_SOURCE", diagnostics[0].Code)
	}
	return clone, nil
}

func cloneValue(value any) (any, error) {
	raw, err := artifact.Marshal(value)
	if err != nil {
		return nil, err
	}
	return decodeRaw(raw)
}

func rawValue(value any) (json.RawMessage, error) {
	raw, err := artifact.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func decodeRaw(raw []byte) (any, error) {
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func patchError(index int, code, message string) error {
	return &PatchError{CommandIndex: index, Code: code, Message: message}
}

func graphByID(source *schema.WorkflowSource, graphID string) (*schema.Graph, error) {
	for index := range source.Graphs {
		if source.Graphs[index].ID == graphID {
			return &source.Graphs[index], nil
		}
	}
	return nil, fmt.Errorf("graph %q does not exist", graphID)
}

func resolveNode(source *schema.WorkflowSource, graphID, reference string, handles map[string]string) (*schema.Graph, *schema.Node, error) {
	graph, err := graphByID(source, graphID)
	if err != nil {
		return nil, nil, err
	}
	nodeID, err := resolveNodeReference(reference, handles)
	if err != nil {
		return nil, nil, err
	}
	node := nodeByID(graph, nodeID)
	if node == nil {
		return nil, nil, fmt.Errorf("node %q does not exist in graph %q", nodeID, graphID)
	}
	return graph, node, nil
}

func resolveNodeReference(reference string, handles map[string]string) (string, error) {
	if !strings.HasPrefix(reference, "$") {
		return reference, nil
	}
	nodeID, ok := handles[strings.TrimPrefix(reference, "$")]
	if !ok {
		return "", fmt.Errorf("node handle %q has not been generated in this patch", reference)
	}
	return nodeID, nil
}

func nodeByID(graph *schema.Graph, nodeID string) *schema.Node {
	for index := range graph.Nodes {
		if graph.Nodes[index].ID == nodeID {
			return &graph.Nodes[index]
		}
	}
	return nil
}

func removeNode(graph *schema.Graph, nodeID string) {
	for index := range graph.Nodes {
		if graph.Nodes[index].ID == nodeID {
			graph.Nodes = append(graph.Nodes[:index], graph.Nodes[index+1:]...)
			break
		}
	}
	edges := graph.Edges[:0]
	for _, edge := range graph.Edges {
		if edge.From.NodeID != nodeID && edge.To.NodeID != nodeID {
			edges = append(edges, edge)
		}
	}
	graph.Edges = edges
}

func hasStateVariable(source schema.WorkflowSource, name string) bool {
	for _, variable := range source.Variables {
		if variable.Name == name {
			return true
		}
	}
	return false
}

func (e *Engine) stateVariableReferenced(source schema.WorkflowSource, name string) bool {
	for _, graph := range source.Graphs {
		for _, node := range graph.Nodes {
			projection, ok := e.projection.Node(node.NodeRef.NodeTypeID)
			if !ok || projection.NodeRef != node.NodeRef {
				continue
			}
			for _, access := range projection.StateAccesses {
				if selected, ok := node.Config[access.SlotConfigKey].(string); ok && selected == name {
					return true
				}
			}
		}
	}
	return false
}

func configField(projection nodeauthoring.Snapshot, node schema.Node, fieldID string) (nodeauthoring.FieldProjection, bool) {
	projected, ok := projection.Node(node.NodeRef.NodeTypeID)
	if !ok || projected.NodeRef != node.NodeRef {
		return nodeauthoring.FieldProjection{}, false
	}
	for _, field := range projected.ConfigFields {
		if field.ID == fieldID {
			return field, true
		}
	}
	return nodeauthoring.FieldProjection{}, false
}

func dataInput(projection nodeauthoring.Snapshot, node schema.Node, portID string) (nodeauthoring.PortProjection, bool) {
	projected, ok := projection.Node(node.NodeRef.NodeTypeID)
	if !ok || projected.NodeRef != node.NodeRef {
		return nodeauthoring.PortProjection{}, false
	}
	for _, port := range projected.DataInputs {
		if port.ID == portID {
			return port, true
		}
	}
	return nodeauthoring.PortProjection{}, false
}

func edgePortsMatch(projection nodeauthoring.Snapshot, from, to schema.Node, edge schema.Edge) bool {
	fromProjection, fromOK := projection.Node(from.NodeRef.NodeTypeID)
	toProjection, toOK := projection.Node(to.NodeRef.NodeTypeID)
	if !fromOK || !toOK || fromProjection.NodeRef != from.NodeRef || toProjection.NodeRef != to.NodeRef {
		return false
	}
	switch edge.Channel {
	case schema.EdgeData:
		fromOK, toOK = false, false
		for _, port := range fromProjection.DataOutputs {
			fromOK = fromOK || port.ID == edge.From.PortID
		}
		for _, port := range toProjection.DataInputs {
			toOK = toOK || port.ID == edge.To.PortID
		}
		return fromOK && toOK
	case schema.EdgeExec, schema.EdgeError:
		for _, signal := range fromProjection.Signals {
			fromOK = fromOK || (signal.ID == edge.From.PortID && signal.Channel == string(edge.Channel) && signal.Direction == "output")
		}
		for _, signal := range toProjection.Signals {
			toOK = toOK || (signal.ID == edge.To.PortID && signal.Direction == "input")
		}
		return fromOK && toOK && toProjection.Instruction.AcceptsSignalInput(string(edge.Channel), edge.To.PortID)
	default:
		return false
	}
}

func edgeExists(graph schema.Graph, candidate schema.Edge) bool {
	for _, edge := range graph.Edges {
		if edge == candidate {
			return true
		}
	}
	return false
}

func removeIncomingData(graph *schema.Graph, nodeID, portID string) {
	edges := graph.Edges[:0]
	for _, edge := range graph.Edges {
		if edge.Channel != schema.EdgeData || edge.To.NodeID != nodeID || edge.To.PortID != portID {
			edges = append(edges, edge)
		}
	}
	graph.Edges = edges
}

func finitePosition(position schema.Position) bool {
	return !math.IsNaN(position.X) && !math.IsInf(position.X, 0) && !math.IsNaN(position.Y) && !math.IsInf(position.Y, 0)
}
