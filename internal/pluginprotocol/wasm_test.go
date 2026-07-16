package pluginprotocol

import (
	"bytes"
	"strings"
	"testing"
)

func TestWasmBootstrapRoundTripAndDigest(t *testing.T) {
	want := WasmBootstrap{MemoryLimitPages: 16, Module: []byte("\x00asm\x01\x00\x00\x00")}
	var stream bytes.Buffer
	if err := WriteWasmBootstrap(&stream, want); err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), stream.Bytes()...)
	got, err := ReadWasmBootstrap(&stream)
	if err != nil {
		t.Fatal(err)
	}
	if got.MemoryLimitPages != want.MemoryLimitPages || !bytes.Equal(got.Module, want.Module) {
		t.Fatalf("bootstrap = %#v, want %#v", got, want)
	}
	raw[len(raw)-1] ^= 0xff
	if _, err := ReadWasmBootstrap(bytes.NewReader(raw)); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("tampered bootstrap error = %v", err)
	}
}
