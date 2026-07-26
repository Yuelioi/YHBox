package noderuntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func httpGet(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		action := compiler.AdapterAction{EffectID: nodes.HTTPGetEffectID, Action: "network.http-response", SummaryCode: "network.http-get", Counters: map[string]int64{}, Facts: map[string]string{}}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, httpegress.CodeRequestFailed, runErr))
		}()

		request, err := httpGetRequest(invocation)
		if err != nil {
			return compiler.AdapterResult{}, httpFailure(httpegress.CodeInvalidRequest, err)
		}
		pathDigest, err := artifact.Sum("yotta/http-relative-path/v1", []byte(request.Path))
		if err != nil {
			return compiler.AdapterResult{}, httpFailure(httpegress.CodeContractViolation, err)
		}
		action.Facts["path_digest"] = pathDigest.String()
		session := invocation.Sessions["origin"]
		if session == nil {
			return compiler.AdapterResult{}, httpFailure(httpegress.CodeContractViolation, errors.New("HTTP origin capability session is missing"))
		}
		handle, err := session.Open(ctx, []string{httpegress.OperationGet}, []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapHTTPFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		payload, err := artifact.Marshal(request)
		if err != nil {
			return compiler.AdapterResult{}, httpFailure(httpegress.CodeContractViolation, err)
		}
		raw, err := session.Invoke(ctx, handle, httpegress.OperationGet, payload)
		if err != nil {
			return compiler.AdapterResult{}, mapHTTPFailure(err)
		}
		response, err := httpegress.OpenGetResponse(raw, httpegress.MaxResponseBytes)
		if err != nil {
			return compiler.AdapterResult{}, mapHTTPFailure(err)
		}
		action.Counters["response_bytes"] = int64(len(response.Body))
		action.Counters["status_code"] = int64(response.StatusCode)
		outputs := map[string]json.RawMessage{}
		outputs["status"], err = json.Marshal(response.StatusCode)
		if err == nil {
			outputs["body"], err = json.Marshal(response.Body)
		}
		if err == nil {
			outputs["content-type"], err = json.Marshal(response.ContentType)
		}
		if err != nil {
			return compiler.AdapterResult{}, httpFailure(httpegress.CodeContractViolation, err)
		}
		sealed := make(map[string]datatype.ValueEnvelope, len(outputs))
		for id, rawValue := range outputs {
			resolved, ok := invocation.OutputTypes[id]
			if !ok {
				return compiler.AdapterResult{}, httpFailure(httpegress.CodeContractViolation, fmt.Errorf("HTTP output %q is unresolved", id))
			}
			value, err := datatype.SealInlineJSON(builtins.Catalog, resolved, rawValue)
			if err != nil {
				return compiler.AdapterResult{}, httpFailure(httpegress.CodeContractViolation, err)
			}
			sealed[id] = value
		}
		return compiler.AdapterResult{Outputs: sealed, ExecOutputs: []string{"completed"}}, nil
	}
}

func httpGetRequest(invocation compiler.Invocation) (httpegress.GetRequest, error) {
	pathInput, ok := invocation.Inputs["path"]
	if !ok {
		return httpegress.GetRequest{}, errors.New("HTTP path input is missing")
	}
	var path string
	if err := json.Unmarshal(pathInput.InlineJSON(), &path); err != nil {
		return httpegress.GetRequest{}, err
	}
	queryInput, ok := invocation.Inputs["query"]
	if !ok {
		return httpegress.GetRequest{}, errors.New("HTTP query input is missing")
	}
	query := map[string][]string{}
	if err := decodeHTTPQuery(queryInput.InlineJSON(), &query); err != nil {
		return httpegress.GetRequest{}, err
	}
	return httpegress.GetRequest{Path: path, Query: query}, nil
}

func decodeHTTPQuery(raw []byte, target *map[string][]string) error {
	if len(raw) == 0 || len(raw) > 128<<10 {
		return errors.New("HTTP query exceeds byte budget")
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return errors.New("HTTP query must be an object of string lists")
	}
	result := make(map[string][]string, len(object))
	for key, rawValues := range object {
		values, ok := rawValues.([]any)
		if !ok {
			return errors.New("HTTP query values must be string lists")
		}
		stringsOnly := make([]string, len(values))
		for index, rawValue := range values {
			value, ok := rawValue.(string)
			if !ok {
				return errors.New("HTTP query values must be strings")
			}
			stringsOnly[index] = value
		}
		result[key] = stringsOnly
	}
	*target = result
	return nil
}

func mapHTTPFailure(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var failure *httpegress.Failure
	if errors.As(err, &failure) && failure.Code != "" {
		return httpFailure(failure.Code, err)
	}
	return httpFailure(httpegress.CodeContractViolation, err)
}

func httpFailure(code string, cause error) error {
	return &compiler.NodeFailure{Code: code, Output: "failed", Cause: cause}
}
