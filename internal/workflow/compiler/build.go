package compiler

import (
	"errors"

	"github.com/yottaapp/yotta/internal/artifact"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const compilerImplementationVersion = "v2"

// BuildDigest identifies the installed lowering/interpreter contract. Bump the
// implementation version whenever executable Program semantics change; stored
// Programs then fail strict-open instead of running under different code.
func BuildDigest() (artifact.Digest, error) {
	if schema.MaxGraphPath != run.MaxJournalGraphPathSegments {
		return "", errors.New("workflow and Run graph path budgets differ")
	}
	manifest, err := artifact.Marshal(map[string]any{
		"workflowFormat":        schema.Format,
		"workflowVersion":       schema.Version,
		"programFormat":         ProgramFormat,
		"programVersion":        ProgramVersion,
		"implementationVersion": compilerImplementationVersion,
		"schedulerBudget":       MaxScheduledInvocations,
		"signalLowering":        "ordered-exec-error-routes/v1",
		"instructionLowering":   "activation-scoped-regions/v1",
		"dataLowering":          "pull-bindings-topological-order/v1",
		"graphLowering":         "source-native-call-expansion/v1",
		"graphDepthBudget":      schema.MaxGraphDepth,
		"graphPathBudget":       schema.MaxGraphPath,
		"journalPathBudget":     run.MaxJournalGraphPathSegments,
	})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/compiler-build-manifest/v1", manifest)
}
