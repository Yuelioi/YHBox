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

func testDependencies() nodes31runtime.Dependencies {
	return nodes31runtime.Dependencies{Script: unusedScriptRuntime{}}
}

func TestInstalledRequiresIsolatedScriptRuntime(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nodes31runtime.Installed(builtins, nodes31runtime.Dependencies{}); err == nil {
		t.Fatal("Installed accepted missing isolated script runtime")
	}
}
