package compiler

import (
	"encoding/json"
	"fmt"
	"sync"
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
	read := stateBinding{mode: nodecontract.StateRead, slot: left.slots["value"]}
	write := stateBinding{mode: nodecontract.StateWrite, slot: left.slots["value"]}
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

func TestStateBindingUpdateIsOneAtomicWriteTransaction(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	typeRef := datatype.RefResolvedType(builtins.IntegerType.TypeRef())
	initial, err := datatype.SealInlineJSON(builtins.Catalog, typeRef, []byte("0"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := newRunState([]programStateSlot{{Name: "count", Type: typeRef, Initial: initial.Artifact()}}, builtins.Catalog, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	write := stateBinding{mode: nodecontract.StateWrite, slot: state.slots["count"]}
	read := stateBinding{mode: nodecontract.StateRead, slot: state.slots["count"]}
	if _, err := read.Update(func(value datatype.ValueEnvelope) (datatype.ValueEnvelope, error) { return value, nil }); err == nil {
		t.Fatal("read binding widened itself to atomic update")
	}
	var group sync.WaitGroup
	errorsSeen := make(chan error, 100)
	for range 100 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := write.Update(func(current datatype.ValueEnvelope) (datatype.ValueEnvelope, error) {
				var value int64
				if err := json.Unmarshal(current.InlineJSON(), &value); err != nil {
					return datatype.ValueEnvelope{}, err
				}
				return datatype.SealInlineJSON(builtins.Catalog, typeRef, []byte(fmt.Sprint(value+1)))
			})
			if err != nil {
				errorsSeen <- err
			}
		}()
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatal(err)
	}
	got := state.slots["count"].read()
	if string(got.Value.InlineJSON()) != "100" || got.Revision != 100 {
		t.Fatalf("atomic state=%s revision=%d", got.Value.InlineJSON(), got.Revision)
	}
}
