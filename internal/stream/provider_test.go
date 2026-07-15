package stream_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
	"github.com/yottaapp/yotta/internal/stream"
)

type allow struct{}

func (allow) AuthorizeOpen(context.Context, resource.OpenRequest) error       { return nil }
func (allow) AuthorizeBorrow(context.Context, resource.BorrowRequest) error   { return nil }
func (allow) AuthorizeCall(context.Context, resource.AuthorizationCall) error { return nil }

func scope(run, invocation string) resource.Scope {
	return resource.Scope{
		ProgramHash:          artifact.Digest("sha256:" + strings.Repeat("1", 64)),
		CapabilityPlanDigest: artifact.Digest("sha256:" + strings.Repeat("2", 64)),
		RunID:                run, Principal: "user", PluginInstanceID: "builtin", SessionID: "session-1", InvocationID: invocation,
	}
}

func newBroker(t *testing.T) *resource.Broker {
	t.Helper()
	provider, err := stream.NewProvider(stream.Limits{MaxCapacity: 8, MaxChunkBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	broker, err := resource.New(allow{}, map[string]resource.Provider{stream.ProviderID: provider}, resource.Options{MaxPayloadBytes: 2048})
	if err != nil {
		t.Fatal(err)
	}
	return broker
}

func openStream(t *testing.T, broker *resource.Broker) (resource.Scope, resource.Handle) {
	t.Helper()
	config, err := json.Marshal(stream.Config{Capacity: 1, MaxChunkBytes: 16})
	if err != nil {
		t.Fatal(err)
	}
	owner := scope("run-1", "producer")
	handle, err := broker.Open(context.Background(), resource.OpenRequest{
		Scope: owner, ProviderID: stream.ProviderID, TargetID: "memory", Kind: stream.Kind,
		Operations: []string{stream.OperationSend, stream.OperationReceive, stream.OperationFinish, stream.OperationCancel},
		ExpiresAt:  time.Now().Add(time.Minute), Config: config,
	})
	if err != nil {
		t.Fatal(err)
	}
	return owner, handle
}

func TestBrokerOwnedStreamAppliesBackpressureAndDrainsBeforeEOF(t *testing.T) {
	broker := newBroker(t)
	producer, handle := openStream(t, broker)
	consumer := scope(producer.RunID, "consumer")
	receiver, err := broker.Borrow(context.Background(), producer, handle, consumer, []string{stream.OperationReceive}, time.Now().Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: producer, Handle: handle, Operation: stream.OperationSend, Payload: []byte("one")}); err != nil {
		t.Fatal(err)
	}
	secondDone := make(chan error, 1)
	go func() {
		_, err := broker.Invoke(context.Background(), resource.Call{Scope: producer, Handle: handle, Operation: stream.OperationSend, Payload: []byte("two")})
		secondDone <- err
	}()
	select {
	case err := <-secondDone:
		t.Fatalf("second send bypassed capacity backpressure: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	first, err := broker.Invoke(context.Background(), resource.Call{Scope: consumer, Handle: receiver, Operation: stream.OperationReceive})
	if err != nil || string(first) != "one" {
		t.Fatalf("first receive = %q, %v", first, err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: producer, Handle: handle, Operation: stream.OperationFinish}); err != nil {
		t.Fatal(err)
	}
	second, err := broker.Invoke(context.Background(), resource.Call{Scope: consumer, Handle: receiver, Operation: stream.OperationReceive})
	if err != nil || string(second) != "two" {
		t.Fatalf("second receive = %q, %v", second, err)
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: consumer, Handle: receiver, Operation: stream.OperationReceive}); !errors.Is(err, io.EOF) {
		t.Fatalf("terminal receive = %v, want EOF", err)
	}
}

func TestBrokerOwnedStreamCancellationWakesBlockedConsumer(t *testing.T) {
	broker := newBroker(t)
	owner, handle := openStream(t, broker)
	consumer := scope(owner.RunID, "consumer")
	receiver, err := broker.Borrow(context.Background(), owner, handle, consumer, []string{stream.OperationReceive}, time.Now().Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	receiveDone := make(chan error, 1)
	go func() {
		_, err := broker.Invoke(context.Background(), resource.Call{Scope: consumer, Handle: receiver, Operation: stream.OperationReceive})
		receiveDone <- err
	}()
	if _, err := broker.Invoke(context.Background(), resource.Call{
		Scope: owner, Handle: handle, Operation: stream.OperationCancel, Payload: []byte("producer failed"),
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-receiveDone:
		if !errors.Is(err, stream.ErrCanceled) {
			t.Fatalf("blocked receive returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not wake blocked receive")
	}
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: owner, Handle: handle, Operation: stream.OperationSend, Payload: []byte("late")}); !errors.Is(err, stream.ErrCanceled) {
		t.Fatalf("send after cancel = %v", err)
	}
}
