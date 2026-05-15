// rhythm-calibrate 独立 CLI 标定脚本（亮度版）。
//
// 用法：
//
//  1. 启动异环，进入任意超强音曲目（曲目正在 play，音符在掉）
//  2. 切回 PowerShell，cd 项目根目录
//  3. go run ./cmd/rhythm-calibrate
//  4. 倒数 3 秒内 alt-tab 回游戏
//  5. 等 30 秒，脚本结束输出 CSV 路径 + 每轨 dark_ratio 直方图
//  6. 肉眼看 ROI 截图位置对不对；看 CSV 跟阈值 (DarkRatioThreshold=0.06) 的差距
//     - 多数 tick ratio 接近 0（背景亮）✓
//     - 音符进入时 ratio 跳到 0.5+ ✓
//     - 边缘平均 ratio 在 0.06 附近 → 阈值偏高/偏低，回 constants.go 调
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/lxn/win"

	"yhbox/pkg/capture"
	"yhbox/pkg/locale"
	"yhbox/pkg/platform"
	"yhbox/pkg/winutil"
	"yhbox/tools/rhythm"
)

const calibrateDuration = 30 * time.Second

func main() {
	if err := rhythm.LoadConfig(locale.Zh); err != nil {
		fmt.Fprintf(os.Stderr, "❌ rhythm 配置加载失败: %v\n", err)
		os.Exit(1)
	}

	targets := winutil.FindGame()
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "❌ 没找到异环窗口")
		os.Exit(1)
	}
	t := targets[0]
	hwnd := win.HWND(uintptr(t.HWND))

	w, h, err := capture.ClientSize(hwnd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ ClientSize: %v\n", err)
		os.Exit(1)
	}
	cfg, ok := rhythm.PickConfig(w, h)
	if !ok {
		fmt.Fprintf(os.Stderr, "❌ 不支持的分辨率 %dx%d（仅 1920×1080 / 1280×720）\n", w, h)
		os.Exit(1)
	}
	fmt.Printf("✓ 找到异环窗口: HWND=0x%X  客户区 %dx%d\n", uintptr(t.HWND), w, h)

	fmt.Println()
	fmt.Println("准备 30s 标定。请进入任意超强音曲目，曲目正在 play 中。")
	for i := 3; i >= 1; i-- {
		fmt.Printf("  %d...\n", i)
		time.Sleep(1 * time.Second)
	}
	fmt.Println("开始抓帧 + 写 CSV。")

	hires := platform.NewHighResTimer()
	if err := hires.Begin(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: HighResTimer 失败 %v（tick 可能不准）\n", err)
	}
	defer hires.End()

	cp, err := capture.NewCapturer(hwnd, cfg.ROIs[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Capturer: %v\n", err)
		os.Exit(1)
	}
	defer cp.Close()

	csvW, err := rhythm.NewCSVWriter(platform.DevOutputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ CSV: %v\n", err)
		os.Exit(1)
	}
	defer csvW.Close()

	ctx, cancel := context.WithTimeout(context.Background(), calibrateDuration)
	defer cancel()

	ticker := time.NewTicker(rhythm.TickInterval)
	defer ticker.Stop()

	// 4 轨 dark_ratio 历史，结束后算统计
	hist := [4][]float32{}

	var ratios [4]float32
	var pressed [4]bool // 始终 false，标定不实际按键
	dumped := false     // 第一帧 dump ROI 截图供肉眼验证

	for {
		select {
		case <-ctx.Done():
			goto done
		case now := <-ticker.C:
			imgs, err := cp.Frame()
			if err != nil {
				continue
			}
			if !dumped {
				for i, img := range imgs {
					path := platform.DevOutputPath(fmt.Sprintf("rhythm-roi-T%d.png", i+1))
					if err := rhythm.SavePNG(path, img); err != nil {
						fmt.Fprintf(os.Stderr, "warn: save %s: %v\n", path, err)
					} else {
						fmt.Printf("  ROI 截图: %s  (%dx%d)\n", path, img.Bounds().Dx(), img.Bounds().Dy())
					}
				}
				dumped = true
			}
			rhythm.DarkRatios(imgs, &ratios)
			for i := 0; i < 4; i++ {
				hist[i] = append(hist[i], ratios[i])
			}
			_ = csvW.Write(now, ratios, pressed)
		}
	}

done:
	fmt.Println()
	fmt.Printf("✓ 标定结束。CSV: %s\n", csvW.Path())
	fmt.Println()
	fmt.Println("每轨 dark_ratio 分布：")
	fmt.Printf("  %-8s %-8s %-8s %-8s %-8s %-8s\n", "track", "min", "p50", "p95", "p99", "max")
	for i := 0; i < 4; i++ {
		if len(hist[i]) == 0 {
			continue
		}
		xs := append([]float32{}, hist[i]...)
		sort.Slice(xs, func(a, b int) bool { return xs[a] < xs[b] })
		stat := func(p int) float32 { return xs[(len(xs)-1)*p/100] }
		fmt.Printf("  %-8s %-8.3f %-8.3f %-8.3f %-8.3f %-8.3f\n",
			rhythm.TrackNames[i], xs[0], stat(50), stat(95), stat(99), xs[len(xs)-1])
	}
	fmt.Println()
	fmt.Printf("当前阈值：DarkRatioThreshold = %.2f\n", rhythm.DarkRatioThreshold)
	fmt.Println("调参建议：")
	fmt.Println("  - p50 应该 << 阈值 (背景多数时间没音符，ratio 接近 0)")
	fmt.Println("  - p95/p99 应该 >> 阈值 (音符出现的 tick，ratio 0.5+)")
	fmt.Println("  - p95 跟阈值差太近 → 阈值偏高，调低让边缘音符也触发")
	fmt.Println("  - p50 接近阈值 → 阈值偏低，调高免得静止背景误触发")
}
