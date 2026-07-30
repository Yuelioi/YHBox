package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/durablefs"
)

const (
	SettingsFormat        = "yotta.settings"
	SettingsSchemaVersion = "1"
	settingsPayloadDomain = "yotta/settings-payload/v1"
	maxSettingsBytes      = 16 << 20
)

var ErrSettingsRecoveryRequired = errors.New("settings recovery is required")

type settingsEnvelope struct {
	Format     string          `json:"format"`
	Version    string          `json:"version"`
	Generation uint64          `json:"generation"`
	Checksum   artifact.Digest `json:"checksum"`
	Payload    json.RawMessage `json:"payload"`
}

type settingsCandidate struct {
	path            string
	raw             []byte
	settings        *Settings
	generation      uint64
	checksum        artifact.Digest
	priority        int
	requiresRewrite bool
}

// SettingsStore owns versioned settings generations and crash recovery.
// Callers only load one validated snapshot and save complete Settings values.
type SettingsStore struct {
	mu         sync.Mutex
	path       string
	generation uint64
}

func OpenSettingsStore(path string) (*SettingsStore, *Settings, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil, errors.New("settings path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, err
	}
	store := &SettingsStore{path: filepath.Clean(absolute)}
	candidates, discovered, err := store.readCandidates()
	if err != nil {
		return nil, nil, err
	}
	if len(candidates) == 0 {
		if discovered {
			return nil, nil, fmt.Errorf("%w: no valid settings generation at %q", ErrSettingsRecoveryRequired, store.path)
		}
		return store, defaultSettings(), nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].generation == candidates[j].generation {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].generation > candidates[j].generation
	})
	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.generation != selected.generation {
			break
		}
		if candidate.checksum != selected.checksum {
			return nil, nil, fmt.Errorf("%w: conflicting settings generation %d", ErrSettingsRecoveryRequired, selected.generation)
		}
	}
	store.generation = selected.generation
	if selected.requiresRewrite {
		if err := store.Save(selected.settings); err != nil {
			return nil, nil, fmt.Errorf("rewrite migrated settings: %w", err)
		}
	}
	return store, selected.settings.Clone(), nil
}

func (s *SettingsStore) Generation() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.generation
}

func (s *SettingsStore) Save(settings *Settings) error {
	if s == nil || settings == nil {
		return errors.New("settings store requires a complete settings snapshot")
	}
	if err := settings.Validate(); err != nil {
		return fmt.Errorf("validate settings: %w", err)
	}
	payload, err := artifact.Marshal(settings)
	if err != nil {
		return err
	}
	checksum, err := artifact.Sum(settingsPayloadDomain, payload)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.generation == ^uint64(0) {
		return errors.New("settings generation exhausted")
	}
	nextGeneration := s.generation + 1
	raw, err := json.MarshalIndent(settingsEnvelope{
		Format: SettingsFormat, Version: SettingsSchemaVersion,
		Generation: nextGeneration, Checksum: checksum, Payload: payload,
	}, "", "  ")
	if err != nil {
		return err
	}
	if len(raw) > maxSettingsBytes {
		return errors.New("settings envelope exceeds byte budget")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	staged, err := os.CreateTemp(filepath.Dir(s.path), "."+filepath.Base(s.path)+".staging-*")
	if err != nil {
		return fmt.Errorf("create settings staging file: %w", err)
	}
	stagedPath := staged.Name()
	committed := false
	defer func() {
		_ = staged.Close()
		if !committed {
			_ = os.Remove(stagedPath)
		}
	}()
	if err := staged.Chmod(0o600); err != nil {
		return err
	}
	if _, err := staged.Write(raw); err != nil {
		return err
	}
	if err := staged.Sync(); err != nil {
		return err
	}
	if err := staged.Close(); err != nil {
		return err
	}
	if err := s.preserveCurrent(); err != nil {
		return err
	}
	replaceErr := durablefs.Replace(stagedPath, s.path)
	if replaceErr == nil || durablefs.Committed(replaceErr) {
		committed = true
		s.generation = nextGeneration
		s.cleanupStaging()
	}
	if replaceErr != nil {
		if durablefs.Committed(replaceErr) {
			return &settingsCommittedError{err: fmt.Errorf("publish settings generation %d: %w", nextGeneration, replaceErr)}
		}
		return fmt.Errorf("publish settings generation %d: %w", nextGeneration, replaceErr)
	}
	return nil
}

func LoadSettings(path string) (*Settings, error) {
	_, settings, err := OpenSettingsStore(path)
	return settings, err
}

func SaveSettings(path string, settings *Settings) error {
	store, _, err := OpenSettingsStore(path)
	if err != nil {
		return err
	}
	return store.Save(settings)
}

func (s *SettingsStore) readCandidates() ([]settingsCandidate, bool, error) {
	paths := []struct {
		path     string
		priority int
	}{{s.path, 0}, {s.path + ".bak", 2}}
	staging, err := filepath.Glob(filepath.Join(filepath.Dir(s.path), "."+filepath.Base(s.path)+".staging-*"))
	if err != nil {
		return nil, false, err
	}
	sort.Strings(staging)
	for _, path := range staging {
		paths = append(paths, struct {
			path     string
			priority int
		}{path, 1})
	}
	discovered := false
	candidates := make([]settingsCandidate, 0, len(paths))
	for _, source := range paths {
		raw, err := os.ReadFile(source.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		discovered = true
		if err != nil {
			return nil, true, fmt.Errorf("read settings candidate %q: %w", source.path, err)
		}
		settings, envelope, requiresRewrite, err := decodeSettingsEnvelope(raw)
		if err != nil {
			continue
		}
		candidates = append(candidates, settingsCandidate{
			path: source.path, raw: raw, settings: settings,
			generation: envelope.Generation, checksum: envelope.Checksum, priority: source.priority,
			requiresRewrite: requiresRewrite,
		})
	}
	return candidates, discovered, nil
}

func decodeSettingsEnvelope(raw []byte) (*Settings, settingsEnvelope, bool, error) {
	if len(raw) == 0 || len(raw) > maxSettingsBytes {
		return nil, settingsEnvelope{}, false, errors.New("settings envelope exceeds byte budget")
	}
	var envelope settingsEnvelope
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, settingsEnvelope{}, false, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, settingsEnvelope{}, false, errors.New("settings envelope must contain exactly one JSON value")
	}
	if envelope.Format != SettingsFormat || envelope.Version != SettingsSchemaVersion || envelope.Generation == 0 || !envelope.Checksum.Valid() {
		return nil, settingsEnvelope{}, false, errors.New("unsupported settings envelope")
	}
	canonical, err := artifact.Canonicalize(envelope.Payload)
	if err != nil {
		return nil, settingsEnvelope{}, false, err
	}
	checksum, err := artifact.Sum(settingsPayloadDomain, canonical)
	if err != nil || checksum != envelope.Checksum {
		return nil, settingsEnvelope{}, false, errors.New("settings payload checksum mismatch")
	}
	canonical, requiresRewrite, err := migrateLegacySettingsPayload(canonical)
	if err != nil {
		return nil, settingsEnvelope{}, false, err
	}
	settings := defaultSettings()
	payloadDecoder := json.NewDecoder(bytes.NewReader(canonical))
	payloadDecoder.DisallowUnknownFields()
	if err := payloadDecoder.Decode(settings); err != nil {
		return nil, settingsEnvelope{}, false, err
	}
	if err := payloadDecoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, settingsEnvelope{}, false, errors.New("settings payload must contain exactly one JSON value")
	}
	if err := settings.Validate(); err != nil {
		return nil, settingsEnvelope{}, false, err
	}
	return settings, envelope, requiresRewrite, nil
}

// migrateLegacySettingsPayload is the one-way compatibility reader for
// settings written before configured targets became plain configuration.
// Unknown fields remain strict; only explicitly retired fields are removed
// before decoding the current model.
func migrateLegacySettingsPayload(raw []byte) ([]byte, bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil {
		return nil, false, err
	}
	removed := false
	for _, migration := range []struct {
		section string
		entries string
		fields  []string
	}{
		{section: "ai", entries: "profiles", fields: []string{"workflowConsent"}},
		{section: "network", entries: "httpOrigins", fields: []string{"workflowConsent", "allowPrivateNetwork"}},
		{section: "applications", entries: "profiles", fields: []string{"workflowConsent", "executableDigest"}},
		{section: "automation", entries: "targets", fields: []string{"workflowConsent"}},
	} {
		section, ok := payload[migration.section].(map[string]any)
		if !ok {
			continue
		}
		entries, ok := section[migration.entries].([]any)
		if !ok {
			continue
		}
		for _, entry := range entries {
			if object, ok := entry.(map[string]any); ok {
				for _, field := range migration.fields {
					if _, exists := object[field]; exists {
						delete(object, field)
						removed = true
					}
				}
			}
		}
	}
	migrated, err := artifact.Marshal(payload)
	return migrated, removed, err
}

func (s *SettingsStore) preserveCurrent() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read current settings: %w", err)
	}
	if _, _, _, decodeErr := decodeSettingsEnvelope(raw); decodeErr == nil {
		if err := durablefs.WriteFile(s.path+".bak", raw, 0o600); err != nil {
			return fmt.Errorf("backup current settings: %w", err)
		}
		return nil
	}
	digest, err := artifact.Sum("yotta/settings-recovery/v1", raw)
	if err != nil {
		return err
	}
	recoveryDir := filepath.Join(filepath.Dir(s.path), "recovery")
	if err := os.MkdirAll(recoveryDir, 0o700); err != nil {
		return fmt.Errorf("create settings recovery directory: %w", err)
	}
	name := "settings-" + strings.TrimPrefix(digest.String(), "sha256:") + ".json"
	recoveryPath := filepath.Join(recoveryDir, name)
	if existing, readErr := os.ReadFile(recoveryPath); readErr == nil {
		if !bytes.Equal(existing, raw) {
			return errors.New("settings recovery digest collision")
		}
		return nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	if err := durablefs.WriteFile(recoveryPath, raw, 0o600); err != nil {
		return fmt.Errorf("preserve invalid settings: %w", err)
	}
	return nil
}

func (s *SettingsStore) cleanupStaging() {
	paths, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), "."+filepath.Base(s.path)+".staging-*"))
	for _, path := range paths {
		if path == s.path {
			continue
		}
		_ = os.Remove(path)
	}
}
