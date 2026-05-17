package template

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"strings"
)

// v2 spec §1.5：本包降级为基础库（PNG IO / 哈希 / TemplateMeta 序列化），不再独立暴露 wails3 RPC。
// 容器内模板由 ContainerService 托管（Task 1.12）；库模板由 LibraryService 托管（Task 1.17）。
// 本文件原 *Service 方法保留作为这些上层 service 的内部依赖；main.go 已不再 Bind 该 service。

// CaptureProvider 抽象屏幕截图（runtime 注入；测试时可 nil）。
// 实现方应该返回当前游戏窗口 / 主屏的 png bytes。
type CaptureProvider interface {
	CapturePNG() ([]byte, int, int, error) // 返 png bytes + w + h
}

// Service wails3 RPC 入口。
type Service struct {
	store   *Store
	capture CaptureProvider
}

func NewService(store *Store, capture CaptureProvider) *Service {
	return &Service{store: store, capture: capture}
}

// List 全部模板 meta（key → meta）。
func (s *Service) List() map[string]TemplateMeta {
	return s.store.List()
}

// Get 单个 meta。
func (s *Service) Get(key string) (TemplateMeta, error) {
	m, ok := s.store.Get(key)
	if !ok {
		return TemplateMeta{}, fmt.Errorf("template %q not found", key)
	}
	return m, nil
}

// Save 通过 dataURL 保存。recordedResolution 调用方按当前截图分辨率填；
// region 是 ratio xywh。
func (s *Service) Save(
	key, dataURL, name, description string,
	recordedResolution [2]int,
	region [4]float32,
) error {
	pngData, err := decodeDataURL(dataURL)
	if err != nil {
		return err
	}
	w, h, err := pngDims(pngData)
	if err != nil {
		return err
	}
	_, err = s.store.Save(key, pngData, TemplateMeta{
		Name:               name,
		Description:        description,
		RecordedResolution: recordedResolution,
		Width:              w,
		Height:             h,
		Region:             region,
	})
	return err
}

// SaveRaw 测试 / 内部用：直接传 bytes 不走 dataURL 解码。
func (s *Service) SaveRaw(key string, pngData []byte, meta TemplateMeta) error {
	if meta.Width == 0 || meta.Height == 0 {
		w, h, err := pngDims(pngData)
		if err == nil {
			meta.Width, meta.Height = w, h
		}
	}
	_, err := s.store.Save(key, pngData, meta)
	return err
}

func (s *Service) Delete(key string) error {
	return s.store.Delete(key)
}

// UpdateMeta 仅更新名称/描述（不动 PNG / region / resolution / hash）。
// 前端模板库卡片"编辑"用。
func (s *Service) UpdateMeta(key, name, description string) error {
	cur, ok := s.store.Get(key)
	if !ok {
		return fmt.Errorf("template %q not found", key)
	}
	cur.Name = name
	cur.Description = description
	return s.store.UpdateMeta(key, cur)
}

// CaptureScreenshot 截当前屏 → dataURL。前端在截图弹窗里用。
func (s *Service) CaptureScreenshot() (string, error) {
	if s.capture == nil {
		return "", errors.New("capture provider 未注入")
	}
	pngData, _, _, err := s.capture.CapturePNG()
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData), nil
}

// ReadPngDataURL 把 key 对应的 png 转 dataURL 返给前端。
func (s *Service) ReadPngDataURL(key string) (string, error) {
	b, err := s.store.ReadPng(key)
	if err != nil {
		return "", fmt.Errorf("read template %q: %w", key, err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b), nil
}

func decodeDataURL(s string) ([]byte, error) {
	prefix := "data:image/png;base64,"
	if !strings.HasPrefix(s, prefix) {
		return nil, fmt.Errorf("dataURL 必须以 %q 开头", prefix)
	}
	b, err := base64.StdEncoding.DecodeString(s[len(prefix):])
	if err != nil {
		return nil, fmt.Errorf("dataURL base64 解码失败: %w", err)
	}
	if len(b) < 8 {
		return nil, errors.New("dataURL 太短")
	}
	return b, nil
}

func pngDims(b []byte) (int, int, error) {
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return 0, 0, fmt.Errorf("png decode: %w", err)
	}
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}
