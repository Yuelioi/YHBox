// wire_container.go 适配器：容器节点运行时跟 services / pkg 类型对接。
//
// container/runtime 包定义了 Matcher / Color / InputDriver / Runner 等接口；
// 这里把具体实现（pkg/vision + pkg/capture + pkg/input + actionsruntime.Win32Driver）
// 绑到接口上。
package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lxn/win"

	"yhbox/internal/hotkey"
	"yhbox/internal/services"
	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
	"yhbox/internal/services/template"
	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/vision"
)

// ---- TemplateMatcher: 接 pkg/vision + template store + capture ----
//
// 容器节点 WaitTemplate / CheckTemplate / ClickTemplate 用：
//   - 从 template store 读 png bytes（key = 模板路径）
//   - 抓当前游戏窗口一帧
//   - 用 pkg/vision.Match 在 ROI / 全屏内做 CCOEFF_NORMED 匹配
//   - 返 found + 命中比例坐标 + 实际 region
//
// 命中坐标转换：vision.Match 返 ROI 内左上角像素 → 转客户区比例。

type templateMatcherAdapter struct {
	app       *services.App
	tplStore  *template.Store
	loadCache sync.Map // key string → *vision.Template
}

func (m *templateMatcherAdapter) loadTemplate(key string) (*vision.Template, error) {
	if cached, ok := m.loadCache.Load(key); ok {
		return cached.(*vision.Template), nil
	}
	data, err := m.tplStore.ReadPng(key)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", key, err)
	}
	tpl, err := vision.LoadPNG(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode template %q: %w", key, err)
	}
	m.loadCache.Store(key, tpl)
	return tpl, nil
}

func (m *templateMatcherAdapter) Detect(_ context.Context, key string, threshold float64, region []float64) (bool, expr.Point, [4]float64, error) {
	g := m.app.Game()
	if g == nil || !g.OK {
		return false, expr.Point{}, [4]float64{}, nil
	}
	tpl, err := m.loadTemplate(key)
	if err != nil {
		return false, expr.Point{}, [4]float64{}, err
	}
	frame, err := capture.Frame(win.HWND(g.HWND))
	if err != nil {
		return false, expr.Point{}, [4]float64{}, fmt.Errorf("capture.Frame: %w", err)
	}

	// region 优先级：caller 传的 > template meta 的 Region > 全屏（兜底）。
	// 全屏匹配开销巨大（1080p 上 400×40 模板要 3-5s/match），用 ROI 后 ~10ms。
	rx, ry, rw, rh := 0.0, 0.0, 1.0, 1.0
	if len(region) == 4 && (region[2] > 0 || region[3] > 0) {
		rx, ry, rw, rh = region[0], region[1], region[2], region[3]
	} else if meta, ok := m.tplStore.Get(key); ok && (meta.Region[2] > 0 || meta.Region[3] > 0) {
		// 模板录制 bbox → meta.Region，作为搜索 ROI；扩 30% 边距防 UI 漂移
		mx, my, mw, mh := float64(meta.Region[0]), float64(meta.Region[1]), float64(meta.Region[2]), float64(meta.Region[3])
		padX, padY := mw*0.3, mh*0.3
		rx = clamp01(mx - padX)
		ry = clamp01(my - padY)
		rw = clamp01Bound(mw+padX*2, rx)
		rh = clamp01Bound(mh+padY*2, ry)
	}

	roiGray, roiPxX, roiPxY, roiPxW, roiPxH := vision.CropROI(frame, rx, ry, rw, rh)
	if roiPxW <= 0 || roiPxH <= 0 {
		return false, expr.Point{}, [4]float64{}, nil
	}
	x, y, conf := vision.Match(roiGray, roiPxW, roiPxH, tpl, vision.DefaultParallel())
	if x < 0 || conf < float32(threshold) {
		return false, expr.Point{}, [4]float64{}, nil
	}
	frameW := frame.Bounds().Dx()
	frameH := frame.Bounds().Dy()
	cx := float64(roiPxX+x+tpl.W/2) / float64(frameW)
	cy := float64(roiPxY+y+tpl.H/2) / float64(frameH)
	out := [4]float64{rx, ry, rw, rh}
	return true, expr.Point{X: cx, Y: cy}, out, nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// clamp01Bound 限制 w 不超过剩余空间（x + w <= 1）。
func clamp01Bound(w, x float64) float64 {
	if x+w > 1 {
		return 1 - x
	}
	if w < 0 {
		return 0
	}
	return w
}

// ---- ColorDetector: capture.Frame + HSV/RGB 像素扫描 ----
//
// DetectColor 节点用。抠 ROI（客户区比例 → 像素）→ 遍历像素 → 返命中数 +
// 命中中心客户区比例坐标。HSV 用 pkg/vision.RGBToHSV（H 0-360, S/V 0-255）。

type containerColorAdapter struct {
	app *services.App
}

func (c *containerColorAdapter) Detect(_ context.Context, region [4]float64, mode string, rng [6]int) (int, float64, float64, error) {
	g := c.app.Game()
	if g == nil || !g.OK {
		return 0, 0, 0, fmt.Errorf("游戏窗口未就绪")
	}
	hwnd := win.HWND(g.HWND)
	frame, err := capture.Frame(hwnd)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("capture.Frame: %w", err)
	}

	frameW := frame.Bounds().Dx()
	frameH := frame.Bounds().Dy()
	rx, ry, rw, rh := region[0], region[1], region[2], region[3]
	if rw == 0 || rh == 0 {
		rx, ry, rw, rh = 0, 0, 1, 1
	}
	x0 := int(rx * float64(frameW))
	y0 := int(ry * float64(frameH))
	x1 := x0 + int(rw*float64(frameW))
	y1 := y0 + int(rh*float64(frameH))
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > frameW {
		x1 = frameW
	}
	if y1 > frameH {
		y1 = frameH
	}
	if x1 <= x0 || y1 <= y0 {
		return 0, 0, 0, nil
	}

	useHSV := mode != "rgb"
	count := 0
	sumX, sumY := 0, 0
	stride := frame.Stride

	for y := y0; y < y1; y++ {
		off := y * stride
		for x := x0; x < x1; x++ {
			i := off + x*4
			r, gC, b := frame.Pix[i], frame.Pix[i+1], frame.Pix[i+2]
			hit := false
			if useHSV {
				hh, ss, vv := vision.RGBToHSV(r, gC, b)
				hit = hh >= rng[0] && hh <= rng[1] && ss >= rng[2] && ss <= rng[3] && vv >= rng[4] && vv <= rng[5]
			} else {
				hit = int(r) >= rng[0] && int(r) <= rng[1] && int(gC) >= rng[2] && int(gC) <= rng[3] && int(b) >= rng[4] && int(b) <= rng[5]
			}
			if hit {
				count++
				sumX += x
				sumY += y
			}
		}
	}
	if count == 0 {
		return 0, 0, 0, nil
	}
	cxPx := float64(sumX) / float64(count)
	cyPx := float64(sumY) / float64(count)
	return count, cxPx / float64(frameW), cyPx / float64(frameH), nil
}

// ---- InputDriver: Container 输入原语 → pkg/input Win32 路径 ----
//
// ClickAt / KeyPress / MouseMoveRel / Scroll / ClickTemplate.click 阶段调这里。
// 每次操作内查一次游戏窗口 hwnd + 客户区尺寸；hwnd 缺失则 error。

// containerWin32Driver 封装 pkg/input 原语，满足 containerInputDriver 内部使用。
// Task 1.22 会把这部分移到独立 pkg 或统一 InputDriver 包；暂时内联这里。
type containerWin32Driver struct {
	activateDelay     time.Duration
	cursorSettleDelay time.Duration
}

func (d *containerWin32Driver) click(hwnd win.HWND, x, y int, button input.MouseButton, holdMs int) error {
	input.ClickButtonNoRestore(hwnd, x, y, button, time.Duration(holdMs)*time.Millisecond, d.activateDelay, d.cursorSettleDelay)
	return nil
}

func (d *containerWin32Driver) keyDown(hwnd win.HWND, vk string) error {
	if !input.KeyDown(hwnd, vk) {
		return fmt.Errorf("unknown vk %q", vk)
	}
	return nil
}

func (d *containerWin32Driver) keyUp(hwnd win.HWND, vk string) error {
	if !input.KeyUp(hwnd, vk) {
		return fmt.Errorf("unknown vk %q", vk)
	}
	return nil
}

func (d *containerWin32Driver) mouseScroll(hwnd win.HWND, notches int) error {
	input.MouseScroll(hwnd, notches, d.activateDelay)
	return nil
}

func (d *containerWin32Driver) mouseMoveRel(hwnd win.HWND, dx, dy int, durationMs int) error {
	input.MouseMoveRel(hwnd, dx, dy, time.Duration(durationMs)*time.Millisecond, d.activateDelay)
	return nil
}

type containerInputDriver struct {
	app    *services.App
	driver *containerWin32Driver
}

func newContainerInputDriver(app *services.App) *containerInputDriver {
	return &containerInputDriver{
		app: app,
		driver: &containerWin32Driver{
			activateDelay:     30 * time.Millisecond,
			cursorSettleDelay: 20 * time.Millisecond,
		},
	}
}

func (d *containerInputDriver) hwndOrErr() (win.HWND, int, int, error) {
	g := d.app.Game()
	if g == nil || !g.OK {
		return 0, 0, 0, fmt.Errorf("游戏窗口未就绪")
	}
	hwnd := win.HWND(g.HWND)
	w, h, err := capture.ClientSize(hwnd)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("capture.ClientSize: %w", err)
	}
	return hwnd, w, h, nil
}

func (d *containerInputDriver) Click(_ context.Context, xR, yR float64, button string, durMs int) error {
	hwnd, w, h, err := d.hwndOrErr()
	if err != nil {
		return err
	}
	x := int(xR * float64(w))
	y := int(yR * float64(h))
	return d.driver.click(hwnd, x, y, mouseButtonFromString(button), durMs)
}

func (d *containerInputDriver) KeyPress(ctx context.Context, vk string, durMs int) error {
	hwnd, _, _, err := d.hwndOrErr()
	if err != nil {
		return err
	}
	if err := d.driver.keyDown(hwnd, vk); err != nil {
		return err
	}
	// defer KeyUp 保证任意分支退出（含 panic）都释放按键。
	defer func() { _ = d.driver.keyUp(hwnd, vk) }()
	if durMs > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(durMs) * time.Millisecond):
		}
	}
	return nil
}

func (d *containerInputDriver) MouseMoveRel(_ context.Context, dx, dy, durMs int) error {
	hwnd, _, _, err := d.hwndOrErr()
	if err != nil {
		return err
	}
	return d.driver.mouseMoveRel(hwnd, dx, dy, durMs)
}

func (d *containerInputDriver) Scroll(_ context.Context, _, _ float64, delta int) error {
	hwnd, _, _, err := d.hwndOrErr()
	if err != nil {
		return err
	}
	return d.driver.mouseScroll(hwnd, delta)
}

func mouseButtonFromString(s string) input.MouseButton {
	switch s {
	case "middle":
		return input.MouseMiddle
	case "right":
		return input.MouseRight
	}
	return input.MouseLeft
}

// ---- Container Runner: ExecutionQueue + Worker → container.Runner ----
//
// container.Service 暴露给前端的 Run / StopAll 走这条；source=manual。

type containerRunnerAdapter struct {
	queue  *execution.ExecutionQueue
	worker *execution.Worker
}

func (a *containerRunnerAdapter) RunOnce(id string) error {
	_, ok := a.queue.Enqueue(execution.QueuedRun{
		Targets: []execution.TargetRef{{Kind: "container", ID: id}},
		OnError: execution.OnErrorStop,
		Source:  execution.SourceManual,
	})
	if !ok {
		return fmt.Errorf("execution queue closed")
	}
	return nil
}

func (a *containerRunnerAdapter) StopAll() error {
	a.queue.CancelAll()
	a.worker.CancelCurrent()
	return nil
}

// ---- Container hotkey binder ----
//
// 把 container.Hotkey（用户在 ContainerEditor 底栏配的字符串）注册到 hotkey
// registry。按下后 enqueue 单 target manual run。CRUD 后 Refresh()：
// unregister 所有旧 entries → 重扫 container store → 注册当前列表。

type containerHotkeyBinder struct {
	store    *container.Store
	registry *hotkey.HotkeyRegistry
	queue    *execution.ExecutionQueue
	mu       sync.Mutex
	bound    map[string]string // containerID → registry key
}

func newContainerHotkeyBinder(store *container.Store, reg *hotkey.HotkeyRegistry, q *execution.ExecutionQueue) *containerHotkeyBinder {
	return &containerHotkeyBinder{store: store, registry: reg, queue: q, bound: map[string]string{}}
}

func (b *containerHotkeyBinder) Refresh() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, key := range b.bound {
		_ = b.registry.Unregister(key)
	}
	b.bound = map[string]string{}
	for _, c := range b.store.List() {
		hk := strings.TrimSpace(c.Hotkey)
		if hk == "" {
			continue
		}
		key := "container." + c.ID
		cid := c.ID
		err := b.registry.Register(key, hotkey.HotkeySourceContainer, "容器 "+c.Name, hk, "",
			func() {
				_, _ = b.queue.Enqueue(execution.QueuedRun{
					Targets: []execution.TargetRef{{Kind: "container", ID: cid}},
					OnError: execution.OnErrorStop,
					Source:  execution.SourceHotkey,
				})
			})
		if err == nil {
			b.bound[c.ID] = key
		}
	}
}
