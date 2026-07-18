// Package winutil 提供 Windows 顶层窗口枚举 / 匹配 / 元数据查询.
// 跨权限读 admin 进程 exe 名用 PROCESS_QUERY_LIMITED_INFORMATION (Vista+).
//
// 核心 API: WindowHandle / MatchSpec / ResolveWindow — Win32WindowTarget 节点用,
// 支持任意 title/class/processName 匹配 + 元数据完整返回. 另有 BringToFront
// (置前台) 给 recording 用.
package winutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procEnumWindows           = user32.NewProc("EnumWindows")
	procGetWindowTextW        = user32.NewProc("GetWindowTextW")
	procGetClassNameW         = user32.NewProc("GetClassNameW")
	procSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	procShowWindow            = user32.NewProc("ShowWindow")
	procIsIconic              = user32.NewProc("IsIconic")
	procAttachThreadInput     = user32.NewProc("AttachThreadInput")
	procGetWindowThreadProcId = user32.NewProc("GetWindowThreadProcessId")
	procGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	procGetClientRect         = user32.NewProc("GetClientRect")

	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadId         = kernel32.NewProc("GetCurrentThreadId")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
)

const (
	swRestore                      = 9
	processQueryLimitedInformation = 0x1000
)

// ---------------------------------------------------------------------------
// BringToFront — 把窗口置前台 (recording 用).
// ---------------------------------------------------------------------------

// BringToFront 把目标窗口拉到前台并恢复（如最小化）。Windows 限制：当前进程
// 不持有 fg 锁时直接 SetForegroundWindow 会被忽略，这里用 AttachThreadInput 借
// 当前前台线程的输入队列把限制绕过。失败返 false 不抛错——非关键路径。
func BringToFront(hwnd uintptr) error {
	if hwnd == 0 {
		return errors.New("hwnd 0")
	}
	// 最小化的话先 restore
	iconic, _, _ := procIsIconic.Call(uintptr(hwnd))
	if iconic != 0 {
		procShowWindow.Call(uintptr(hwnd), swRestore)
	}
	curTid, _, _ := procGetCurrentThreadId.Call()
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	fgTid, _, _ := procGetWindowThreadProcId.Call(fgHwnd, 0)
	attached := false
	if fgTid != 0 && fgTid != curTid {
		ret, _, _ := procAttachThreadInput.Call(curTid, fgTid, 1)
		attached = ret != 0
	}
	defer func() {
		if attached {
			procAttachThreadInput.Call(curTid, fgTid, 0)
		}
	}()
	ret, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	if ret == 0 {
		return errors.New("SetForegroundWindow rejected the request")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Win32WindowTarget API — runtime/recording/tools 共用.
// ---------------------------------------------------------------------------

// windowMatchOnce 枚举一遍顶层可见窗口, 有任一匹配 spec/titleRe 的窗口则返 true.
// targetProc 是 strings.ToLower(spec.ProcessName), 调用方预先处理好传入.
// 注意: syscall.NewCallback 分配的内存进程生命周期不回收 — 每次调用永久 leak 一个 callback slot.
// 仅 WaitWindowGone 调用, 频率 = 其 poll interval (500ms), 可接受.
func windowMatchOnce(spec MatchSpec, titleRe *regexp.Regexp, targetProc string) bool {
	var found bool
	callback := syscall.NewCallback(func(hwnd win.HWND, _ uintptr) uintptr {
		if !win.IsWindowVisible(hwnd) {
			return 1
		}
		title := getWindowText(hwnd)
		class := getClassName(hwnd)
		pid := getWindowPID(hwnd)
		procName, procErr := queryProcessName(pid)
		if procErr != nil {
			return 1
		}
		procName = strings.ToLower(procName)

		if spec.Title != "" {
			if titleRe != nil {
				if !titleRe.MatchString(title) {
					return 1
				}
			} else if title != spec.Title {
				return 1
			}
		}
		if spec.Class != "" && class != spec.Class {
			return 1
		}
		if targetProc != "" && procName != targetProc {
			return 1
		}

		found = true
		return 0 // stop enumeration
	})
	procEnumWindows.Call(callback, 0)
	return found
}

// ResolveWindow 按 spec 匹配条件枚 top-level visible window, 第一个匹配返 WindowHandle.
// EnumWindows 按 Z-order (前台最上为先) 顺序回调, MSDN 有写. fallback: GetTopWindow + GetWindow(GW_HWNDNEXT).
// OpenProcess 用 PROCESS_QUERY_LIMITED_INFORMATION 跨权限. 单进程 query 失败 → 视该进程不匹配 + 继续.
func ResolveWindow(ctx context.Context, spec MatchSpec, timeout, interval time.Duration) (WindowHandle, error) {
	if IsEmptyMatch(spec) {
		return WindowHandle{}, errors.New("Win32WindowTarget match spec is empty or matches anything")
	}
	titleRe, err := CompileTitle(spec)
	if err != nil {
		return WindowHandle{}, fmt.Errorf("title regex invalid: %w", err)
	}

	targetProc := strings.ToLower(spec.ProcessName)

	// 构造完整 WindowHandle 的 callback — 只在找到匹配时才用, 不每轮都构造.
	// syscall.NewCallback 永久 leak, 这里只构造一次.
	var result WindowHandle
	var found bool
	fullCallback := syscall.NewCallback(func(hwnd win.HWND, _ uintptr) uintptr {
		if !win.IsWindowVisible(hwnd) {
			return 1
		}
		title := getWindowText(hwnd)
		class := getClassName(hwnd)
		pid := getWindowPID(hwnd)
		procName, procErr := queryProcessName(pid)
		if procErr != nil {
			return 1
		}
		procName = strings.ToLower(procName)

		if spec.Title != "" {
			if titleRe != nil {
				if !titleRe.MatchString(title) {
					return 1
				}
			} else if title != spec.Title {
				return 1
			}
		}
		if spec.Class != "" && class != spec.Class {
			return 1
		}
		if targetProc != "" && procName != targetProc {
			return 1
		}

		cw, ch := getClientSize(hwnd)
		result = WindowHandle{
			HWND:        uintptr(hwnd),
			Title:       title,
			Class:       class,
			ProcessName: procName,
			PID:         pid,
			ClientW:     cw,
			ClientH:     ch,
		}
		found = true
		return 0
	})

	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return WindowHandle{}, err
		}
		result = WindowHandle{}
		found = false
		procEnumWindows.Call(fullCallback, 0)
		if found {
			return result, nil
		}
		if time.Now().After(deadline) {
			return WindowHandle{}, fmt.Errorf("%w (title=%q class=%q process=%q), 请打开游戏后重试",
				ErrWindowNotFound, spec.Title, spec.Class, spec.ProcessName)
		}
		select {
		case <-ctx.Done():
			return WindowHandle{}, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// WaitWindowGone 轮询等待匹配窗口从系统消失. 窗口已不存在(或一开始就不存在) → return nil;
// timeout 内仍存在 → return ErrWindowStillPresent; ctx 取消 → ctx.Err().
// 空 spec → error (同 ResolveWindow 守卫); titleMatch=regex 且 regex 非法 → error.
func WaitWindowGone(ctx context.Context, spec MatchSpec, timeout, interval time.Duration) error {
	if IsEmptyMatch(spec) {
		return errors.New("WaitWindowGone match spec is empty or matches anything")
	}
	titleRe, err := CompileTitle(spec)
	if err != nil {
		return fmt.Errorf("title regex invalid: %w", err)
	}

	targetProc := strings.ToLower(spec.ProcessName)

	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !windowMatchOnce(spec, titleRe, targetProc) {
			return nil // 窗口已消失
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w (title=%q class=%q process=%q)",
				ErrWindowStillPresent, spec.Title, spec.Class, spec.ProcessName)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// --- Win32 helpers (新版 API 专用) ---

func getWindowText(hwnd win.HWND) string {
	buf := make([]uint16, 512)
	n, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), 512)
	return syscall.UTF16ToString(buf[:n])
}

func getClassName(hwnd win.HWND) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), 256)
	return syscall.UTF16ToString(buf[:n])
}

func getWindowPID(hwnd win.HWND) uint32 {
	var pid uint32
	procGetWindowThreadProcId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}

// queryProcessName 用 PROCESS_QUERY_LIMITED_INFORMATION 跨权限读 exe basename.
// 不能用 PROCESS_QUERY_INFORMATION (高权限游戏会 ERROR_ACCESS_DENIED).
func queryProcessName(pid uint32) (string, error) {
	full, err := queryProcessPath(pid)
	if err != nil {
		return "", err
	}
	return filepath.Base(full), nil
}

func queryProcessPath(pid uint32) (string, error) {
	if pid == 0 {
		return "", errors.New("pid zero")
	}
	h, _, err := procOpenProcess.Call(uintptr(processQueryLimitedInformation), 0, uintptr(pid))
	if h == 0 {
		return "", fmt.Errorf("OpenProcess pid=%d: %v", pid, err)
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	r, _, err := procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 {
		return "", fmt.Errorf("QueryFullProcessImageName pid=%d: %v", pid, err)
	}
	// 防御性 clamp — 按 MSDN 不该发生, 但 size 是 in/out 参数, 异常返回若 > cap 会 panic
	if size > uint32(len(buf)) {
		size = uint32(len(buf))
	}
	return filepath.Clean(syscall.UTF16ToString(buf[:size])), nil
}

// ResolveUniqueExecutableWindow resolves one visible top-level window owned by
// the exact installed executable. The title uses the explicitly installed
// exact/regex mode; class remains exact. Multiple matches fail instead of
// silently selecting by Z order.
func ResolveUniqueExecutableWindow(ctx context.Context, executable string, spec MatchSpec, timeout, interval time.Duration) (WindowHandle, error) {
	return ResolveExecutableWindow(ctx, executable, spec, "unique", timeout, interval)
}

// ResolveExecutableWindow applies an explicit multi-match policy. "unique"
// fails on ambiguity; "topmost" deterministically follows current Win32
// Z-order and is intended for dynamic multi-window applications.
func ResolveExecutableWindow(ctx context.Context, executable string, spec MatchSpec, selection string, timeout, interval time.Duration) (WindowHandle, error) {
	if ctx == nil || executable == "" || !filepath.IsAbs(executable) || timeout <= 0 || interval <= 0 {
		return WindowHandle{}, errors.New("invalid exact executable window selector")
	}
	if selection != "unique" && selection != "topmost" {
		return WindowHandle{}, errors.New("invalid executable window selection policy")
	}
	query, err := newExecutableWindowQuery(spec)
	if err != nil {
		return WindowHandle{}, err
	}
	configured, err := os.Stat(executable)
	if err != nil || !configured.Mode().IsRegular() {
		return WindowHandle{}, errors.Join(err, errors.New("installed executable is unavailable"))
	}
	deadline := time.Now().Add(timeout)
	query.configured = configured
	for {
		matches := enumExecutableWindows(query)
		if len(matches) == 1 || selection == "topmost" && len(matches) > 0 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			return WindowHandle{}, ErrWindowAmbiguous
		}
		if err := ctx.Err(); err != nil {
			return WindowHandle{}, err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return WindowHandle{}, ErrWindowNotFound
		}
		wait := min(interval, remaining)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return WindowHandle{}, ctx.Err()
		case <-timer.C:
		}
	}
}

// VerifyExecutableWindow revalidates a cached HWND against the exact installed
// executable and the installed title/class selector before each operation.
func VerifyExecutableWindow(handle uintptr, executable string, spec MatchSpec) (WindowHandle, error) {
	if handle == 0 || executable == "" || !filepath.IsAbs(executable) || !win.IsWindowVisible(win.HWND(handle)) {
		return WindowHandle{}, ErrWindowNotFound
	}
	query, err := newExecutableWindowQuery(spec)
	if err != nil {
		return WindowHandle{}, err
	}
	metadata, err := WindowMetadata(handle)
	if err != nil || !query.matchesTitle(metadata.Title) || spec.Class != "" && metadata.Class != spec.Class {
		return WindowHandle{}, ErrWindowNotFound
	}
	configured, err := os.Stat(executable)
	if err != nil {
		return WindowHandle{}, err
	}
	processPath, err := queryProcessPath(metadata.PID)
	if err != nil {
		return WindowHandle{}, err
	}
	processFile, err := os.Stat(processPath)
	if err != nil || !os.SameFile(configured, processFile) {
		return WindowHandle{}, ErrWindowNotFound
	}
	return metadata, nil
}

func enumExecutableWindows(query *executableWindowQuery) []WindowHandle {
	query.matches = query.matches[:0]
	withExecutableWindowQuery(query, func(state uintptr) {
		procEnumWindows.Call(enumExecutableWindowsCallback, state)
	})
	return query.matches
}

type executableWindowQuery struct {
	configured os.FileInfo
	title      string
	titleMatch string
	titleRegex *regexp.Regexp
	class      string
	matches    []WindowHandle
}

func newExecutableWindowQuery(spec MatchSpec) (*executableWindowQuery, error) {
	mode := spec.TitleMatch
	if mode == "" {
		mode = "exact"
	}
	if mode != "exact" && mode != "regex" {
		return nil, errors.New("invalid executable window title match mode")
	}
	var titleRegex *regexp.Regexp
	var err error
	if mode == "regex" && spec.Title != "" {
		titleRegex, err = regexp.Compile(spec.Title)
		if err != nil {
			return nil, fmt.Errorf("compile executable window title regex: %w", err)
		}
	}
	return &executableWindowQuery{title: spec.Title, titleMatch: mode, titleRegex: titleRegex, class: spec.Class}, nil
}

func (query *executableWindowQuery) matchesTitle(title string) bool {
	if query.title == "" {
		return true
	}
	if query.titleMatch == "regex" {
		return query.titleRegex != nil && query.titleRegex.MatchString(title)
	}
	return title == query.title
}

func withExecutableWindowQuery(query *executableWindowQuery, enumerate func(uintptr)) {
	state := nextExecutableWindowQuery.Add(1)
	executableWindowQueries.Store(state, query)
	defer executableWindowQueries.Delete(state)
	enumerate(state)
}

var nextExecutableWindowQuery atomic.Uintptr
var executableWindowQueries sync.Map

func executableWindowQueryFromState(state uintptr) *executableWindowQuery {
	query, ok := executableWindowQueries.Load(state)
	if !ok {
		return nil
	}
	return query.(*executableWindowQuery)
}

// A Windows callback allocation lives for the process lifetime. Reuse one
// callback and pass a temporary registry token through the synchronous lParam.
var enumExecutableWindowsCallback = syscall.NewCallback(func(hwnd win.HWND, state uintptr) uintptr {
	query := executableWindowQueryFromState(state)
	if query == nil {
		return 0
	}
	if !win.IsWindowVisible(hwnd) {
		return 1
	}
	windowTitle, windowClass := getWindowText(hwnd), getClassName(hwnd)
	if !query.matchesTitle(windowTitle) || query.class != "" && windowClass != query.class {
		return 1
	}
	pid := getWindowPID(hwnd)
	processPath, err := queryProcessPath(pid)
	if err != nil {
		return 1
	}
	processFile, err := os.Stat(processPath)
	if err != nil || !os.SameFile(query.configured, processFile) {
		return 1
	}
	cw, ch := getClientSize(hwnd)
	query.matches = append(query.matches, WindowHandle{
		HWND: uintptr(hwnd), Title: windowTitle, Class: windowClass,
		ProcessName: strings.ToLower(filepath.Base(processPath)), PID: pid, ClientW: cw, ClientH: ch,
	})
	return 1
})

func getClientSize(hwnd win.HWND) (int, int) {
	var rect win.RECT
	procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	return int(rect.Right - rect.Left), int(rect.Bottom - rect.Top)
}

// EnumTopWindows 枚举全部顶层可见窗口, 返完整元数据列表 (MCP list_windows 用).
// 与 ResolveWindow 同枚举/同跳过语义: 不可见窗口跳过; 进程名 query 失败 (admin/僵尸) 跳过.
func EnumTopWindows() []WindowHandle {
	var out []WindowHandle
	callback := syscall.NewCallback(func(hwnd win.HWND, _ uintptr) uintptr {
		if !win.IsWindowVisible(hwnd) {
			return 1 // continue
		}
		pid := getWindowPID(hwnd)
		procName, err := queryProcessName(pid)
		if err != nil {
			return 1
		}
		cw, ch := getClientSize(hwnd)
		out = append(out, WindowHandle{
			HWND:        uintptr(hwnd),
			Title:       getWindowText(hwnd),
			Class:       getClassName(hwnd),
			ProcessName: strings.ToLower(procName),
			PID:         pid,
			ClientW:     cw,
			ClientH:     ch,
		})
		return 1 // continue 枚下一个
	})
	procEnumWindows.Call(callback, 0)
	return out
}

// ClientSize 客户区像素尺寸 (录制等非 capture 路径用, 不依赖 capture 全局后端).
// GetClientRect 失败或窗口尺寸为 0 时返 error.
func ClientSize(handle uintptr) (int, int, error) {
	hwnd := win.HWND(handle)
	var rect win.RECT
	r, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if r == 0 {
		return 0, 0, fmt.Errorf("GetClientRect 失败")
	}
	w, h := int(rect.Right-rect.Left), int(rect.Bottom-rect.Top)
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("GetClientRect 失败或窗口尺寸为 0")
	}
	return w, h, nil
}

// ---------------------------------------------------------------------------
// Win32WindowTarget capture hotkey helpers (tools.CaptureForegroundWindow).
// 用户先把目标窗口切到前台, 然后我们查它的 hwnd + metadata 一次性返给前端填表.
// ---------------------------------------------------------------------------

// ForegroundWindow 当前前台窗口 hwnd. 返 0 = 无前台窗口 (罕见, 仅锁屏 / 切换瞬间).
func ForegroundWindow() uintptr {
	r, _, _ := procGetForegroundWindow.Call()
	return r
}

// WindowMetadata 按 hwnd 直接查 metadata (跳 EnumWindows). 给 capture hotkey 用 — 用户
// 已经把目标窗口置前, 我们只要查它的属性. ProcessName 已 lowercase.
func WindowMetadata(hwnd uintptr) (WindowHandle, error) {
	if hwnd == 0 {
		return WindowHandle{}, errors.New("hwnd 0")
	}
	winH := win.HWND(hwnd)
	title := getWindowText(winH)
	class := getClassName(winH)
	pid := getWindowPID(winH)
	procName, err := queryProcessName(pid)
	if err != nil {
		return WindowHandle{}, fmt.Errorf("queryProcessName: %w", err)
	}
	cw, ch := getClientSize(winH)
	return WindowHandle{
		HWND:        hwnd,
		Title:       title,
		Class:       class,
		ProcessName: strings.ToLower(procName),
		PID:         pid,
		ClientW:     cw,
		ClientH:     ch,
	}, nil
}

// WindowExecutable returns the executable path owned by one native window.
// It is an installation-authoring helper; callers must not persist the path in
// a Workflow or journal.
func WindowExecutable(hwnd uintptr) (string, error) {
	if hwnd == 0 {
		return "", errors.New("hwnd 0")
	}
	return queryProcessPath(getWindowPID(win.HWND(hwnd)))
}
