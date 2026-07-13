package compiler

import (
	"context"
	"testing"
)

func FuzzCompileDraft(f *testing.F) {
	compiler, catalogSnapshot := testCompiler(f)
	f.Add([]byte(validSource("1", 0, 0)))
	f.Add([]byte(`{"format":"yotta.workflow","version":2}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: raw, Catalog: catalogSnapshot})
		if err != nil {
			return
		}
		if program, ok := result.Program(); ok {
			if len(result.Diagnostics) != 0 || !program.Valid() {
				t.Fatal("invalid successful compile result")
			}
			if _, err := OpenProgram(program.Artifact(), catalogSnapshot, compiler.build); err != nil {
				t.Fatalf("sealed program does not open: %v", err)
			}
		}
	})
}

func FuzzOpenProgram(f *testing.F) {
	compiler, catalogSnapshot := testCompiler(f)
	result, _ := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSource("1", 0, 0)), Catalog: catalogSnapshot})
	program, _ := result.Program()
	f.Add(program.Artifact())
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) { _, _ = OpenProgram(raw, catalogSnapshot, compiler.build) })
}
