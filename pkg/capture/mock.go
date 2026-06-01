// Mock backend: 从磁盘读 PNG 序列伪装成截屏帧源，给离线调参 / 回放调试用。
//
// 场景：开发者想调 fish 的 HSV 阈值 / rhythm 的 dark_ratio 阈值。开真游戏
// 每轮要 alt-tab 5-10 分钟。改成 mock backend 后：
//   1. 真游戏跑一次，开启 settings.capture.debug.dump 录一段 PNG 序列到
//      `bin/captures/<bot>/<date>/`
//   2. 把该容器 WindowTarget 节点的 CaptureBackend 设成 "mock" (per-container,
//      settings.capture.method 现在只作新建容器的默认值, 不再全局生效),
//      可选设环境变量 YHBOX_MOCK_DIR 指向那个目录
//   3. 启动 bot，detect 拿到的就是录制好的帧，调参立即可见
//
// 跟 hwnd 无关——mock 忽略 hwnd 参数，只看磁盘 PNG。mockBackend.Frame 也一样。

package capture

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/lxn/win"
)

// 包级状态。Init 时一次性加载 mock 目录里所有 PNG 进 frames。Frame() 调用 advance
// cursor 模拟"时间往前走一帧"。
var (
	mockOnce      sync.Once
	mockLoadErr   error
	mockFrames    []*image.RGBA // 整张 frame，按文件名排序
	mockCursorPkg atomic.Int64  // 包级 Frame/FrameROI 用的游标
)

// MockDir 是 mock backend 读 PNG 的目录。优先级：
//  1. 环境变量 YHBOX_MOCK_DIR
//  2. exe 同目录 mock-frames/
//
// 没找到目录或目录空 → SetBackend(BackendMock) 失败，main.go 回退 GDI。
func mockDir() string {
	if env := os.Getenv("YHBOX_MOCK_DIR"); env != "" {
		return env
	}
	exe, err := os.Executable()
	if err != nil {
		return "mock-frames"
	}
	return filepath.Join(filepath.Dir(exe), "mock-frames")
}

func initMock() error {
	mockOnce.Do(func() {
		dir := mockDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			mockLoadErr = fmt.Errorf("mock 目录 %s 读取失败: %w", dir, err)
			return
		}
		var paths []string
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
				continue
			}
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
		sort.Strings(paths) // 文件名排序保证回放顺序稳定
		if len(paths) == 0 {
			mockLoadErr = fmt.Errorf("mock 目录 %s 无 PNG 文件，至少放一张", dir)
			return
		}
		mockFrames = make([]*image.RGBA, 0, len(paths))
		for _, p := range paths {
			img, err := readPNGAsRGBA(p)
			if err != nil {
				mockLoadErr = fmt.Errorf("解码 %s 失败: %w", p, err)
				return
			}
			mockFrames = append(mockFrames, img)
		}
	})
	return mockLoadErr
}

// readPNGAsRGBA 读 PNG 文件返回 *image.RGBA（统一格式，避免 sub-image 不连续 stride）。
func readPNGAsRGBA(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	// 拷贝逐像素让结果一定是 standard RGBA（不论 src 是 NRGBA / Paletted / etc）
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, a := src.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			i := y*dst.Stride + x*4
			dst.Pix[i+0] = uint8(r >> 8)
			dst.Pix[i+1] = uint8(g >> 8)
			dst.Pix[i+2] = uint8(b >> 8)
			dst.Pix[i+3] = uint8(a >> 8)
		}
	}
	return dst, nil
}

// mockFrame 取当前 cursor 指向的 frame 返回 deep copy（避免 caller 写 mutates 缓存）。
func mockFrame(_ win.HWND) (*image.RGBA, error) {
	if err := initMock(); err != nil {
		return nil, err
	}
	idx := mockCursorPkg.Add(1) - 1
	src := mockFrames[idx%int64(len(mockFrames))]
	return cloneRGBA(src), nil
}

// mockFrameROI 取当前 frame 的指定区域返回 deep copy。
func mockFrameROI(_ win.HWND, roiX, roiY, roiW, roiH int) (*image.RGBA, error) {
	if err := initMock(); err != nil {
		return nil, err
	}
	idx := mockCursorPkg.Add(1) - 1
	src := mockFrames[idx%int64(len(mockFrames))]
	// 裁 ROI 到 src 边界内
	srcW, srcH := src.Bounds().Dx(), src.Bounds().Dy()
	if roiX < 0 {
		roiW += roiX
		roiX = 0
	}
	if roiY < 0 {
		roiH += roiY
		roiY = 0
	}
	if roiX+roiW > srcW {
		roiW = srcW - roiX
	}
	if roiY+roiH > srcH {
		roiH = srcH - roiY
	}
	if roiW <= 0 || roiH <= 0 {
		return nil, fmt.Errorf("ROI 无效")
	}
	dst := image.NewRGBA(image.Rect(0, 0, roiW, roiH))
	for y := 0; y < roiH; y++ {
		copy(dst.Pix[y*dst.Stride:], src.Pix[(roiY+y)*src.Stride+roiX*4:(roiY+y)*src.Stride+(roiX+roiW)*4])
	}
	return dst, nil
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

// mockClientSize 给 ClientSize 调用用：返回第一帧的尺寸（mock 假设所有帧同分辨率）。
func mockClientSize(_ win.HWND) (int, int, error) {
	if err := initMock(); err != nil {
		return 0, 0, err
	}
	first := mockFrames[0]
	return first.Bounds().Dx(), first.Bounds().Dy(), nil
}
