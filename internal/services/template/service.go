package template

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Service per-container template RPC 服务. 内部 map[containerID]*Store.
// 启动期 main.go 注入 dataRoot, 每个 container 第一次访问时 lazy create Store.
type Service struct {
	dataRoot string
	capture  CaptureAdapter
	mu       sync.Mutex
	stores   map[string]*Store // containerID → store
}

type CaptureAdapter interface {
	Capture() ([]byte, error) // 返 PNG bytes
}

func NewService(dataRoot string, capture CaptureAdapter) *Service {
	return &Service{
		dataRoot: dataRoot,
		capture:  capture,
		stores:   map[string]*Store{},
	}
}

func (s *Service) storeFor(containerID string) (*Store, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.stores[containerID]; ok {
		return st, nil
	}
	root := filepath.Join(s.dataRoot, "containers", containerID, "templates")
	st, err := NewStore(root)
	if err != nil {
		return nil, err
	}
	s.stores[containerID] = st
	return st, nil
}

// Save 用 RPC 入口. containerID 是必填参数. dataURL 必须 data:image/png;base64,...
func (s *Service) Save(containerID, key, dataURL, name, description string, recRes [2]int, region [4]float32) error {
	st, err := s.storeFor(containerID)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		return fmt.Errorf("dataURL must be data:image/png;base64,...")
	}
	pngData, err := base64.StdEncoding.DecodeString(dataURL[len("data:image/png;base64,"):])
	if err != nil {
		return fmt.Errorf("decode dataURL: %w", err)
	}
	_, err = st.Save(key, pngData, TemplateMeta{
		Name: name, Description: description,
		RecordedResolution: recRes, Region: region,
		Width:  int(region[2] * float32(recRes[0])),
		Height: int(region[3] * float32(recRes[1])),
	})
	return err
}

func (s *Service) List(containerID string) (map[string]TemplateMeta, error) {
	st, err := s.storeFor(containerID)
	if err != nil {
		return nil, err
	}
	return st.List(), nil
}

func (s *Service) Delete(containerID, key string) error {
	st, err := s.storeFor(containerID)
	if err != nil {
		return err
	}
	return st.Delete(key)
}

func (s *Service) ReadPngDataURL(containerID, key string) (string, error) {
	st, err := s.storeFor(containerID)
	if err != nil {
		return "", err
	}
	b, err := st.ReadPng(key)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b), nil
}

// Capture 截取游戏窗口当前帧 (跟 containerID 无关, 共享 capture backend).
func (s *Service) Capture() (string, error) {
	pngData, err := s.capture.Capture()
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData), nil
}
