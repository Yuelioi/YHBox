package compiler

import (
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
)

func TestRunStateIsIsolatedTypedAndOperationAttenuated(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	typeRef := datatype.RefResolvedType(builtins.StringType.TypeRef())
	initial, err := datatype.SealInlineJSON(builtins.Catalog, typeRef, []byte(`"initial"`))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 15, 14, 30, 0, 0, time.UTC)
	left, err := newRunState([]programStateSlot{{Name: "value", Type: typeRef, Initial: initial.Artifact()}}, builtins.Catalog, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	right, err := newRunState([]programStateSlot{{Name: "value", Type: typeRef, Initial: initial.Artifact()}}, builtins.Catalog, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	read := StateBinding{mode: nodecontract.StateRead, slot: left.slots["value"]}
	write := StateBinding{mode: nodecontract.StateWrite, slot: left.slots["value"]}
	if _, err := read.Write(initial); err == nil {
		t.Fatal("read binding widened itself to write")
	}
	if _, err := write.Read(); err == nil {
		t.Fatal("write binding widened itself to read")
	}
	next, err := datatype.SealInlineJSON(builtins.Catalog, typeRef, []byte(`"next"`))
	if err != nil {
		t.Fatal(err)
	}
	written, err := write.Write(next)
	if err != nil || written.Revision != 1 || string(written.Value.InlineJSON()) != `"next"` {
		t.Fatalf("written=%#v err=%v", written, err)
	}
	if got := right.slots["value"].read(); got.Revision != 0 || string(got.Value.InlineJSON()) != `"initial"` {
		t.Fatalf("parallel Run state leaked = %#v", got)
	}
	wrong, err := datatype.SealInlineJSON(builtins.Catalog, datatype.RefResolvedType(builtins.BooleanType.TypeRef()), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write.Write(wrong); err == nil {
		t.Fatal("state slot accepted a value outside its frozen type")
	}
}
