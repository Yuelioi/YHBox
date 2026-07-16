package noderuntime_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/scriptengine"
)

type unusedScriptRuntime struct{}

func (unusedScriptRuntime) Execute(context.Context, scriptengine.Request) (scriptengine.Response, error) {
	return scriptengine.Response{}, errors.New("unexpected script execution in non-script test")
}

type unusedLogEmitter struct{}

func (unusedLogEmitter) EmitWorkflowLog(context.Context, noderuntime.LogEntry) error {
	return errors.New("unexpected workflow log in non-log test")
}

func testDependencies() noderuntime.Dependencies {
	return noderuntime.Dependencies{Script: unusedScriptRuntime{}, Log: unusedLogEmitter{}}
}

func TestInstalledRequiresIsolatedScriptRuntime(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := noderuntime.Installed(builtins, noderuntime.Dependencies{}); err == nil {
		t.Fatal("Installed accepted missing effect runtimes")
	}
	if _, err := noderuntime.Installed(builtins, noderuntime.Dependencies{Script: unusedScriptRuntime{}}); err == nil {
		t.Fatal("Installed accepted missing workflow log runtime")
	}
}
