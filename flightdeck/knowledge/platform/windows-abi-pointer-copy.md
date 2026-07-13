---
kind: checklist
summary: "Always copy Win32 callback / native DLL `uintptr` memory into Go-owned values through `pkg/winutil` helpers; never retain or directly cast external addresses to Go pointers."
activation: action
read_when: "before decoding Win32 callback lParam, reading a native DLL C string, fixing `possible misuse of unsafe.Pointer`, or adding a Windows ABI adapter."
---
# Windows ABI 裸地址复制 checklist
Win32 callback 和 `syscall.LazyProc.Call` 用 `uintptr` 搬运 OS/C 地址。该地址不是 Go object，把它直接写成 `(*T)(unsafe.Pointer(src))` 不属于 Go `unsafe.Pointer` 文档列出的安全往返模式，也会触发 `go vet` 的 unsafeptr analyzer。

Yotta 的统一入口是：

- `winutil.ReadStructFromPointer[T](src)`：用 `RtlMoveMemory` 把 pointer-free C struct 立即复制到 Go-owned `T`。
- `winutil.ReadCString(src)`：先用 `lstrlenA` 取长度，再复制为 Go-owned bytes/string。

调用约束：native 地址必须在调用期间可读；struct `T` 不得含 Go pointer。不要为消除 vet warning 改用 reflect 绕过分析器、关闭 unsafeptr，或把 `uintptr` 存到长期字段。
