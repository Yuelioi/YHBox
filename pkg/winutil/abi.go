package winutil

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	abiKernel32       = syscall.NewLazyDLL("kernel32.dll")
	procRtlMoveMemory = abiKernel32.NewProc("RtlMoveMemory")
	procLstrlenA      = abiKernel32.NewProc("lstrlenA")
)

// ReadStructFromPointer copies a pointer-free C struct into Go-owned memory.
// src must remain readable for the duration of the call and point to at least sizeof(T) bytes.
func ReadStructFromPointer[T any](src uintptr) (value T) {
	size := unsafe.Sizeof(value)
	if src == 0 || size == 0 {
		return value
	}
	procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&value)), src, size)
	runtime.KeepAlive(&value)
	return value
}

// ReadCString copies a null-terminated C string into Go-owned memory.
func ReadCString(src uintptr) string {
	if src == 0 {
		return ""
	}
	length, _, _ := procLstrlenA.Call(src)
	if length == 0 {
		return ""
	}
	buf := make([]byte, length)
	procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&buf[0])), src, length)
	runtime.KeepAlive(buf)
	return string(buf)
}
