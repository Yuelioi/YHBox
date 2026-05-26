// cmd/wgc-debug: 独立 CLI 验 capture_wgc.dll 是否在本机工作.
// 不依赖 yhbox.exe / 不启 GUI / 不依赖 fishing 状态机. 1 秒出结果.
//
// 用法:
//   wgc-debug.exe                              # 自动找异环窗口, WGC 抓 5 帧
//   wgc-debug.exe -backend gdi                 # 切 GDI 对比 baseline
//   wgc-debug.exe -hwnd 0x10EFE                # 手动指定 hwnd
//   wgc-debug.exe -rounds 10 -interval 200     # 10 帧 200ms 间隔
//
// 输出:
//   - 每次 grab: 成功/失败, 帧尺寸, 耗时, 前 16 字节 hex, 黑像素占比
//   - 成功的帧保存为 wgc_debug_<i>.png 你眼睛看内容对不对
//   - WGC init warning (SetIsBorderRequired 在 Win10 老版本会 warn) 也打印
//
// 解决 "改一行代码就要重启 GUI 跑 fishing 看结果" 的浪费.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/locale"
	"yhbox/pkg/winutil"
	"yhbox/tools/fish"
)

func main() {
	backend := flag.String("backend", "wgc", "wgc / gdi")
	rounds := flag.Int("rounds", 5, "grab attempts")
	interval := flag.Int("interval", 500, "ms between grabs")
	hwndStr := flag.String("hwnd", "", "target HWND (decimal or 0xHEX, 空=自动找异环)")
	outDir := flag.String("outdir", ".", "PNG output dir")
	fishDetect := flag.Bool("fishdetect", false, "抓帧后跑 fish.Detector 报每个 slot 的 conf")
	tapF := flag.Bool("tapf", false, "模拟 yhbox states_fishing 流程: 抓帧 → tap F → sleep 300ms → 再抓 frame2 → detect frame2 上 NeedBait/StartFish (复现 '按 F 后 300ms 误判换饵' 现象)")
	flag.Parse()

	hwnd, err := resolveHwnd(*hwndStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hwnd 解析失败:", err)
		os.Exit(2)
	}

	// 参考: 对比 GetClientRect + GetWindowRect + ClientToScreen 跟 WGC frame 尺寸,
	// 看 client offset 在 frame 内位置 (这是 Rust 侧 compute_client_offset 一样的算法).
	var cr win.RECT
	win.GetClientRect(hwnd, &cr)
	var wr win.RECT
	win.GetWindowRect(hwnd, &wr)
	type pt struct{ X, Y int32 }
	var origin pt
	syscall.NewLazyDLL("user32.dll").NewProc("ClientToScreen").Call(
		uintptr(hwnd), uintptr(unsafe.Pointer(&origin)))
	offX := origin.X - wr.Left
	offY := origin.Y - wr.Top
	fmt.Printf("[geom] GetClientRect: %dx%d  GetWindowRect: %dx%d  client→screen=(%d,%d)  window=(%d,%d)  client_off=(%d,%d)\n",
		cr.Right, cr.Bottom,
		wr.Right-wr.Left, wr.Bottom-wr.Top,
		origin.X, origin.Y,
		wr.Left, wr.Top,
		offX, offY)

	if *backend == "wgc" {
		if err := capture.SetBackend(capture.BackendWGC); err != nil {
			fmt.Fprintln(os.Stderr, "[backend] SetBackend wgc 失败:", err)
			os.Exit(1)
		}
		fmt.Println("[backend] wgc")
		if w := capture.LastInitWarning; w != "" {
			fmt.Println("[backend] init warning:", w)
		}
	} else {
		fmt.Println("[backend] gdi (默认)")
	}

	var det *fish.Detector
	if *fishDetect {
		// NewDetector 前必须 LoadConfig 加载 locale templates (跟 main.go 启动期一致)
		if err := fish.LoadConfig(locale.Zh); err != nil {
			fmt.Fprintln(os.Stderr, "[fishdetect] fish.LoadConfig:", err)
			os.Exit(1)
		}
		var derr error
		det, derr = fish.NewDetector()
		if derr != nil {
			fmt.Fprintln(os.Stderr, "[fishdetect] NewDetector 失败:", derr)
			os.Exit(1)
		}
		fmt.Println("[fishdetect] enabled (locale=zh)")
	}

	successCount := 0
	for i := 0; i < *rounds; i++ {
		t0 := time.Now()
		img, err := capture.Frame(hwnd)
		elapsed := time.Since(t0)
		if err != nil {
			fmt.Printf("[%d] FAIL: %v (elapsed %v)\n", i, err, elapsed)
		} else {
			successCount++
			w := img.Rect.Dx()
			h := img.Rect.Dy()
			// 前 16 字节 hex (RGBA 顺序 — pkg/capture 内部已 BGRA→RGBA)
			n := 16
			if len(img.Pix) < n {
				n = len(img.Pix)
			}
			hexBuf := fmt.Sprintf("%x", img.Pix[:n])
			// 黑像素 (RGB 三通道全 0, alpha 忽略) 占比检测
			black := 0
			total := len(img.Pix) / 4
			for j := 0; j < len(img.Pix); j += 4 {
				if img.Pix[j] == 0 && img.Pix[j+1] == 0 && img.Pix[j+2] == 0 {
					black++
				}
			}
			fmt.Printf("[%d] OK: %dx%d (elapsed %v) pix[0..16]=%s black=%d/%d (%.1f%%)\n",
				i, w, h, elapsed, hexBuf, black, total, float64(black)*100/float64(total))
			out := fmt.Sprintf("%s/cap_%s_%d.png", *outDir, *backend, i)
			if f, ferr := os.Create(out); ferr == nil {
				_ = png.Encode(f, img)
				_ = f.Close()
				fmt.Printf("    saved: %s\n", out)
			}
			if *tapF {
				fmt.Println("    [tapf] input.Tap(F) ...")
				input.Tap(hwnd, "f", 50*time.Millisecond, 30*time.Millisecond)
				time.Sleep(300 * time.Millisecond)
				fmt.Println("    [tapf] 再抓 frame2 (按 F 后 300ms)")
				img2, err2 := capture.Frame(hwnd)
				if err2 != nil {
					fmt.Printf("    [tapf] frame2 grab fail: %v\n", err2)
				} else {
					out2 := fmt.Sprintf("%s/cap_%s_%d_afterF.png", *outDir, *backend, i)
					if f, ferr := os.Create(out2); ferr == nil {
						_ = png.Encode(f, img2)
						_ = f.Close()
						fmt.Printf("    [tapf] saved: %s\n", out2)
					}
					img = img2 // 让下面 detect 跑在 frame2 上
				}
			}
			if det != nil {
				slots := []struct {
					name string
					fn   func(*image.RGBA) (fish.Hit, bool)
				}{
					{"start_fish", det.StartFish},
					{"hook_icon", det.HookIconDim},
					{"need_bait", det.NeedBait},
					{"warehouse_full", det.WarehouseFull},
					{"fish_escape", det.FishEscape},
					{"result", det.Result},
					{"hook_text", det.HookText},
				}
				for _, sl := range slots {
					hit, ok := sl.fn(img)
					mark := "·"
					if ok {
						mark = "✓"
					}
					fmt.Printf("    %s %-16s conf=%.3f @ (%d,%d)\n",
						mark, sl.name, hit.Conf, hit.ClientX, hit.ClientY)
				}
			}
		}
		if i < *rounds-1 {
			time.Sleep(time.Duration(*interval) * time.Millisecond)
		}
	}
	fmt.Printf("\n[result] %d/%d 帧成功. backend=%s\n", successCount, *rounds, *backend)
	if successCount == 0 {
		os.Exit(3)
	}
}

func resolveHwnd(s string) (win.HWND, error) {
	if s == "" {
		targets := winutil.FindGame()
		if len(targets) == 0 {
			return 0, fmt.Errorf("自动找异环窗口失败 (要求 title 前缀 '异环' + class 'UnrealWindow'). 用 -hwnd 0xXXXX 手动指定")
		}
		t := targets[0]
		fmt.Printf("[hwnd] auto-found: 0x%X title=%q class=%q\n", uintptr(t.HWND), t.Title, t.Class)
		return t.HWND, nil
	}
	var v uint64
	var err error
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, err = strconv.ParseUint(s[2:], 16, 64)
	} else {
		v, err = strconv.ParseUint(s, 10, 64)
	}
	if err != nil {
		return 0, fmt.Errorf("hwnd %q parse: %w", s, err)
	}
	fmt.Printf("[hwnd] manual: 0x%X\n", v)
	return win.HWND(uintptr(v)), nil
}
