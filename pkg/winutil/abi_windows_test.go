package winutil

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestReadStructFromPointer(t *testing.T) {
	type sample struct {
		A uint32
		B int32
	}
	src := sample{A: 42, B: -7}
	got := ReadStructFromPointer[sample](uintptr(unsafe.Pointer(&src)))
	runtime.KeepAlive(&src)
	if got != src {
		t.Fatalf("copied struct = %#v, want %#v", got, src)
	}
}

func TestReadCString(t *testing.T) {
	src := []byte{'y', 'o', 't', 't', 'a', 0}
	got := ReadCString(uintptr(unsafe.Pointer(&src[0])))
	runtime.KeepAlive(src)
	if got != "yotta" {
		t.Fatalf("copied string = %q, want %q", got, "yotta")
	}
}
