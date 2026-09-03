package macro

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/apperr"
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
		return nil, problem("macro.unavailable", apperr.CategoryInfrastructure, nil, true, errors.New("macro service requires a store and value"))
	}
	label := strings.TrimSpace(value.Label)
	if label == "" || len([]rune(label)) > 80 {
		return nil, problem("macro.invalid", apperr.CategoryValidation, map[string]any{"field": "label"}, false, errors.New("macro label must contain 1 to 80 characters"))
	}
	if err := Validate(value.Document); err != nil {
		return nil, problem("macro.invalid", apperr.CategoryValidation, nil, false, err)
	}
	id := strings.TrimSpace(value.ID)
	if id == "" {
		id = "macro-" + uuid.NewString()
	}
	createdAt := time.Now().UTC()
	existing, ok, err := s.store.Record(id)
	if err != nil {
		return nil, problem("macro.load_failed", apperr.CategoryInfrastructure, map[string]any{"id": id}, true, err)
	}
	if ok {
		if existing.Kind != asset.KindMacro {
			return nil, problem("macro.identity_conflict", apperr.CategoryDomain, map[string]any{"id": id}, false, fmt.Errorf("asset is not a macro"))
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
		return nil, problem("macro.invalid", apperr.CategoryValidation, map[string]any{"id": id}, false, err)
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
		return nil, problem("macro.save_failed", apperr.CategoryInfrastructure, map[string]any{"id": id}, true, err)
	}
	s.emitChangedSince(before)
	saved := cloneMacro(value)
	saved.ID, saved.Label, saved.CreatedAt, saved.Blob = id, label, createdAt.Format(time.RFC3339), ref
	saved.Description, saved.Category, saved.Tags = strings.TrimSpace(value.Description), strings.TrimSpace(value.Category), normalizeTags(value.Tags)
	return saved, nil
}

func (s *Service) Get(id string) (*Macro, error) {
	if s.store == nil {
		return nil, problem("macro.unavailable", apperr.CategoryInfrastructure, nil, true, errors.New("macro service store is unavailable"))
	}
	record, ok, err := s.store.Record(id)
	if err != nil {
		return nil, problem("macro.load_failed", apperr.CategoryInfrastructure, map[string]any{"id": id}, true, err)
	}
	if !ok || record.Kind != asset.KindMacro || record.Blob == nil {
		return nil, problem("macro.not_found", apperr.CategoryDomain, map[string]any{"id": id}, false, fmt.Errorf("macro not found"))
	}
	content, err := s.store.ReadBlob(context.Background(), *record.Blob)
	if err != nil {
		return nil, problem("macro.load_failed", apperr.CategoryInfrastructure, map[string]any{"id": id}, true, err)
	}
	decoded, err := decode(bytes.NewReader(content))
	if err != nil {
		return nil, problem("macro.corrupt", apperr.CategoryDomain, map[string]any{"id": id}, false, err)
	}
	document := decoded.Document
	ref := *record.Blob
	if decoded.SourceVersion != SchemaVersion {
		var carrier bytes.Buffer
		if err := Encode(&carrier, document); err != nil {
			return nil, fmt.Errorf("encode migrated macro %q: %w", id, err)
		}
		before := s.store.Revision()
		ref, err = s.store.ReplaceRecordBlob(context.Background(), MediaType, bytes.NewReader(carrier.Bytes()), record.GUID, *record.Blob)
		if err != nil {
			return nil, fmt.Errorf("publish migrated macro %q: %w", id, err)
		}
		s.emitChangedSince(before)
	}
	return &Macro{
		ID: record.GUID, Label: record.Name, Description: record.Description, Category: record.Category,
		Tags: append([]string(nil), record.Tags...), CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
		Document: document, Blob: ref,
	}, nil
}

func (s *Service) List() ([]Summary, error) {
	if s.store == nil {
		return []Summary{}, nil
	}
	result := []Summary{}
	records, err := s.store.Records()
	if err != nil {
		return nil, problem("macro.list_failed", apperr.CategoryInfrastructure, nil, true, err)
	}
	for _, record := range records {
		if record.Kind != asset.KindMacro {
			continue
		}
		value, err := s.Get(record.GUID)
		if err != nil {
			return nil, fmt.Errorf("list macro %q: %w", record.GUID, err)
		}
		analysis := Analyze(value.Document)
		result = append(result, Summary{
			ID: value.ID, Label: value.Label, Description: value.Description, Category: value.Category,
			Tags: append([]string(nil), value.Tags...), CreatedAt: value.CreatedAt,
			ActionCount: len(value.Document.Actions), DurationUs: analysis.DurationUs, Blob: value.Blob,
		})
	}
	return result, nil
}

func (s *Service) Analyze(document Document) Analysis {
	return Analyze(document)
}

func (s *Service) Delete(id string) error {
	if s.store == nil {
		return problem("macro.unavailable", apperr.CategoryInfrastructure, nil, true, errors.New("macro service store is unavailable"))
	}
	record, ok, err := s.store.Record(id)
	if err != nil {
		return problem("macro.load_failed", apperr.CategoryInfrastructure, map[string]any{"id": id}, true, err)
	}
	if !ok || record.Kind != asset.KindMacro {
		return problem("macro.not_found", apperr.CategoryDomain, map[string]any{"id": id}, false, fmt.Errorf("macro not found"))
	}
	before := s.store.Revision()
	if err := s.store.DeleteRecord(id); err != nil {
		return problem("macro.delete_failed", apperr.CategoryInfrastructure, map[string]any{"id": id}, true, err)
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
