package inputclip

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
)

// ClipSummary 列表项 — 不含 events, 给 UI list view.
type ClipSummary struct {
	ID          string       `json:"id"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Category    string       `json:"category,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	DurationUs  uint64       `json:"durationUs"`
	CreatedAt   string       `json:"createdAt"`
	Meta        ClipMeta     `json:"meta"`
	EventCount  int          `json:"eventCount"`
	Blob        blob.BlobRef `json:"blob"`
	Tracks      []EventTrack `json:"tracks"`
}

type EventTrack struct {
	Kind    string `json:"kind"`
	Count   int    `json:"count"`
	FirstUs uint64 `json:"firstUs"`
	LastUs  uint64 `json:"lastUs"`
}

const maxEventPageSize = 200

type EventPage struct {
	Items  []Event `json:"items"`
	Total  int     `json:"total"`
	Offset int     `json:"offset"`
	Limit  int     `json:"limit"`
}

// Service wails3 RPC 入口 — clip CRUD 背后落全局 asset 库的 clip kind.
//
// clip 字节 (binary carrier, Encode 产物) → 共享 blob store (内容寻址去重);
// 可变展示元数据只进 asset 记录；carrier 只保留回放所需的不可变内容.
// clip.ID 即资产 GUID (录制侧生成 clip-<uuid>, 稳定唯一).
//
// 工作流通过资产暴露的 nominal InputClip BlobRef 直接读取 carrier，不解析 GUID.
type Service struct {
	store *asset.Store
	emit  func(name string, data any)
}

// NewService 构造 asset-backed clip 服务. store = 全局 asset.Store.
func NewService(store *asset.Store, emit ...func(name string, data any)) *Service {
	service := &Service{store: store}
	if len(emit) != 0 {
		service.emit = emit[0]
	}
	return service
}

func (s *Service) emitChangedSince(before uint64) {
	revision := s.store.Revision()
	if s.emit == nil || revision == before {
		return
	}
	payload := map[string]any{"revision": revision}
	s.emit("asset:changed", payload)
	s.emit("clip:changed", payload)
}

// Save 写盘 (新建或覆盖). clip.ID = GUID. 字节走 blob 池去重, 元数据进记录.
func (s *Service) Save(clip *InputClip) error {
	if clip.ID == "" {
		return fmt.Errorf("clip id 不能为空")
	}
	before := s.store.Revision()
	defer s.emitChangedSince(before)
	var buf bytes.Buffer
	if err := Encode(&buf, clip); err != nil {
		return fmt.Errorf("encode clip: %w", err)
	}
	if _, err := Decode(bytes.NewReader(buf.Bytes())); err != nil {
		return fmt.Errorf("verify encoded clip: %w", err)
	}
	createdAt := clip.CreatedAt
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}
	rec := asset.AssetRecord{
		GUID:        clip.ID,
		Kind:        asset.KindClip,
		Name:        clip.Label,
		Description: clip.Description,
		Category:    clip.Category,
		Tags:        clip.Tags,
		Origin:      asset.Origin{Kind: "user"},
	}
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		rec.CreatedAt = t
	}
	ref, err := s.store.CommitRecordBlob(context.Background(), MediaType, bytes.NewReader(buf.Bytes()), func(ref blob.BlobRef) asset.AssetRecord {
		rec.Blob = &ref
		return rec
	})
	if err != nil {
		return fmt.Errorf("commit clip blob: %w", err)
	}
	clip.Blob = ref
	return nil
}

// Get decodes the carrier for metadata and internal consumers. Event bytes are
// intentionally excluded from ordinary JSON; EventPage exposes a bounded
// diagnostic window for the precise-recording workbench.
func (s *Service) Get(id string) (*InputClip, error) {
	rec, ok := s.store.Get(id)
	if !ok || rec.Kind != asset.KindClip {
		return nil, fmt.Errorf("clip %q not found", id)
	}
	if rec.Blob == nil {
		return nil, fmt.Errorf("clip %q has no blob reference", id)
	}
	data, err := s.store.ReadBlob(context.Background(), *rec.Blob)
	if err != nil {
		return nil, fmt.Errorf("read clip blob %q: %w", id, err)
	}
	clip, err := Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode clip %q: %w", id, err)
	}
	clip.ID = rec.GUID
	clip.Label = rec.Name
	clip.Description = rec.Description
	clip.Category = rec.Category
	clip.Tags = append([]string(nil), rec.Tags...)
	clip.CreatedAt = rec.CreatedAt.UTC().Format(time.RFC3339)
	clip.Blob = *rec.Blob
	return clip, nil
}

func (s *Service) Events(id string, offset, limit int) (EventPage, error) {
	if offset < 0 || limit <= 0 || limit > maxEventPageSize {
		return EventPage{}, fmt.Errorf("clip event page is outside the bounded range")
	}
	clip, err := s.Get(id)
	if err != nil {
		return EventPage{}, err
	}
	total := len(clip.Events)
	start := min(offset, total)
	end := min(start+limit, total)
	return EventPage{
		Items: append([]Event(nil), clip.Events[start:end]...),
		Total: total, Offset: start, Limit: limit,
	}, nil
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
		out = append(out, clipSummary(clip))
	}
	return out
}

func (s *Service) Summary(id string) (ClipSummary, error) {
	clip, err := s.Get(id)
	if err != nil {
		return ClipSummary{}, err
	}
	return clipSummary(clip), nil
}

func clipSummary(clip *InputClip) ClipSummary {
	return ClipSummary{
		ID: clip.ID, Label: clip.Label, Description: clip.Description, Category: clip.Category,
		Tags: clip.Tags, DurationUs: clip.DurationUs, CreatedAt: clip.CreatedAt,
		Meta: clip.Meta, EventCount: len(clip.Events), Blob: clip.Blob, Tracks: eventTracks(clip.Events),
	}
}

func eventTracks(events []Event) []EventTrack {
	tracks := map[string]*EventTrack{}
	for _, event := range events {
		kind := ""
		switch event.Type {
		case EventTypeKeyDown, EventTypeKeyUp:
			kind = "keyboard"
		case EventTypeMouseBtnDown, EventTypeMouseBtnUp:
			kind = "mouse-buttons"
		case EventTypeMouseMove:
			kind = "absolute-motion"
		case EventTypeRawDelta:
			kind = "relative-motion"
		case EventTypeScroll:
			kind = "scroll"
		}
		if kind == "" {
			continue
		}
		track := tracks[kind]
		if track == nil {
			track = &EventTrack{Kind: kind, FirstUs: event.TUs}
			tracks[kind] = track
		}
		track.Count++
		track.LastUs = event.TUs
	}
	result := make([]EventTrack, 0, len(tracks))
	for _, kind := range []string{"keyboard", "mouse-buttons", "absolute-motion", "relative-motion", "scroll"} {
		if track := tracks[kind]; track != nil {
			result = append(result, *track)
		}
	}
	return result
}

// Delete 删 clip 记录 (blob 由 GC 回收孤儿字节).
func (s *Service) Delete(id string) error {
	before := s.store.Revision()
	defer s.emitChangedSince(before)
	if err := s.store.DeleteRecord(id); err != nil {
		return err
	}
	return nil
}

// Update only changes presentation metadata; the content-addressed carrier remains stable.
func (s *Service) Update(id string, label, description, category string, tags []string) error {
	before := s.store.Revision()
	defer s.emitChangedSince(before)
	if err := s.store.PutRecordMeta(id, label, description, category, tags); err != nil {
		return err
	}
	return nil
}
