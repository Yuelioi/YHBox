package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/codexcli"
)

// codexProvider delegates generation to the locally installed Codex
// app-server. Authentication remains entirely owned by Codex and may use the
// user's ChatGPT subscription; Yotta never reads or persists Codex tokens.
type codexProvider struct{ profile ModelProfile }

func newCodexProvider(profile ModelProfile) (*codexProvider, error) {
	if !profile.Valid() || profile.Machine().Provider != ProviderCodexSubscription {
		return nil, errors.New("Codex provider requires a Codex subscription profile")
	}
	return &codexProvider{profile: profile}, nil
}

func (p *codexProvider) Generate(ctx context.Context, _ string, request GenerateRequest) (Outcome, error) {
	if err := request.Validate(); err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	input, err := request.Prompt.ProviderInput()
	if err != nil {
		return Outcome{}, contractFailure(err.Error())
	}
	client, err := startCodexAppServer(ctx)
	if err != nil {
		return Outcome{}, codexFailure(err)
	}
	defer client.close()
	if err := client.requireChatGPTAccount(); err != nil {
		return Outcome{}, codexFailure(err)
	}
	threadID, resolved, err := client.startThread(p.profile.Machine(), nil)
	if err != nil {
		return Outcome{}, codexFailure(err)
	}
	var schema json.RawMessage
	if request.Output != nil {
		schema = request.Output.Schema
	}
	result, err := client.startTurn(threadID, input, schema)
	if err != nil {
		return Outcome{}, codexFailure(err)
	}
	return p.outcome(request.AttemptID, resolved, result, request.Output != nil), nil
}

func (p *codexProvider) StartAgent(ctx context.Context, _ string, request AgentStartRequest) (Outcome, any, error) {
	if err := request.Validate(); err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	input, err := request.Prompt.ProviderInput()
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	toolSet, err := request.ToolSet.Open()
	if err != nil {
		return Outcome{}, nil, contractFailure(err.Error())
	}
	client, err := startCodexAppServer(ctx)
	if err != nil {
		return Outcome{}, nil, codexFailure(err)
	}
	if err := client.requireChatGPTAccount(); err != nil {
		client.close()
		return Outcome{}, nil, codexFailure(err)
	}
	threadID, resolved, err := client.startThread(p.profile.Machine(), toolSet.Machine().Tools)
	if err != nil {
		client.close()
		return Outcome{}, nil, codexFailure(err)
	}
	result, err := client.startTurn(threadID, input, nil)
	if err != nil {
		client.close()
		return Outcome{}, nil, codexFailure(err)
	}
	state := &codexAgentState{client: client, threadID: threadID, resolvedModel: resolved, pending: result.pending}
	outcome := p.outcome(request.AttemptID, resolved, result, false)
	if len(result.pending) == 0 {
		client.close()
		return outcome, nil, nil
	}
	return outcome, state, nil
}

func (p *codexProvider) ContinueAgent(_ context.Context, _ string, state any, request AgentContinueRequest) (Outcome, any, error) {
	current, ok := state.(*codexAgentState)
	if !ok || current.client == nil {
		return Outcome{}, nil, contractFailure("Codex agent continuation state is unavailable")
	}
	if err := request.Validate(); err != nil {
		current.client.close()
		return Outcome{}, nil, contractFailure(err.Error())
	}
	byCall := make(map[string]ToolResult, len(request.Results))
	for _, result := range request.Results {
		byCall[result.CallID] = result
	}
	for _, pending := range current.pending {
		result, exists := byCall[pending.callID]
		if !exists {
			current.client.close()
			return Outcome{}, nil, contractFailure("Codex agent tool result is missing")
		}
		if err := current.client.respondTool(pending.requestID, result.Value); err != nil {
			current.client.close()
			return Outcome{}, nil, codexFailure(err)
		}
	}
	result, err := current.client.readTurn()
	if err != nil {
		current.client.close()
		return Outcome{}, nil, codexFailure(err)
	}
	current.pending = result.pending
	outcome := p.outcome(request.AttemptID, current.resolvedModel, result, false)
	if len(result.pending) == 0 {
		current.client.close()
		return outcome, nil, nil
	}
	return outcome, current, nil
}

func (p *codexProvider) outcome(attemptID, resolved string, result codexTurnResult, structured bool) Outcome {
	items := make([]OutputItem, 0, len(result.pending)+1)
	for _, call := range result.pending {
		items = append(items, OutputItem{Kind: OutputToolCall, ToolCall: &ToolCall{CallID: call.callID, Name: call.name, Arguments: call.arguments}})
	}
	finish := FinishCompleted
	if len(result.pending) != 0 {
		finish = FinishToolCalls
	} else if structured {
		items = append(items, OutputItem{Kind: OutputStructured, Structured: &StructuredOutput{Value: json.RawMessage(result.text)}})
	} else {
		items = append(items, OutputItem{Kind: OutputText, Text: &TextOutput{Text: result.text}})
	}
	input, output, cached, reasoning, cost := result.inputTokens, result.outputTokens, result.cachedTokens, result.reasoningTokens, int64(0)
	return Outcome{
		Provider: ProviderCodexSubscription, RequestedModel: p.profile.Machine().Model, ResolvedModel: resolved,
		ProviderRequestID: attemptID, Items: items, Finish: Finish{Kind: finish},
		Usage: TokenUsage{InputTotal: &input, CacheRead: &cached, OutputTotal: &output, ReasoningOutput: &reasoning, CostMicrounits: &cost},
	}
}

type codexAgentState struct {
	client        *codexClient
	threadID      string
	resolvedModel string
	pending       []codexPendingTool
}

type codexPendingTool struct {
	requestID json.RawMessage
	callID    string
	name      string
	arguments json.RawMessage
}

type codexTurnResult struct {
	text                                    string
	pending                                 []codexPendingTool
	inputTokens, outputTokens, cachedTokens int64
	reasoningTokens                         int64
}

type codexClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	stderr  bytes.Buffer
	mu      sync.Mutex
	nextID  int64
}

func startCodexAppServer(ctx context.Context) (*codexClient, error) {
	cmd, err := codexcli.CommandContext(ctx, "app-server", "--stdio")
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	client := &codexClient{cmd: cmd, stdin: stdin, scanner: bufio.NewScanner(stdout)}
	client.scanner.Buffer(make([]byte, 64<<10), MaxAgentProviderStateBytes)
	cmd.Stderr = &client.stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var initialized json.RawMessage
	if err := client.call("initialize", map[string]any{
		"clientInfo":   map[string]any{"name": "yotta", "title": "Yotta", "version": "1"},
		"capabilities": map[string]any{"experimentalApi": true},
	}, &initialized); err != nil {
		client.close()
		return nil, err
	}
	if err := client.notify("initialized", map[string]any{}); err != nil {
		client.close()
		return nil, err
	}
	return client, nil
}

func (c *codexClient) requireChatGPTAccount() error {
	var response struct {
		Account *struct {
			Type string `json:"type"`
		} `json:"account"`
	}
	if err := c.call("account/read", map[string]any{"refreshToken": false}, &response); err != nil {
		return err
	}
	if response.Account == nil || response.Account.Type != "chatgpt" {
		return errors.New("Codex is not signed in with a ChatGPT subscription; run `codex login`")
	}
	return nil
}

func (c *codexClient) startThread(profile ModelProfileDraft, tools []ToolManifestDraft) (string, string, error) {
	dynamic := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		var schema any
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			return "", "", err
		}
		dynamic = append(dynamic, map[string]any{"type": "function", "name": tool.Name, "description": tool.Description, "inputSchema": schema})
	}
	params := map[string]any{
		"model": profile.Model, "ephemeral": true, "approvalPolicy": "never", "sandbox": "read-only",
		"cwd": os.TempDir(), "environments": []any{}, "runtimeWorkspaceRoots": []any{},
		"baseInstructions": "Act only as Yotta's bounded model provider. Do not inspect files, run commands, browse, or use tools other than the dynamic tools supplied by Yotta.",
		"dynamicTools":     dynamic,
	}
	var response struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		Model string `json:"model"`
	}
	if err := c.call("thread/start", params, &response); err != nil {
		return "", "", err
	}
	if response.Thread.ID == "" {
		return "", "", errors.New("Codex app-server omitted the thread id")
	}
	return response.Thread.ID, response.Model, nil
}

func (c *codexClient) startTurn(threadID, input string, outputSchema json.RawMessage) (codexTurnResult, error) {
	params := map[string]any{"threadId": threadID, "input": []map[string]any{{"type": "text", "text": input}}}
	if len(outputSchema) != 0 {
		var schema any
		if err := json.Unmarshal(outputSchema, &schema); err != nil {
			return codexTurnResult{}, err
		}
		params["outputSchema"] = schema
	}
	var response json.RawMessage
	if err := c.call("turn/start", params, &response); err != nil {
		return codexTurnResult{}, err
	}
	return c.readTurn()
}

func (c *codexClient) readTurn() (codexTurnResult, error) {
	var result codexTurnResult
	for c.scanner.Scan() {
		line := append([]byte(nil), c.scanner.Bytes()...)
		var message struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(line, &message); err != nil {
			return result, err
		}
		switch message.Method {
		case "item/tool/call":
			var params struct {
				CallID    string          `json:"callId"`
				Tool      string          `json:"tool"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if err := json.Unmarshal(message.Params, &params); err != nil {
				return result, err
			}
			result.pending = append(result.pending, codexPendingTool{requestID: message.ID, callID: params.CallID, name: params.Tool, arguments: params.Arguments})
			return result, nil
		case "item/completed":
			var params struct {
				Item struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"item"`
			}
			if json.Unmarshal(message.Params, &params) == nil && params.Item.Type == "agentMessage" {
				result.text = params.Item.Text
			}
		case "thread/tokenUsage/updated":
			var params struct {
				TokenUsage struct {
					Last struct {
						Input     int64 `json:"inputTokens"`
						Output    int64 `json:"outputTokens"`
						Cached    int64 `json:"cachedInputTokens"`
						Reasoning int64 `json:"reasoningOutputTokens"`
					} `json:"last"`
				} `json:"tokenUsage"`
			}
			if json.Unmarshal(message.Params, &params) == nil {
				result.inputTokens, result.outputTokens = params.TokenUsage.Last.Input, params.TokenUsage.Last.Output
				result.cachedTokens, result.reasoningTokens = params.TokenUsage.Last.Cached, params.TokenUsage.Last.Reasoning
			}
		case "turn/completed":
			var params struct {
				Turn struct {
					Status string `json:"status"`
					Error  *struct {
						Message string `json:"message"`
					} `json:"error"`
				} `json:"turn"`
			}
			if err := json.Unmarshal(message.Params, &params); err != nil {
				return result, err
			}
			if params.Turn.Status != "completed" {
				message := "Codex turn ended with status " + params.Turn.Status
				if params.Turn.Error != nil && params.Turn.Error.Message != "" {
					message += ": " + params.Turn.Error.Message
				}
				return result, errors.New(message)
			}
			if result.text == "" {
				return result, errors.New("Codex completed without a model response")
			}
			return result, nil
		case "item/commandExecution/requestApproval", "item/fileChange/requestApproval", "item/permissions/requestApproval":
			return result, errors.New("Codex requested an operation outside Yotta's bounded AI provider tools")
		}
	}
	return result, c.scanError()
}

func (c *codexClient) respondTool(id json.RawMessage, value json.RawMessage) error {
	content := string(value)
	return c.write(map[string]any{"id": id, "result": map[string]any{"success": true, "contentItems": []map[string]any{{"type": "inputText", "text": content}}}})
}

func (c *codexClient) call(method string, params any, target any) error {
	c.nextID++
	id := c.nextID
	if err := c.write(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return err
	}
	for c.scanner.Scan() {
		var response struct {
			ID     *int64          `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(c.scanner.Bytes(), &response); err != nil {
			return err
		}
		if response.ID == nil || *response.ID != id {
			continue
		}
		if response.Error != nil {
			return errors.New(response.Error.Message)
		}
		if target != nil {
			return json.Unmarshal(response.Result, target)
		}
		return nil
	}
	return c.scanError()
}

func (c *codexClient) notify(method string, params any) error {
	return c.write(map[string]any{"method": method, "params": params})
}

func (c *codexClient) write(value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	_, err = c.stdin.Write(raw)
	return err
}

func (c *codexClient) scanError() error {
	if err := c.scanner.Err(); err != nil {
		return err
	}
	detail := strings.TrimSpace(c.stderr.String())
	if len(detail) > 4096 {
		detail = detail[len(detail)-4096:]
	}
	if detail != "" {
		return fmt.Errorf("Codex app-server stopped: %s", detail)
	}
	return errors.New("Codex app-server stopped unexpectedly")
}

func (c *codexClient) close() {
	if c == nil || c.cmd == nil {
		return
	}
	_ = c.stdin.Close()
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	_ = c.cmd.Wait()
	c.cmd = nil
}

func codexFailure(err error) error {
	return &ProviderFailure{Stage: FailureTransport, Class: FailureUnknown, Message: err.Error(), Retry: RetryNever}
}
