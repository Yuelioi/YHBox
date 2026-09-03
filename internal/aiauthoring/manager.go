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
	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

const (
	maxInstructionBytes  = 64 << 10
	maxTraceEvents       = 256
	maxRetainedReviews   = 128
	activeReviewTTL      = 30 * time.Minute
	terminalReviewTTL    = 10 * time.Minute
	MinMaxIterations     = 8
	DefaultMaxIterations = 24
	MaxMaxIterations     = ai.MaxAgentIterations
)

var (
	ErrReviewNotFound                = errors.New("AI authoring review not found")
	ErrReviewTerminal                = errors.New("AI authoring review is already terminal")
	ErrReviewCapacity                = errors.New("AI authoring review capacity exhausted")
	ErrNoProposal                    = errors.New("AI authoring completed without an exact patch proposal")
	ErrDiagnosticRunUnavailable      = errors.New("AI diagnostic Run is unavailable")
	ErrDiagnosticRunWorkflowMismatch = errors.New("AI diagnostic Run belongs to another Workflow")
)

type ToolInputError struct {
	Tool  string
	Cause error
}

func (e *ToolInputError) Error() string {
	return fmt.Sprintf("AI tool %s input is invalid: %v", e.Tool, e.Cause)
}
func (e *ToolInputError) Unwrap() error { return e.Cause }

type Runtime struct {
	Profile       ai.ModelProfile
	Provider      ai.AgentProvider
	Credential    string
	MaxIterations int
}

type ProposeRequest struct {
	WorkflowID      string                     `json:"workflowId"`
	RunID           string                     `json:"runId,omitempty"`
	BaseRevision    int64                      `json:"baseRevision"`
	Instruction     string                     `json:"instruction"`
	TrustClass      string                     `json:"trustClass"`
	History         []ConversationMessage      `json:"-"`
	Progress        func(ConversationProgress) `json:"-"`
	AllowAnswerOnly bool                       `json:"-"`
}

type ConversationProgress struct {
	ConversationID string            `json:"conversationId"`
	TurnID         string            `json:"turnId"`
	Kind           string            `json:"kind"`
	Facts          map[string]string `json:"facts,omitempty"`
}

type ConversationTurnRequest struct {
	ConversationID string
	WorkflowID     string
	RunID          string
	BaseRevision   int64
	Instruction    string
	TrustClass     string
	Progress       func(ConversationProgress)
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
	review     Review
	prepared   appcore.PreparedPatch
	createdAt  time.Time
	terminalAt time.Time
}

type Manager struct {
	application   *appcore.Application
	projection    nodeauthoring.Snapshot
	prompt        ai.PromptManifest
	tools         ai.ToolSet
	now           func() time.Time
	mu            sync.RWMutex
	reviews       map[string]*reviewState
	inflight      int
	conversations *ConversationStore
}

func NewManager(application *appcore.Application, builtins nodes.Builtins, now func() time.Time) (*Manager, error) {
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

func (m *Manager) AttachConversationStore(store *ConversationStore) error {
	if store == nil {
		return errors.New("AI authoring conversation store is unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conversations != nil {
		return errors.New("AI authoring conversation store is already attached")
	}
	m.conversations = store
	return nil
}

func (m *Manager) CreateConversation(workflowID string) (Conversation, error) {
	if m.conversations == nil {
		return Conversation{}, errors.New("AI authoring conversation store is unavailable")
	}
	return m.conversations.Create(workflowID)
}

func (m *Manager) ListConversations(workflowID string) ([]ConversationSummary, error) {
	if m.conversations == nil {
		return nil, errors.New("AI authoring conversation store is unavailable")
	}
	return m.conversations.List(workflowID)
}

func (m *Manager) GetConversation(workflowID, conversationID string) (Conversation, error) {
	if m.conversations == nil {
		return Conversation{}, errors.New("AI authoring conversation store is unavailable")
	}
	return m.conversations.Get(workflowID, conversationID)
}

func (m *Manager) DeleteConversation(workflowID, conversationID string) error {
	if m.conversations == nil {
		return errors.New("AI authoring conversation store is unavailable")
	}
	return m.conversations.Delete(workflowID, conversationID)
}

func (m *Manager) RecordFailedTurn(workflowID, conversationID, instruction, problemID string, problemParams map[string]any, operationID ...string) (Conversation, error) {
	if m.conversations == nil {
		return Conversation{}, errors.New("AI authoring conversation store is unavailable")
	}
	userID, err := runid.New()
	if err != nil {
		return Conversation{}, err
	}
	if _, err := m.conversations.Append(workflowID, conversationID, ConversationMessage{ID: userID, Role: "user", Content: instruction, CreatedAt: m.now().UTC()}); err != nil {
		return Conversation{}, err
	}
	return m.RecordFailure(workflowID, conversationID, problemID, problemParams, operationID...)
}

func (m *Manager) RecordFailure(workflowID, conversationID, problemID string, problemParams map[string]any, operationID ...string) (Conversation, error) {
	if m.conversations == nil {
		return Conversation{}, errors.New("AI authoring conversation store is unavailable")
	}
	responseID, err := runid.New()
	if err != nil {
		return Conversation{}, err
	}
	if problemID == "" {
		problemID = "ai.authoring.failed"
	}
	operation := ""
	if len(operationID) != 0 {
		operation = operationID[0]
	}
	return m.conversations.Append(workflowID, conversationID, ConversationMessage{ID: responseID, Role: "assistant", Content: problemID, ProblemID: problemID, ProblemParams: problemParams, OperationID: operation, CreatedAt: m.now().UTC()})
}

func (m *Manager) ProposeTurn(ctx context.Context, runtime Runtime, request ConversationTurnRequest) (Conversation, Review, error) {
	if m.conversations == nil {
		return Conversation{}, Review{}, errors.New("AI authoring conversation store is unavailable")
	}
	conversation, err := m.conversations.Get(request.WorkflowID, request.ConversationID)
	if err != nil {
		return Conversation{}, Review{}, err
	}
	messageID, err := runid.New()
	if err != nil {
		return Conversation{}, Review{}, err
	}
	prior := append([]ConversationMessage(nil), conversation.Messages...)
	conversation, err = m.conversations.Append(request.WorkflowID, request.ConversationID, ConversationMessage{ID: messageID, Role: "user", Content: request.Instruction, CreatedAt: m.now().UTC()})
	if err != nil {
		return Conversation{}, Review{}, err
	}
	turnID, err := runid.New()
	if err != nil {
		return conversation, Review{}, err
	}
	emit := func(kind string, facts map[string]string) {
		if request.Progress != nil {
			request.Progress(ConversationProgress{ConversationID: request.ConversationID, TurnID: turnID, Kind: kind, Facts: facts})
		}
	}
	emit("started", nil)
	review, err := m.Propose(ctx, runtime, ProposeRequest{
		WorkflowID: request.WorkflowID, RunID: request.RunID, BaseRevision: request.BaseRevision,
		Instruction: request.Instruction, TrustClass: request.TrustClass, History: prior, AllowAnswerOnly: true,
		Progress: func(event ConversationProgress) {
			event.ConversationID, event.TurnID = request.ConversationID, turnID
			if request.Progress != nil {
				request.Progress(event)
			}
		},
	})
	if err != nil {
		emit("failed", nil)
		return conversation, Review{}, err
	}
	responseID, err := runid.New()
	if err != nil {
		return conversation, Review{}, err
	}
	message := ConversationMessage{ID: responseID, Role: "assistant", Content: review.Summary, CreatedAt: m.now().UTC()}
	if review.ReviewID != "" {
		message.ReviewID, message.Review = review.ReviewID, &review
	}
	conversation, err = m.conversations.Append(request.WorkflowID, request.ConversationID, message)
	if err != nil {
		return conversation, Review{}, err
	}
	emit("completed", map[string]string{"reviewId": review.ReviewID})
	return conversation, review, nil
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
	if err := m.reserveReview(m.now()); err != nil {
		return Review{}, err
	}
	defer m.releaseReviewReservation()
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
	state := &proposalState{manager: m, workflowID: request.WorkflowID, runID: request.RunID, baseRevision: request.BaseRevision, baseHash: base.Hash(), basePlan: []capability.PlanEntry{}, baseDiagnostics: []schema.Diagnostic{}, progress: request.Progress}
	if request.RunID != "" {
		record, runErr := m.application.GetRun(request.RunID)
		if runErr != nil {
			return Review{}, fmt.Errorf("%w: %v", ErrDiagnosticRunUnavailable, runErr)
		}
		if record.Admission().WorkflowID != request.WorkflowID {
			return Review{}, ErrDiagnosticRunWorkflowMismatch
		}
	}
	if preview, previewErr := m.application.PreviewRun(ctx, request.WorkflowID); previewErr == nil {
		state.basePlan = preview.CapabilityPlan
		state.baseDiagnostics = append([]schema.Diagnostic(nil), preview.Diagnostics...)
	} else if len(preview.Diagnostics) != 0 {
		state.baseDiagnostics = append([]schema.Diagnostic(nil), preview.Diagnostics...)
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
	history := boundedConversationHistory(request.History)
	contextBlock, _ := artifact.Marshal(map[string]any{
		"workflowId": request.WorkflowID, "baseRevision": request.BaseRevision, "runId": request.RunID,
		"conversation": history,
	})
	rendered, err := ai.RenderPrompt(m.prompt, []ai.PromptBlock{{Kind: ai.PromptBlockUser, Content: request.Instruction}, {Kind: ai.PromptBlockContext, Content: string(contextBlock)}})
	if err != nil {
		return Review{}, err
	}
	maxIterations := runtime.MaxIterations
	if maxIterations == 0 {
		maxIterations = DefaultMaxIterations
	}
	budget := ai.RunBudget{MaxInputTokens: 250_000, MaxOutputTokens: 64_000, MaxCostMicrounits: 50_000_000, MaxWallTimeMillis: 120_000, MaxIterations: maxIterations, MaxToolCalls: 48, MaxParallelism: 4}
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
				if !request.AllowAnswerOnly {
					return Review{}, ErrNoProposal
				}
				review.ReviewID = ""
				review.Status = ""
				review.Summary = outcomeText(outcome)
				review.Usage = tracker.Usage()
				return review, nil
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
	m.reviews[reviewID] = &reviewState{review: cloneReview(review), prepared: state.prepared, createdAt: m.now()}
	m.mu.Unlock()
	return cloneReview(review), nil
}

func boundedConversationHistory(messages []ConversationMessage) []map[string]string {
	const historyBudget = 128 << 10
	result := make([]map[string]string, 0, len(messages))
	used := 0
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message.ProblemID != "" || (message.Role != "user" && message.Role != "assistant") {
			continue
		}
		if used+len(message.Content) > historyBudget {
			break
		}
		used += len(message.Content)
		result = append(result, map[string]string{"role": message.Role, "content": message.Content})
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return result
}

func (m *Manager) Accept(ctx context.Context, reviewID string) (Review, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneReviewsLocked(m.now())
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
			state.prepared = appcore.PreparedPatch{}
			state.terminalAt = m.now()
			if m.conversations != nil {
				_ = m.conversations.UpdateReview(state.review)
			}
		}
		return cloneReview(state.review), err
	}
	state.review.Status = StatusAccepted
	state.review.NewRevision = committed.Source.Revision()
	appendTrace(&state.review.Trace, m.now(), "approval", "", map[string]string{"decision": "accepted", "source_hash": committed.Source.Hash().String()})
	state.prepared = appcore.PreparedPatch{}
	state.terminalAt = m.now()
	if m.conversations != nil {
		_ = m.conversations.UpdateReview(state.review)
	}
	return cloneReview(state.review), nil
}

func (m *Manager) Reject(reviewID string) (Review, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneReviewsLocked(m.now())
	state, ok := m.reviews[reviewID]
	if !ok {
		return Review{}, ErrReviewNotFound
	}
	if state.review.Status != StatusProposed {
		return Review{}, ErrReviewTerminal
	}
	state.review.Status = StatusRejected
	appendTrace(&state.review.Trace, m.now(), "approval", "", map[string]string{"decision": "rejected"})
	state.prepared = appcore.PreparedPatch{}
	state.terminalAt = m.now()
	if m.conversations != nil {
		_ = m.conversations.UpdateReview(state.review)
	}
	return cloneReview(state.review), nil
}

func (m *Manager) Get(reviewID string) (Review, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneReviewsLocked(m.now())
	state, ok := m.reviews[reviewID]
	if !ok {
		return Review{}, ErrReviewNotFound
	}
	return cloneReview(state.review), nil
}

func (m *Manager) reserveReview(now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneReviewsLocked(now)
	if len(m.reviews)+m.inflight >= maxRetainedReviews {
		return ErrReviewCapacity
	}
	m.inflight++
	return nil
}

func (m *Manager) releaseReviewReservation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inflight > 0 {
		m.inflight--
	}
}

func (m *Manager) pruneReviewsLocked(now time.Time) {
	for reviewID, state := range m.reviews {
		expiresAt := state.createdAt.Add(activeReviewTTL)
		if !state.terminalAt.IsZero() {
			expiresAt = state.terminalAt.Add(terminalReviewTTL)
		}
		if !now.Before(expiresAt) {
			delete(m.reviews, reviewID)
		}
	}
}

type proposalState struct {
	manager           *Manager
	workflowID        string
	runID             string
	baseRevision      int64
	baseHash          artifact.Digest
	basePlan          []capability.PlanEntry
	baseDiagnostics   []schema.Diagnostic
	candidatePlan     []capability.PlanEntry
	prepared          appcore.PreparedPatch
	changes           []Change
	diagnostics       []schema.Diagnostic
	permissions       PermissionDelta
	commands          []authoring.Command
	nodeAliases       map[string]string
	compileChecked    bool
	previewChecked    bool
	diagnosticKey     string
	diagnosticRepeats int
	trace             *[]TraceEvent
	progress          func(ConversationProgress)
}

func (s *proposalState) executor() (ai.ToolExecutor, error) {
	bindings := []ai.ToolBinding{
		{Name: "catalog_search", Handler: s.catalogSearch},
		{Name: "catalog_describe", Handler: s.catalogDescribe},
		{Name: "workflow_inspect", Handler: s.workflowInspect},
		{Name: "run_get", Handler: s.runGet},
		{Name: "workflow_add_node", Handler: s.workflowAddNode},
		{Name: "workflow_connect", Handler: s.workflowConnect},
		{Name: "workflow_set_numeric_input", Handler: s.workflowSetNumericInput},
		{Name: "workflow_set_input_json", Handler: s.workflowSetInputJSON},
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

func (s *proposalState) runGet(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct{}
	if err := decodeExact(raw, &input); err != nil {
		return nil, err
	}
	if s.runID == "" {
		return nil, errors.New("AI authoring review has no diagnostic Run")
	}
	timeline, err := s.manager.application.GetRunTimelinePage(ctx, s.runID, 1, 500)
	if err != nil {
		return nil, err
	}
	record, err := s.manager.application.GetRun(s.runID)
	if err != nil {
		return nil, err
	}
	admission := record.Admission()
	timing := record.Timing()
	elapsedMilliseconds := int64(0)
	if timing.StartedAt != nil {
		end := s.manager.now().UTC()
		if timing.EndedAt != nil {
			end = *timing.EndedAt
		}
		if !end.Before(*timing.StartedAt) {
			elapsedMilliseconds = end.Sub(*timing.StartedAt).Milliseconds()
		}
	}
	evidence, err := artifact.Marshal(map[string]any{
		"runId": admission.RunID, "workflowId": admission.WorkflowID,
		"sourceHash": admission.SourceHash, "sourceRevision": admission.SourceRevision,
		"programHash": admission.ProgramHash, "status": record.Status(), "elapsedMilliseconds": elapsedMilliseconds,
		"timelinePage": timeline.Page, "timelinePages": timeline.Pages, "timelineTotal": timeline.Total,
		"timeline": timeline.Entries,
	})
	if err != nil {
		return nil, err
	}
	s.addToolTrace("run_get", raw, evidence)
	return json.Marshal(map[string]string{"evidenceJson": string(evidence)})
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
	if err := decodeAuthoringCommands([]byte(input.CommandsJSON), &commands); err != nil {
		return nil, &ToolInputError{Tool: "workflow_propose_patch", Cause: err}
	}
	return s.prepareCommands(ctx, "workflow_propose_patch", raw, commands)
}

func (s *proposalState) workflowSetNumericInput(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GraphID string  `json:"graphId"`
		NodeID  string  `json:"nodeId"`
		InputID string  `json:"inputId"`
		Value   float64 `json:"value"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, &ToolInputError{Tool: "workflow_set_numeric_input", Cause: err}
	}
	command := authoring.Command{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{
		GraphID: input.GraphID, NodeID: input.NodeID, PortID: input.InputID, Value: input.Value,
	}}
	return s.prepareCommands(ctx, "workflow_set_numeric_input", raw, []authoring.Command{command})
}

func (s *proposalState) workflowAddNode(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GraphID    string  `json:"graphId"`
		NodeTypeID string  `json:"nodeTypeId"`
		Handle     string  `json:"handle"`
		X          float64 `json:"x"`
		Y          float64 `json:"y"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, &ToolInputError{Tool: "workflow_add_node", Cause: err}
	}
	command := authoring.Command{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
		GraphID: input.GraphID, NodeTypeID: input.NodeTypeID, Handle: input.Handle,
		Position: schema.Position{X: input.X, Y: input.Y},
	}}
	return s.prepareCommands(ctx, "workflow_add_node", raw, []authoring.Command{command})
}

func (s *proposalState) workflowConnect(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GraphID    string `json:"graphId"`
		Channel    string `json:"channel"`
		FromNodeID string `json:"fromNodeId"`
		FromPortID string `json:"fromPortId"`
		ToNodeID   string `json:"toNodeId"`
		ToPortID   string `json:"toPortId"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, &ToolInputError{Tool: "workflow_connect", Cause: err}
	}
	command := authoring.Command{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{
		GraphID: input.GraphID, Edge: authoring.PatchEdge{Channel: schema.EdgeChannel(input.Channel),
			From: authoring.PatchEndpoint{NodeID: authoring.PatchNodeReference(input.FromNodeID), PortID: input.FromPortID},
			To:   authoring.PatchEndpoint{NodeID: authoring.PatchNodeReference(input.ToNodeID), PortID: input.ToPortID}},
	}}
	return s.prepareCommands(ctx, "workflow_connect", raw, []authoring.Command{command})
}

func (s *proposalState) workflowSetInputJSON(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GraphID   string `json:"graphId"`
		NodeID    string `json:"nodeId"`
		InputID   string `json:"inputId"`
		ValueJSON string `json:"valueJson"`
	}
	if err := decodeExact(raw, &input); err != nil {
		return nil, &ToolInputError{Tool: "workflow_set_input_json", Cause: err}
	}
	if len(input.ValueJSON) == 0 || len(input.ValueJSON) > 64<<10 {
		return nil, &ToolInputError{Tool: "workflow_set_input_json", Cause: errors.New("valueJson exceeds byte budget")}
	}
	var value any
	if err := decodeExact([]byte(input.ValueJSON), &value); err != nil {
		return nil, &ToolInputError{Tool: "workflow_set_input_json", Cause: err}
	}
	command := authoring.Command{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{
		GraphID: input.GraphID, NodeID: input.NodeID, PortID: input.InputID, Value: value,
	}}
	return s.prepareCommands(ctx, "workflow_set_input_json", raw, []authoring.Command{command})
}

func (s *proposalState) prepareCommands(ctx context.Context, toolName string, raw json.RawMessage, commands []authoring.Command) (json.RawMessage, error) {
	handles := map[string]struct{}{}
	for _, command := range append(append([]authoring.Command(nil), s.commands...), commands...) {
		if command.AddNode != nil && command.AddNode.Handle != "" {
			handles[command.AddNode.Handle] = struct{}{}
		}
	}
	normalizeCandidateNodeReferences(commands, handles, s.nodeAliases)
	combined := append(append([]authoring.Command(nil), s.commands...), commands...)
	prepared, err := s.manager.application.PreparePatch(ctx, authoring.PatchRequest{WorkflowID: s.workflowID, BaseRevision: s.baseRevision, Commands: combined})
	if err != nil {
		return nil, err
	}
	s.prepared = prepared.Patch
	s.commands = combined
	if s.nodeAliases == nil {
		s.nodeAliases = map[string]string{}
	}
	for _, generated := range prepared.Patch.GeneratedNodes() {
		if generated.Handle != "" {
			s.nodeAliases[generated.NodeID] = "$" + generated.Handle
		}
	}
	s.changes = normalizeChanges(combined)
	s.diagnostics = append([]schema.Diagnostic(nil), prepared.Diagnostics...)
	s.candidatePlan = append([]capability.PlanEntry(nil), prepared.CapabilityPlan...)
	s.compileChecked = false
	s.previewChecked = false
	s.permissions = permissionDelta(s.basePlan, s.candidatePlan)
	s.addToolTrace(toolName, raw, prepared.Patch.CandidateArtifact())
	s.addTrace("patch", "", map[string]string{"base_revision": fmt.Sprint(s.baseRevision), "new_revision": fmt.Sprint(s.baseRevision + 1), "base_hash": prepared.Patch.BaseHash().String(), "candidate_hash": prepared.Patch.CandidateHash().String(), "commands": fmt.Sprint(len(combined))})
	diagnostics, _ := artifact.Marshal(prepared.Diagnostics)
	return json.Marshal(map[string]any{"candidateHash": prepared.Patch.CandidateHash().String(), "newRevision": s.baseRevision + 1, "diagnosticsJson": string(diagnostics)})
}

func normalizeCandidateNodeReferences(commands []authoring.Command, handles map[string]struct{}, aliases map[string]string) {
	normalize := func(value authoring.PatchNodeReference) authoring.PatchNodeReference {
		if alias := aliases[string(value)]; alias != "" {
			return authoring.PatchNodeReference(alias)
		}
		if _, ok := handles[string(value)]; ok {
			return authoring.PatchNodeReference("$" + string(value))
		}
		return value
	}
	for index := range commands {
		command := &commands[index]
		if command.Connect != nil {
			command.Connect.Edge.From.NodeID = normalize(command.Connect.Edge.From.NodeID)
			command.Connect.Edge.To.NodeID = normalize(command.Connect.Edge.To.NodeID)
		}
		if command.BindValue != nil {
			command.BindValue.NodeID = string(normalize(authoring.PatchNodeReference(command.BindValue.NodeID)))
		}
		if command.BindDefault != nil {
			command.BindDefault.NodeID = string(normalize(authoring.PatchNodeReference(command.BindDefault.NodeID)))
		}
	}
}

func decodeAuthoringCommands(raw []byte, commands *[]authoring.Command) error {
	if commands == nil {
		return errors.New("authoring command target is nil")
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return errors.New("authoring command batch is empty")
	}
	var encoded []json.RawMessage
	if trimmed[0] == '[' {
		if err := decodeExact(trimmed, &encoded); err != nil {
			return err
		}
	} else {
		var envelope struct {
			WorkflowID   string            `json:"workflowId,omitempty"`
			BaseRevision *int64            `json:"baseRevision,omitempty"`
			GraphID      string            `json:"graphId,omitempty"`
			Commands     []json.RawMessage `json:"commands"`
		}
		if err := decodeExact(trimmed, &envelope); err != nil {
			return err
		}
		if len(envelope.Commands) == 0 {
			return errors.New("authoring command envelope is empty")
		}
		encoded = envelope.Commands
	}
	decoded := make([]authoring.Command, 0, len(encoded))
	for _, item := range encoded {
		var command authoring.Command
		if err := decodeExact(item, &command); err == nil {
			decoded = append(decoded, command)
			continue
		}
		var flat struct {
			Kind    string `json:"kind"`
			GraphID string `json:"graphId"`
			NodeID  string `json:"nodeId"`
			PortID  string `json:"portId"`
			InputID string `json:"inputId"`
			Binding struct {
				Kind  string `json:"kind"`
				Value any    `json:"value,omitempty"`
			} `json:"binding"`
		}
		if err := decodeExact(item, &flat); err != nil {
			return err
		}
		if flat.Kind != "set-node-binding" {
			return fmt.Errorf("unsupported flat authoring command kind %q", flat.Kind)
		}
		portID := flat.PortID
		if portID == "" {
			portID = flat.InputID
		}
		switch flat.Binding.Kind {
		case "value":
			command = authoring.Command{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: flat.GraphID, NodeID: flat.NodeID, PortID: portID, Value: flat.Binding.Value}}
		case "default":
			command = authoring.Command{Kind: authoring.CommandBindDefault, BindDefault: &authoring.PortCommand{GraphID: flat.GraphID, NodeID: flat.NodeID, PortID: portID}}
		default:
			return fmt.Errorf("unsupported flat binding kind %q", flat.Binding.Kind)
		}
		decoded = append(decoded, command)
	}
	*commands = decoded
	return nil
}

func (s *proposalState) workflowCompile(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	diagnostics := s.diagnostics
	candidateHash := s.baseHash
	if s.prepared.Valid() {
		candidateHash = s.prepared.CandidateHash()
	} else {
		diagnostics = s.baseDiagnostics
	}
	keyParts := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
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
	if s.prepared.Valid() {
		s.compileChecked = true
	}
	encoded, err := artifact.Marshal(diagnostics)
	if err != nil {
		return nil, err
	}
	s.addToolTrace("workflow_compile", raw, encoded)
	s.addTrace("compiler", "", map[string]string{"candidate_hash": candidateHash.String(), "diagnostics": key, "errors": fmt.Sprint(schema.HasErrors(diagnostics))})
	return json.Marshal(map[string]string{"diagnosticsJson": string(encoded)})
}

func (s *proposalState) workflowPreview(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	permissions := s.permissions
	if !s.prepared.Valid() {
		permissions = PermissionDelta{Added: []CapabilityChange{}, Removed: []CapabilityChange{}}
	} else {
		s.previewChecked = true
	}
	encoded, err := artifact.Marshal(permissions)
	if err != nil {
		return nil, err
	}
	s.addToolTrace("workflow_preview", raw, encoded)
	s.addTrace("run-preview", "", map[string]string{"admission": "not-run", "effects": "none", "added_capabilities": fmt.Sprint(len(permissions.Added)), "removed_capabilities": fmt.Sprint(len(permissions.Removed))})
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
	if s.progress != nil {
		s.progress(ConversationProgress{Kind: kind, Facts: facts})
	}
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
	case authoring.CommandRenameWorkflow, authoring.CommandUpdateWorkflowMetadata:
		return "workflow"
	case authoring.CommandAddStateVariable:
		return "state:" + command.AddStateVariable.Name
	case authoring.CommandUpdateStateVariable:
		return "state:" + command.UpdateStateVariable.Name
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
		"UNKNOWN_NODE_TYPE":             {"The source pins a node type absent from the admitted Catalog.", []string{"Search and describe the trusted catalog, then choose an admitted node."}},
		"NODE_CONTRACT_MISMATCH":        {"A node semantic digest does not match its admitted Node Contract.", []string{"Remove and re-add the node through typed authoring commands."}},
		"INVALID_CONFIG":                {"Node config violates the exact Node Contract schema.", []string{"Describe the node and set only declared fields with valid values."}},
		"INVALID_INSTRUCTION_PLACEMENT": {"A host-lowered node instruction is placed in a graph where its lifecycle semantics are invalid.", []string{"Keep run-root nodes in the main entry graph and use declared entries for subgraphs."}},
		"UNBOUND_INPUT":                 {"A required data input has no edge, value, blob, or default binding.", []string{"Connect a compatible output or bind an explicit value/default."}},
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
