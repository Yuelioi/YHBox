---
kind: checklist
summary: "Win32 callback / native DLL uintptr must not be cast directly to Go pointers: copy native memory through winutil helpers, and carry Go callback state through a temporary registry token."
activation: action
read_when: "before decoding Win32 callback lParam, reading a native DLL C string, passing Go state through a Windows callback, fixing possible misuse of unsafe.Pointer, or adding a Windows ABI adapter."
---
# Windows ABI 裸地址与 callback state checklist
Win32 callback 和 syscall.LazyProc.Call 的 uintptr 有两种不同来源，必须分别处理。

OS/C 提供的地址不是 Go object，把它直接写成 (*T)(unsafe.Pointer(src)) 不属于 Go unsafe.Pointer 文档列出的安全往返模式，也会触发 go vet 的 unsafeptr analyzer。统一使用 winutil.ReadStructFromPointer[T](src) 或 winutil.ReadCString(src) 立即复制到 Go-owned 值；native 地址必须在调用期间可读，struct T 不得含 Go pointer。

Go-owned callback state 也不能先转 uintptr 再转回 Go pointer。为同步 Win32 API 建立短期、并发安全的 token registry，把 token 放进 lParam，callback 通过 registry 取回对象，并在原生调用返回后删除。不要用 reflect 绕过分析器；runtime/cgo.Handle 会让 Windows CGO_ENABLED=0 compile gate 在链接期失败，因此不适用于同时支持该门禁的包。
