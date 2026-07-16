// Package aiauthoring owns the bounded AI workflow proposal and human review
// boundary. Model tools can inspect and prepare an exact candidate, but only a
// later explicit Accept call can publish that application-sealed artifact.
package aiauthoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/ai"
	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

const (
	maxInstructionBytes = 64 << 10
	maxTraceEvents      = 256
)

var (
	ErrReviewNotFound = errors.New("AI authoring review not found")
	ErrReviewTerminal = errors.New("AI authoring review is already terminal")
	ErrNoProposal     = errors.New("AI authoring completed without an exact patch proposal")
)

type Runtime struct {
	Profile    ai.ModelProfile
	Provider   ai.AgentProvider
	Credential string
}

type ProposeRequest struct {
	WorkflowID   string `json:"workflowId"`
	BaseRevision int64  `json:"baseRevision"`
	Instruction  string `json:"instruction"`
	TrustClass   string `json:"trustClass"`
}

type InputFingerprint struct {
	TrustClass string          `json:"trustClass"`
	Digest     artifact.Digest `json:"digest"`
	Bytes      int             `json:"bytes"`
}

type Change struct {
	Index     int    `json:"index"`
	Kind      string `json:"kind"`
	Target    string `json:"target"`
	Sensitive bool   `json:"sensitive"`
}

type CapabilityChange struct {
	CapabilityID   string   `json:"capabilityId"`
	Operations     []string `json:"operations"`
	TargetSlot     string   `json:"targetSlot"`
	CredentialSlot string   `json:"credentialSlot,omitempty"`
}

type PermissionDelta struct {
	Added   []CapabilityChange `json:"added"`
	Removed []CapabilityChange `json:"removed"`
}

type TraceEvent struct {
	Sequence          int               `json:"sequence"`
	Kind              string            `json:"kind"`
	OccurredAt        string            `json:"occurredAt"`
	ProviderRequestID string            `json:"providerRequestId,omitempty"`
	Facts             map[string]string `json:"facts"`
}

type ReviewStatus string

const (
	StatusProposed ReviewStatus = "proposed"
	StatusAccepted ReviewStatus = "accepted"
	StatusRejected ReviewStatus = "rejected"
	StatusStale    ReviewStatus = "stale"
)

type Review struct {
	ReviewID       string              `json:"reviewId"`
	Status         ReviewStatus        `json:"status"`
	WorkflowID     string              `json:"workflowId"`
	BaseRevision   int64               `json:"baseRevision"`
	NewRevision    int64               `json:"newRevision"`
	BaseHash       artifact.Digest     `json:"baseHash"`
	CandidateHash  artifact.Digest     `json:"candidateHash"`
	Input          InputFingerprint    `json:"input"`
	ProfileSubject artifact.Digest     `json:"profileSubject"`
	PromptManifest artifact.Digest     `json:"promptManifest"`
	ToolSet        artifact.Digest     `json:"toolSet"`
	Summary        string              `json:"summary"`
	Changes        []Change            `json:"changes"`
	Diagnostics    []schema.Diagnostic `json:"diagnostics"`
	Permissions    PermissionDelta     `json:"permissions"`
	Risks          []string            `json:"risks"`
	Usage          ai.BudgetUsage      `json:"usage"`
	Trace          []TraceEvent        `json:"trace"`
}

type reviewState struct {
	review   Review
	prepared app31.PreparedPatch
}

type Manager struct {
	application *app31.Application
	projection  nodeauthoring.Snapshot
	prompt      ai.PromptManifest
	tools       ai.ToolSet
	now         func() time.Time
	mu          sync.RWMutex
	reviews     map[string]*reviewState
}

func NewManager(application *app31.Application, builtins nodes31.Builtins, now func() time.Time) (*Manager, error) {
	if application == nil || !builtins.AIAuthoringPrompt.Valid() || !builtins.AIAuthoringToolSet.Valid() {
		return nil, errors.New("AI authoring manager requires Application and trusted authoring artifacts")
	}
	projection := application.AuthoringProjection()
	if !projection.Valid() {
		return nil, errors.New("AI authoring manager requires trusted Authoring Projection")
	}
	if now == nil {
		now = time.Now
	}
	return &Manager{application: application, projection: projection, prompt: builtins.AIAuthoringPrompt, tools: builtins.AIAuthoringToolSet, now: now, reviews: make(map[string]*reviewState)}, nil
}

func (m *Manager) Propose(ctx context.Context, runtime Runtime, request ProposeRequest) (Review, error) {
	if ctx == nil || !runtime.Profile.Valid() || runtime.Provider == nil || runtime.Credential == "" {
		return Review{}, errors.New("AI authoring runtime is unavailable")
	}
	request.Instruction = strings.TrimSpace(request.Instruction)
	request.TrustClass = strings.TrimSpace(request.TrustClass)
	if request.WorkflowID == "" || request.BaseRevision < 0 || request.Instruction == "" || len(request.Instruction) > maxInstructionBytes || request.TrustClass == "" {
		return Review{}, errors.New("invalid AI authoring request")
	}
	base, err := m.application.GetSource(request.WorkflowID)
	if err != nil {
		return Review{}, err
	}
	if base.Revision() != request.BaseRevision {
		return Review{}, workflowstore.ErrSourceConflict
	}
	reviewID, err := runid.New()
	if err != nil {
		return Review{}, err
	}
	inputDigest, err := artifact.Sum("yotta/ai-authoring-input/v1", []byte(request.Instruction))
	if err != nil {
		return Review{}, err
	}
	state := &proposalState{manager: m, workflowID: request.WorkflowID, baseRevision: request.BaseRevision, basePlan: []capability.PlanEntry{}}
	if preview, previewErr := m.application.PreviewRun(ctx, request.WorkflowID); previewErr == nil {
		state.basePlan = preview.CapabilityPlan
	}
	executor, err := state.executor()
	if err != nil {
		return Review{}, err
	}
	toolArtifact, err := ai.ResolveToolSet(m.tools)
	if err != nil {
		return Review{}, err
	}
	profileDraft := runtime.Profile.Machine()
	profileSubject, err := ai.EvaluationSubjectDigest(runtime.Profile)
	if err != nil {
		return Review{}, err
	}
	review := Review{
		ReviewID: reviewID, Status: StatusProposed, WorkflowID: request.WorkflowID,
		BaseRevision: request.BaseRevision, NewRevision: request.BaseRevision + 1, BaseHash: base.Hash(),
		Input:          InputFingerprint{TrustClass: request.TrustClass, Digest: inputDigest, Bytes: len(request.Instruction)},
		ProfileSubject: profileSubject, PromptManifest: m.prompt.Digest(), ToolSet: m.tools.Digest(),
		Changes: []Change{}, Diagnostics: []schema.Diagnostic{},
		Permissions: PermissionDelta{Added: []CapabilityChange{}, Removed: []CapabilityChange{}}, Risks: []string{}, Trace: []TraceEvent{},
	}
	state.trace = &review.Trace
	state.addTrace("model", "", map[string]string{"provider": string(profileDraft.Provider), "model": profileDraft.Model, "profile_subject": profileSubject.String()})
	state.addTrace("prompt", "", map[string]string{"manifest": m.prompt.Digest().String(), "input_digest": inputDigest.String(), "input_bytes": fmt.Sprint(len(request.Instruction)), "trust_class": request.TrustClass})
	state.addTrace("tool-authority", "", map[string]string{"tool_set": m.tools.Digest().String(), "authority": "pure", "approval": "host-builtin"})
	contextBlock, _ := artifact.Marshal(map[string]any{"workflowId": request.WorkflowID, "baseRevision": request.BaseRevision})
	rendered, err := ai.RenderPrompt(m.prompt, []ai.PromptBlock{{Kind: ai.PromptBlockUser, Content: request.Instruction}, {Kind: ai.PromptBlockContext, Content: string(contextBlock)}})
	if err != nil {
		return Review{}, err
	}
	budget := ai.RunBudget{MaxInputTokens: 250_000, MaxOutputTokens: 64_000, MaxCostMicrounits: 50_000_000, MaxWallTimeMillis: 120_000, MaxIterations: 12, MaxToolCalls: 48, MaxParallelism: 4}
	tracker, err := ai.NewBudgetTracker(budget, m.now())
	if err != nil {
		return Review{}, err
	}
	maximum := int64(8_192)
	attemptID, err := runid.New()
	if err != nil {
		return Review{}, err
	}
	start := ai.AgentStartRequest{AttemptID: attemptID, Prompt: rendered, ToolSet: toolArtifact, Limits: ai.GenerationLimits{MaxOutputTokens: &maximum}, MaxParallelism: budget.MaxParallelism, Retention: ai.RetentionNoApplicationState}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(budget.MaxWallTimeMillis)*time.Millisecond)
	defer cancel()
	if err := tracker.BeforeTurn(m.now()); err != nil {
		return Review{}, err
	}
	outcome, providerState, err := runtime.Provider.StartAgent(runCtx, runtime.Credential, start)
	if err != nil {
		return Review{}, err
	}
	for {
		state.addTrace("provider-response", outcome.ProviderRequestID, map[string]string{
			"response_id": outcome.ProviderResponseID, "finish": string(outcome.Finish.Kind),
			"turn": fmt.Sprint(tracker.Usage().Iterations + 1),
		})
		calls, callErr := toolCalls(outcome)
		if callErr != nil {
			return Review{}, callErr
		}
		if err := tracker.ConsumeTurn(m.now(), outcome, len(calls)); err != nil {
			return Review{}, err
		}
		if outcome.Finish.Kind == ai.FinishCompleted {
			if !state.prepared.Valid() || !state.compileChecked || !state.previewChecked {
				return Review{}, ErrNoProposal
			}
			review.Summary = outcomeText(outcome)
			break
		}
		if outcome.Finish.Kind != ai.FinishToolCalls || len(calls) == 0 || providerState == nil {
			return Review{}, fmt.Errorf("AI authoring finished as %s", outcome.Finish.Kind)
		}
		results, executeErr := executor.Execute(runCtx, calls, budget.MaxParallelism)
		if executeErr != nil {
			return Review{}, executeErr
		}
		continuationID, idErr := runid.New()
		if idErr != nil {
			return Review{}, idErr
		}
		if err := tracker.BeforeTurn(m.now()); err != nil {
			return Review{}, err
		}
		outcome, providerState, err = runtime.Provider.ContinueAgent(runCtx, runtime.Credential, providerState, ai.AgentContinueRequest{AttemptID: continuationID, Results: results})
		if err != nil {
			return Review{}, err
		}
	}
	review.CandidateHash = state.prepared.CandidateHash()
	review.Changes = append([]Change(nil), state.changes...)
	review.Diagnostics = append([]schema.Diagnostic(nil), state.diagnostics...)
	review.Permissions = state.permissions
	if len(review.Permissions.Added) != 0 {
		review.Risks = append(review.Risks, "permission-expansion")
	}
	if schema.HasErrors(review.Diagnostics) {
		review.Risks = append(review.Risks, "compiler-errors")
	}
	review.Usage = tracker.Usage()
	m.mu.Lock()
	m.reviews[reviewID] = &reviewState{review: cloneReview(review), prepared: state.prepared}
	m.mu.Unlock()
	return cloneReview(review), nil
}

func (m *Manager) Accept(ctx context.Context, reviewID string) (Review, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.reviews[reviewID]
	if !ok {
		return Review{}, ErrReviewNotFound
	}
	if state.review.Status != StatusProposed {
		return Review{}, ErrReviewTerminal
	}
	committed, err := m.application.CommitPreparedPatch(ctx, state.prepared)
	if err != nil {
		if errors.Is(err, workflowstore.ErrSourceConflict) {
			state.review.Status = StatusStale
			appendTrace(&state.review.Trace, m.now(), "approval", "", map[string]string{"decision": "stale", "reason": "revision-conflict"})
		}
		return cloneReview(state.review), err
	}
	state.review.Status = StatusAccepted
	state.review.NewRevision = committed.Source.Revision()
	appendTrace(&state.review.Trace, m.now(), "approval", "", map[string]string{"decision": "accepted", "source_hash": committed.Source.Hash().String()})
	return cloneReview(state.review), nil
}

func (m *Manager) Reject(reviewID string) (Review, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.reviews[reviewID]
	if !ok {
		return Review{}, ErrReviewNotFound
	}
	if state.review.Status != StatusProposed {
		return Review{}, ErrReviewTerminal
	}
	state.review.Status = StatusRejected
	appendTrace(&state.review.Trace, m.now(), "approval", "", map[string]string{"decision": "rejected"})
	return cloneReview(state.review), nil
}

func (m *Manager) Get(reviewID string) (Review, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.reviews[reviewID]
	if !ok {
		return Review{}, ErrReviewNotFound
	}
	return cloneReview(state.review), nil
}

type proposalState struct {
	manager           *Manager
	workflowID        string
	baseRevision      int64
	basePlan          []capability.PlanEntry
	candidatePlan     []capability.PlanEntry
	prepared          app31.PreparedPatch
	changes           []Change
	diagnostics       []schema.Diagnostic
	permissions       PermissionDelta
	compileChecked    bool
	previewChecked    bool
	diagnosticKey     string
	diagnosticRepeats int
	trace             *[]TraceEvent
}

func (s *proposalState) executor() (ai.ToolExecutor, error) {
	bindings := []ai.ToolBinding{
		{Name: "catalog_search", Handler: s.catalogSearch},
		{Name: "catalog_describe", Handler: s.catalogDescribe},
		{Name: "workflow_inspect", Handler: s.workflowInspect},
		{Name: "workflow_propose_patch", Handler: s.workflowProposePatch},
		{Name: "workflow_compile", Handler: s.workflowCompile},
		{Name: "workflow_preview", Handler: s.workflowPreview},
		{Name: "diagnostic_explain", Handler: s.diagnosticExplain},
	}
	return ai.NewToolExecutor(s.manager.tools, bindings)
}

func (s *proposalState) catalogSearch(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		Query string `json:"query"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, err
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	type item struct {
		NodeTypeID     string `json:"nodeTypeId"`
		TitleKey       string `json:"titleKey"`
		DescriptionKey string `json:"descriptionKey"`
	}
	items := make([]item, 0, 24)
	for _, node := range s.manager.projection.Nodes() {
		haystack := strings.ToLower(strings.Join([]string{node.NodeRef.NodeTypeID, node.TitleKey, node.DescriptionKey, strings.Join(node.Tags, " ")}, " "))
		if query != "" && !strings.Contains(haystack, query) {
			continue
		}
		items = append(items, item{node.NodeRef.NodeTypeID, node.TitleKey, node.DescriptionKey})
		if len(items) == 24 {
			break
		}
	}
	encoded, err := artifact.Marshal(items)
	if err != nil {
		return nil, err
	}
	s.addToolTrace("catalog_search", raw, encoded)
	return json.Marshal(map[string]string{"itemsJson": string(encoded)})
}

func (s *proposalState) catalogDescribe(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		NodeTypeID string `json:"nodeTypeId"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, err
	}
	node, ok := s.manager.projection.Node(input.NodeTypeID)
	if !ok {
		return nil, fmt.Errorf("UNKNOWN_NODE_TYPE: %s", input.NodeTypeID)
	}
	encoded, err := artifact.Marshal(node)
	if err != nil {
		return nil, err
	}
	s.addToolTrace("catalog_describe", raw, encoded)
	return json.Marshal(map[string]string{"nodeJson": string(encoded)})
}

func (s *proposalState) workflowInspect(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		WorkflowID string `json:"workflowId"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, err
	}
	if input.WorkflowID != s.workflowID {
		return nil, errors.New("AI authoring cannot inspect a workflow outside the locked review")
	}
	snapshot, err := s.manager.application.GetSource(s.workflowID)
	if err != nil {
		return nil, err
	}
	if snapshot.Revision() != s.baseRevision {
		return nil, workflowstore.ErrSourceConflict
	}
	s.addToolTrace("workflow_inspect", raw, snapshot.Artifact())
	return json.Marshal(map[string]any{"revision": snapshot.Revision(), "sourceHash": snapshot.Hash().String(), "sourceJson": string(snapshot.Artifact())})
}

func (s *proposalState) workflowProposePatch(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		CommandsJSON string `json:"commandsJson"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, err
	}
	if len(input.CommandsJSON) == 0 || len(input.CommandsJSON) > ai.MaxPromptBytes {
		return nil, errors.New("AI authoring command batch exceeds byte budget")
	}
	var commands []authoring.Command
	if err := decodeExact([]byte(input.CommandsJSON), &commands); err != nil {
		return nil, fmt.Errorf("decode typed authoring commands: %w", err)
	}
	prepared, err := s.manager.application.PreparePatch(ctx, authoring.PatchRequest{WorkflowID: s.workflowID, BaseRevision: s.baseRevision, Commands: commands})
	if err != nil {
		return nil, err
	}
	s.prepared = prepared.Patch
	s.changes = normalizeChanges(commands)
	s.diagnostics = append([]schema.Diagnostic(nil), prepared.Diagnostics...)
	s.candidatePlan = append([]capability.PlanEntry(nil), prepared.CapabilityPlan...)
	s.compileChecked = false
	s.previewChecked = false
	s.permissions = permissionDelta(s.basePlan, s.candidatePlan)
	s.addToolTrace("workflow_propose_patch", raw, prepared.Patch.CandidateArtifact())
	s.addTrace("patch", "", map[string]string{"base_revision": fmt.Sprint(s.baseRevision), "new_revision": fmt.Sprint(s.baseRevision + 1), "base_hash": prepared.Patch.BaseHash().String(), "candidate_hash": prepared.Patch.CandidateHash().String(), "commands": fmt.Sprint(len(commands))})
	diagnostics, _ := artifact.Marshal(prepared.Diagnostics)
	return json.Marshal(map[string]any{"candidateHash": prepared.Patch.CandidateHash().String(), "newRevision": s.baseRevision + 1, "diagnosticsJson": string(diagnostics)})
}

func (s *proposalState) workflowCompile(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	if !s.prepared.Valid() {
		return nil, ErrNoProposal
	}
	keyParts := make([]string, 0, len(s.diagnostics))
	for _, diagnostic := range s.diagnostics {
		keyParts = append(keyParts, diagnostic.Code)
	}
	key := strings.Join(keyParts, ",")
	if key != "" && key == s.diagnosticKey {
		s.diagnosticRepeats++
	} else {
		s.diagnosticKey, s.diagnosticRepeats = key, 0
	}
	if s.diagnosticRepeats >= 2 {
		return nil, errors.New("AI authoring stopped after repeated identical compiler diagnostics")
	}
	s.compileChecked = true
	encoded, err := artifact.Marshal(s.diagnostics)
	if err != nil {
		return nil, err
	}
	s.addToolTrace("workflow_compile", raw, encoded)
	s.addTrace("compiler", "", map[string]string{"candidate_hash": s.prepared.CandidateHash().String(), "diagnostics": key, "errors": fmt.Sprint(schema.HasErrors(s.diagnostics))})
	return json.Marshal(map[string]string{"diagnosticsJson": string(encoded)})
}

func (s *proposalState) workflowPreview(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	if !s.prepared.Valid() {
		return nil, ErrNoProposal
	}
	s.previewChecked = true
	encoded, err := artifact.Marshal(s.permissions)
	if err != nil {
		return nil, err
	}
	s.addToolTrace("workflow_preview", raw, encoded)
	s.addTrace("run-preview", "", map[string]string{"admission": "not-run", "effects": "none", "added_capabilities": fmt.Sprint(len(s.permissions.Added)), "removed_capabilities": fmt.Sprint(len(s.permissions.Removed))})
	return json.Marshal(map[string]string{"deltaJson": string(encoded)})
}

func (s *proposalState) diagnosticExplain(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		Code string `json:"code"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, err
	}
	explanation, repairs := diagnosticHelp(input.Code)
	encoded, _ := artifact.Marshal(repairs)
	s.addToolTrace("diagnostic_explain", raw, encoded)
	return json.Marshal(map[string]string{"explanation": explanation, "repairsJson": string(encoded)})
}

func (s *proposalState) addToolTrace(name string, input, output []byte) {
	inDigest, _ := artifact.Sum("yotta/ai-authoring-tool-input/v1", input)
	outDigest, _ := artifact.Sum("yotta/ai-authoring-tool-output/v1", output)
	s.addTrace("tool", "", map[string]string{"name": name, "input_digest": inDigest.String(), "input_bytes": fmt.Sprint(len(input)), "output_digest": outDigest.String(), "output_bytes": fmt.Sprint(len(output))})
}

func (s *proposalState) addTrace(kind, requestID string, facts map[string]string) {
	appendTrace(s.trace, s.manager.now(), kind, requestID, facts)
}

func appendTrace(target *[]TraceEvent, now time.Time, kind, requestID string, facts map[string]string) {
	if target == nil || len(*target) >= maxTraceEvents {
		return
	}
	copyFacts := make(map[string]string, len(facts))
	for key, value := range facts {
		copyFacts[key] = value
	}
	*target = append(*target, TraceEvent{Sequence: len(*target) + 1, Kind: kind, OccurredAt: now.UTC().Format(time.RFC3339Nano), ProviderRequestID: requestID, Facts: copyFacts})
}

func toolCalls(outcome ai.Outcome) ([]ai.ToolCall, error) {
	calls := make([]ai.ToolCall, 0)
	for _, item := range outcome.Items {
		if item.Kind == ai.OutputToolCall && item.ToolCall != nil {
			calls = append(calls, *item.ToolCall)
		}
	}
	if outcome.Finish.Kind == ai.FinishToolCalls && len(calls) == 0 {
		return nil, errors.New("AI provider omitted tool calls")
	}
	return calls, nil
}

func outcomeText(outcome ai.Outcome) string {
	parts := make([]string, 0)
	for _, item := range outcome.Items {
		if item.Kind == ai.OutputText && item.Text != nil {
			parts = append(parts, strings.TrimSpace(item.Text.Text))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func decodeExact(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSON contains trailing or invalid values")
	}
	return nil
}

func normalizeChanges(commands []authoring.Command) []Change {
	result := make([]Change, 0, len(commands))
	for index, command := range commands {
		change := Change{Index: index, Kind: string(command.Kind), Target: commandTarget(command)}
		change.Sensitive = command.Kind == authoring.CommandSetConfig || command.Kind == authoring.CommandBindValue || command.Kind == authoring.CommandBindBlob
		result = append(result, change)
	}
	return result
}

func commandTarget(command authoring.Command) string {
	switch command.Kind {
	case authoring.CommandRenameWorkflow:
		return "workflow"
	case authoring.CommandAddStateVariable:
		return "state:" + command.AddStateVariable.Name
	case authoring.CommandRemoveStateVariable:
		return "state:" + command.RemoveStateVariable.Name
	case authoring.CommandAddNode:
		return command.AddNode.GraphID + "/type:" + command.AddNode.NodeTypeID
	case authoring.CommandRemoveNode:
		return command.RemoveNode.GraphID + "/" + command.RemoveNode.NodeID
	case authoring.CommandMoveNode:
		return command.MoveNode.GraphID + "/" + command.MoveNode.NodeID
	case authoring.CommandSetNodeLabel:
		return command.SetNodeLabel.GraphID + "/" + command.SetNodeLabel.NodeID
	case authoring.CommandSetNodeDisabled:
		return command.SetNodeDisabled.GraphID + "/" + command.SetNodeDisabled.NodeID
	case authoring.CommandSetConfig:
		return command.SetConfig.GraphID + "/" + command.SetConfig.NodeID + "/config:" + command.SetConfig.FieldID
	case authoring.CommandClearConfig:
		return command.ClearConfig.GraphID + "/" + command.ClearConfig.NodeID + "/config:" + command.ClearConfig.FieldID
	case authoring.CommandBindValue:
		return command.BindValue.GraphID + "/" + command.BindValue.NodeID + "/input:" + command.BindValue.PortID
	case authoring.CommandBindDefault:
		return command.BindDefault.GraphID + "/" + command.BindDefault.NodeID + "/input:" + command.BindDefault.PortID
	case authoring.CommandBindBlob:
		return command.BindBlob.GraphID + "/" + command.BindBlob.NodeID + "/input:" + command.BindBlob.PortID
	case authoring.CommandClearBinding:
		return command.ClearBinding.GraphID + "/" + command.ClearBinding.NodeID + "/input:" + command.ClearBinding.PortID
	case authoring.CommandConnect:
		return command.Connect.GraphID + "/edge"
	case authoring.CommandDisconnect:
		return command.Disconnect.GraphID + "/edge"
	default:
		return "unknown"
	}
}

func permissionDelta(before, after []capability.PlanEntry) PermissionDelta {
	type indexed struct {
		key   string
		value CapabilityChange
	}
	index := func(entries []capability.PlanEntry) map[string]indexed {
		result := make(map[string]indexed, len(entries))
		for _, entry := range entries {
			requirement := entry.Requirement
			raw, _ := artifact.Marshal(requirement)
			key := string(raw)
			result[key] = indexed{key: key, value: CapabilityChange{CapabilityID: requirement.Capability.CapabilityID, Operations: append([]string(nil), requirement.Operations...), TargetSlot: requirement.TargetSlot, CredentialSlot: requirement.CredentialSlot}}
		}
		return result
	}
	left, right := index(before), index(after)
	delta := PermissionDelta{Added: []CapabilityChange{}, Removed: []CapabilityChange{}}
	for key, item := range right {
		if _, ok := left[key]; !ok {
			delta.Added = append(delta.Added, item.value)
		}
	}
	for key, item := range left {
		if _, ok := right[key]; !ok {
			delta.Removed = append(delta.Removed, item.value)
		}
	}
	sort.Slice(delta.Added, func(i, j int) bool {
		return delta.Added[i].CapabilityID+delta.Added[i].TargetSlot < delta.Added[j].CapabilityID+delta.Added[j].TargetSlot
	})
	sort.Slice(delta.Removed, func(i, j int) bool {
		return delta.Removed[i].CapabilityID+delta.Removed[i].TargetSlot < delta.Removed[j].CapabilityID+delta.Removed[j].TargetSlot
	})
	return delta
}

func diagnosticHelp(code string) (string, []string) {
	help := map[string]struct {
		explanation string
		repairs     []string
	}{
		"UNKNOWN_NODE_TYPE":      {"The source pins a node type absent from the admitted Catalog.", []string{"Search and describe the trusted catalog, then choose an admitted node."}},
		"NODE_CONTRACT_MISMATCH": {"A node semantic digest does not match its admitted Node Contract.", []string{"Remove and re-add the node through typed authoring commands."}},
		"INVALID_CONFIG":         {"Node config violates the exact Node Contract schema.", []string{"Describe the node and set only declared fields with valid values."}},
		"UNBOUND_INPUT":          {"A required data input has no edge, value, blob, or default binding.", []string{"Connect a compatible output or bind an explicit value/default."}},
	}
	if value, ok := help[code]; ok {
		return value.explanation, value.repairs
	}
	return "No specialized explanation is registered for this stable diagnostic code.", []string{"Inspect the diagnostic path and the exact Node Contract before proposing a minimal repair."}
}

func cloneReview(source Review) Review {
	raw, err := json.Marshal(source)
	if err != nil {
		panic(err)
	}
	var clone Review
	if err := json.Unmarshal(raw, &clone); err != nil {
		panic(err)
	}
	return clone
}
