package noderuntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

func fileRead(builtins nodes.Builtins, structured bool) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{
			EffectID: nodes.FileReadEffectID, Action: "filesystem.file-read", SummaryCode: "filesystem.read",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, workspacefs.CodeReadFailed, runErr))
		}()

		path, err := filePath(invocation)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		maxBytes, err := fileReadBudget(invocation.Config)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		pathDigest, err := artifact.Sum("yotta/workspace-file-path/v1", []byte(path))
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		action.Facts["path_digest"] = pathDigest.String()

		session := invocation.Sessions["workspace-files"]
		if session == nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, errors.New("workspace filesystem capability session is missing"))
		}
		handle, err := session.Open(ctx, []string{workspacefs.OperationRead}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapFileFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		payload, err := artifact.Marshal(workspacefs.ReadRequest{Path: path, MaxBytes: maxBytes})
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		raw, err := session.Invoke(ctx, handle, workspacefs.OperationRead, payload)
		if err != nil {
			return compiler.AdapterResult{}, mapFileFailure(err)
		}
		response, err := workspacefs.OpenReadResponse(raw, maxBytes)
		if err != nil {
			return compiler.AdapterResult{}, mapFileFailure(err)
		}
		action.Counters["bytes"] = int64(len(response.Data))

		text, err := decodeWorkspaceText(response.Data, fileEncoding(invocation.Config, structured))
		if err != nil {
			return compiler.AdapterResult{}, fileFailure("filesystem.decode_failed", err)
		}
		outputs := map[string]json.RawMessage{}
		outputs["text"], err = json.Marshal(text)
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		outputs["metadata"], err = artifact.Marshal(response.Metadata)
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		if structured {
			outputs["value"], err = canonicalJSONDocument([]byte(text))
			if err != nil {
				return compiler.AdapterResult{}, fileFailure("filesystem.invalid_json", err)
			}
		}
		sealed, err := sealFileOutputs(builtins, invocation, outputs)
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		return compiler.AdapterResult{Outputs: sealed, ExecOutputs: []string{"completed"}}, nil
	}
}

func fileStat(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{
			EffectID: nodes.FileStatEffectID, Action: "filesystem.file-statted", SummaryCode: "filesystem.stat",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, workspacefs.CodeReadFailed, runErr))
		}()
		path, err := filePath(invocation)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		pathDigest, err := artifact.Sum("yotta/workspace-file-path/v1", []byte(path))
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		action.Facts["path_digest"] = pathDigest.String()
		session := invocation.Sessions["workspace-files"]
		if session == nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, errors.New("workspace filesystem capability session is missing"))
		}
		handle, err := session.Open(ctx, []string{workspacefs.OperationStat}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapFileFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		payload, err := artifact.Marshal(workspacefs.StatRequest{Path: path})
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		raw, err := session.Invoke(ctx, handle, workspacefs.OperationStat, payload)
		if err != nil {
			return compiler.AdapterResult{}, mapFileFailure(err)
		}
		metadata, err := workspacefs.OpenMetadata(raw)
		if err != nil {
			return compiler.AdapterResult{}, mapFileFailure(err)
		}
		action.Counters["bytes"] = metadata.Size
		metadataJSON, err := artifact.Marshal(metadata)
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		sealed, err := sealFileOutputs(builtins, invocation, map[string]json.RawMessage{"metadata": metadataJSON})
		if err != nil {
			return compiler.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		return compiler.AdapterResult{Outputs: sealed, ExecOutputs: []string{"completed"}}, nil
	}
}

func filePath(invocation compiler.Invocation) (string, error) {
	envelope, ok := invocation.Inputs["path"]
	if !ok || len(envelope.InlineJSON()) == 0 {
		return "", fileFailure(workspacefs.CodeInvalidPath, errors.New("workspace file path is missing"))
	}
	var path string
	if err := json.Unmarshal(envelope.InlineJSON(), &path); err != nil || path == "" {
		return "", fileFailure(workspacefs.CodeInvalidPath, errors.New("workspace file path is invalid"))
	}
	return path, nil
}

func fileReadBudget(config map[string]any) (int, error) {
	value, exists := config["maxBytes"]
	if !exists {
		return nodes.DefaultFileReadBytes, nil
	}
	maximum, err := configInt64(value)
	if err != nil || maximum < 1 || maximum > nodes.DefaultFileReadBytes {
		return 0, fileFailure(workspacefs.CodeContractViolation, errors.New("workspace file read budget is invalid"))
	}
	return int(maximum), nil
}

func fileEncoding(config map[string]any, structured bool) string {
	if structured {
		return "utf-8"
	}
	if value, ok := config["encoding"].(string); ok && value != "" {
		return value
	}
	return "auto"
}

func decodeWorkspaceText(data []byte, encoding string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "auto":
		if utf8.Valid(data) {
			return string(data), nil
		}
		return simplifiedchinese.GBK.NewDecoder().String(string(data))
	case "utf-8":
		if !utf8.Valid(data) {
			return "", errors.New("workspace file is not valid UTF-8")
		}
		return string(data), nil
	case "gbk":
		return simplifiedchinese.GBK.NewDecoder().String(string(data))
	default:
		return "", errors.New("workspace file encoding is unsupported")
	}
}

func canonicalJSONDocument(raw []byte) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("JSON document contains trailing values")
	}
	return artifact.Marshal(value)
}

func sealFileOutputs(builtins nodes.Builtins, invocation compiler.Invocation, raw map[string]json.RawMessage) (map[string]datatype.ValueEnvelope, error) {
	outputs := make(map[string]datatype.ValueEnvelope, len(raw))
	for id, value := range raw {
		resolved, ok := invocation.OutputTypes[id]
		if !ok {
			return nil, fmt.Errorf("filesystem output %q is unresolved", id)
		}
		envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, value)
		if err != nil {
			return nil, fmt.Errorf("seal filesystem output %q: %w", id, err)
		}
		outputs[id] = envelope
	}
	return outputs, nil
}

func mapFileFailure(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var failure *workspacefs.Failure
	if errors.As(err, &failure) && failure.Code != "" {
		return fileFailure(failure.Code, err)
	}
	return fileFailure(workspacefs.CodeContractViolation, err)
}

func fileFailure(code string, cause error) error {
	return &compiler.NodeFailure{Code: code, Output: "failed", Cause: cause}
}
