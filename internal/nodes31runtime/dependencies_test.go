package nodes31runtime_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/scriptengine"
)

type unusedScriptRuntime struct{}

func (unusedScriptRuntime) Execute(context.Context, scriptengine.Request) (scriptengine.Response, error) {
	return scriptengine.Response{}, errors.New("unexpected script execution in non-script test")
}

type unusedLogEmitter struct{}

func (unusedLogEmitter) EmitWorkflowLog(context.Context, nodes31runtime.LogEntry) error {
	return errors.New("unexpected workflow log in non-log test")
}

func testDependencies() nodes31runtime.Dependencies {
	return nodes31runtime.Dependencies{Script: unusedScriptRuntime{}, Log: unusedLogEmitter{}}
}

func TestInstalledRequiresIsolatedScriptRuntime(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nodes31runtime.Installed(builtins, nodes31runtime.Dependencies{}); err == nil {
		t.Fatal("Installed accepted missing effect runtimes")
	}
	if _, err := nodes31runtime.Installed(builtins, nodes31runtime.Dependencies{Script: unusedScriptRuntime{}}); err == nil {
		t.Fatal("Installed accepted missing workflow log runtime")
	}
}
