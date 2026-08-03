package macro

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
)

type Service struct {
	store *asset.Store
	emit  func(name string, data any)
}

func NewService(store *asset.Store, emit ...func(name string, data any)) *Service {
	service := &Service{store: store}
	if len(emit) != 0 {
		service.emit = emit[0]
	}
	return service
}

func (s *Service) Save(value *Macro) (*Macro, error) {
	if s.store == nil || value == nil {
		return nil, errors.New("macro service requires a store and value")
	}
	label := strings.TrimSpace(value.Label)
	if label == "" || len([]rune(label)) > 80 {
		return nil, errors.New("macro label must contain 1 to 80 characters")
	}
	if err := Validate(value.Document); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(value.ID)
	if id == "" {
		id = "macro-" + uuid.NewString()
	}
	createdAt := time.Now().UTC()
	if existing, ok := s.store.Get(id); ok {
		if existing.Kind != asset.KindMacro {
			return nil, fmt.Errorf("asset %q is not a macro", id)
		}
		if !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
	} else if value.CreatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, value.CreatedAt); err == nil {
			createdAt = parsed.UTC()
		}
	}
	var carrier bytes.Buffer
	if err := Encode(&carrier, value.Document); err != nil {
		return nil, err
	}
	before := s.store.Revision()
	ref, err := s.store.CommitRecordBlob(context.Background(), MediaType, bytes.NewReader(carrier.Bytes()), func(ref blob.BlobRef) asset.AssetRecord {
		return asset.AssetRecord{
			GUID: id, Kind: asset.KindMacro, Name: label, Description: strings.TrimSpace(value.Description),
			Category: strings.TrimSpace(value.Category), Tags: normalizeTags(value.Tags), Origin: asset.Origin{Kind: "user"},
			Blob: &ref, CreatedAt: createdAt,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("commit macro: %w", err)
	}
	s.emitChangedSince(before)
	saved := cloneMacro(value)
	saved.ID, saved.Label, saved.CreatedAt, saved.Blob = id, label, createdAt.Format(time.RFC3339), ref
	saved.Description, saved.Category, saved.Tags = strings.TrimSpace(value.Description), strings.TrimSpace(value.Category), normalizeTags(value.Tags)
	return saved, nil
}

func (s *Service) Get(id string) (*Macro, error) {
	if s.store == nil {
		return nil, errors.New("macro service store is unavailable")
	}
	record, ok := s.store.Get(id)
	if !ok || record.Kind != asset.KindMacro || record.Blob == nil {
		return nil, fmt.Errorf("macro %q not found", id)
	}
	content, err := s.store.ReadBlob(context.Background(), *record.Blob)
	if err != nil {
		return nil, fmt.Errorf("read macro %q: %w", id, err)
	}
	document, err := Decode(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("decode macro %q: %w", id, err)
	}
	return &Macro{
		ID: record.GUID, Label: record.Name, Description: record.Description, Category: record.Category,
		Tags: append([]string(nil), record.Tags...), CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
		Document: document, Blob: *record.Blob,
	}, nil
}

func (s *Service) List() []Summary {
	if s.store == nil {
		return []Summary{}
	}
	result := []Summary{}
	for _, record := range s.store.List() {
		if record.Kind != asset.KindMacro {
			continue
		}
		value, err := s.Get(record.GUID)
		if err != nil {
			continue
		}
		analysis := Analyze(value.Document)
		result = append(result, Summary{
			ID: value.ID, Label: value.Label, Description: value.Description, Category: value.Category,
			Tags: append([]string(nil), value.Tags...), CreatedAt: value.CreatedAt,
			ActionCount: len(value.Document.Actions), DurationUs: analysis.DurationUs, Blob: value.Blob,
		})
	}
	return result
}

func (s *Service) Analyze(document Document) Analysis {
	return Analyze(document)
}

func (s *Service) Delete(id string) error {
	if s.store == nil {
		return errors.New("macro service store is unavailable")
	}
	record, ok := s.store.Get(id)
	if !ok || record.Kind != asset.KindMacro {
		return fmt.Errorf("macro %q not found", id)
	}
	before := s.store.Revision()
	if err := s.store.DeleteRecord(id); err != nil {
		return err
	}
	s.emitChangedSince(before)
	return nil
}

func (s *Service) emitChangedSince(before uint64) {
	if s.emit == nil || s.store.Revision() == before {
		return
	}
	payload := map[string]any{"revision": s.store.Revision()}
	s.emit("asset:changed", payload)
	s.emit("macro:changed", payload)
}

func cloneMacro(source *Macro) *Macro {
	clone := *source
	clone.Tags = append([]string(nil), source.Tags...)
	clone.Document = CloneDocument(source.Document)
	return &clone
}

func normalizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		key := strings.ToLower(tag)
		if key == "" {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	return result
}
