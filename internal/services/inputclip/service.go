package inputclip

import (
	"bytes"
	"fmt"
	"time"

	"yotta/internal/services/asset"
)

// ClipSummary 列表项 — 不含 events, 给 UI list view.
type ClipSummary struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	DurationUs  uint64   `json:"durationUs"`
	CreatedAt   string   `json:"createdAt"`
	Meta        ClipMeta `json:"meta"`
	EventCount  int      `json:"eventCount"`
}

// Service wails3 RPC 入口 — clip CRUD 背后落全局 asset 库的 clip kind.
//
// clip 字节 (binary csbf, Encode 产物) → asset blob 池 (内容寻址去重);
// clip 元数据 (label/duration/meta...) 进 asset 记录的 Origin/Name + 一份完整 header 编码在 blob 里.
// clip.ID 即资产 GUID (录制侧生成 clip-<uuid>, 稳定唯一).
//
// 回放: ClipResolver.Resolve(guid) → Get 记录 → Read blob → Decode → *InputClip.
type Service struct {
	store *asset.Store
	emit  func(name string, data any)
}

// NewService 构造 asset-backed clip 服务. store = 全局 asset.Store.
func NewService(store *asset.Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetEmit(emit func(name string, data any)) { s.emit = emit }

func (s *Service) emitChanged() {
	if s.emit != nil {
		s.emit("clip:changed", map[string]any{})
	}
}

// Save 写盘 (新建或覆盖). clip.ID = GUID. 字节走 blob 池去重, 元数据进记录.
func (s *Service) Save(clip *InputClip) error {
	if clip.ID == "" {
		return fmt.Errorf("clip id 不能为空")
	}
	var buf bytes.Buffer
	if err := Encode(&buf, clip); err != nil {
		return fmt.Errorf("encode clip: %w", err)
	}
	sha, err := s.store.Blobs().Put(buf.Bytes())
	if err != nil {
		return fmt.Errorf("put clip blob: %w", err)
	}
	createdAt := clip.CreatedAt
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}
	rec := asset.AssetRecord{
		GUID:   clip.ID,
		Kind:   asset.KindClip,
		Name:   clip.Label,
		Tags:   clip.Tags,
		Origin: asset.Origin{Kind: "user"},
		Blob:   sha,
	}
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		rec.CreatedAt = t
	}
	if err := s.store.PutRecord(rec); err != nil {
		return fmt.Errorf("put clip record: %w", err)
	}
	s.emitChanged()
	return nil
}

// Get 拿单个 clip 完整数据 (含 events). 反序列化 blob.
func (s *Service) Get(id string) (*InputClip, error) {
	rec, ok := s.store.Get(id)
	if !ok || rec.Kind != asset.KindClip {
		return nil, fmt.Errorf("clip %q not found", id)
	}
	data, err := s.store.Blobs().Read(rec.Blob)
	if err != nil {
		return nil, fmt.Errorf("read clip blob %q: %w", id, err)
	}
	clip, err := Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode clip %q: %w", id, err)
	}
	return clip, nil
}

// List 列所有 clip 摘要 (不带 events, 列表视图用). 解 blob header 取 metadata.
func (s *Service) List() []ClipSummary {
	out := []ClipSummary{}
	for _, rec := range s.store.List() {
		if rec.Kind != asset.KindClip {
			continue
		}
		clip, err := s.Get(rec.GUID)
		if err != nil {
			continue
		}
		out = append(out, ClipSummary{
			ID: clip.ID, Label: clip.Label, Description: clip.Description,
			Tags: clip.Tags, DurationUs: clip.DurationUs, CreatedAt: clip.CreatedAt,
			Meta: clip.Meta, EventCount: len(clip.Events),
		})
	}
	return out
}

// Delete 删 clip 记录 (blob 由 GC 回收孤儿字节).
func (s *Service) Delete(id string) error {
	if err := s.store.DeleteRecord(id); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

// Resolve 满足 container/runtime.ClipResolver 接口. PlayClip 节点用.
func (s *Service) Resolve(id string) (*InputClip, bool) {
	clip, err := s.Get(id)
	if err != nil {
		return nil, false
	}
	return clip, true
}

// Update 改 metadata (label / description / tags) — 不改 events. 重编码 blob (header 含 metadata).
func (s *Service) Update(id string, label, description string, tags []string) error {
	clip, err := s.Get(id)
	if err != nil {
		return err
	}
	clip.Label = label
	clip.Description = description
	clip.Tags = tags
	if err := s.Save(clip); err != nil {
		return err
	}
	return nil
}
