package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
)

func TestSettingsStoreCreatesVersionedChecksummedGenerations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, settings, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("OpenSettingsStore: %v", err)
	}
	if store.Generation() != 0 || settings.Locale != "zh" {
		t.Fatalf("new store = generation %d, settings %#v", store.Generation(), settings)
	}
	settings.Locale = "en"
	if err := store.Save(settings); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope settingsEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Format != SettingsFormat || envelope.Version != SettingsSchemaVersion ||
		envelope.Generation != 1 || !envelope.Checksum.Valid() {
		t.Fatalf("envelope = %#v", envelope)
	}
	reopened, loaded, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Generation() != 1 || loaded.Locale != "en" {
		t.Fatalf("reopened = generation %d, locale %q", reopened.Generation(), loaded.Locale)
	}
}

func TestSettingsStoreRejectsCorruptOnlyGeneration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := OpenSettingsStore(path)
	if !errors.Is(err, ErrSettingsRecoveryRequired) {
		t.Fatalf("OpenSettingsStore error = %v, want recovery required", err)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil || string(raw) != "{broken" {
		t.Fatalf("corrupt settings were changed: %q, %v", raw, readErr)
	}
}

func TestSettingsStoreRecoversNewestValidBackupOrStaging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store, settings, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatal(err)
	}
	settings.Locale = "en"
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	settings.UI.Window.Width = 1400
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".bak", first, 0o600); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, ".settings.json.staging-recovery")
	if err := os.WriteFile(staged, second, 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, loaded, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if recovered.Generation() != 2 || loaded.UI.Window.Width != 1400 {
		t.Fatalf("recovered generation=%d width=%d", recovered.Generation(), loaded.UI.Window.Width)
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != "{broken" {
		t.Fatalf("recovery changed corrupt primary before commit: %q, %v", raw, err)
	}
	if err := recovered.Save(loaded); err != nil {
		t.Fatalf("save recovered: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "recovery", "settings-*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("recovery files = %v, %v", matches, err)
	}
}

func TestSettingsStoreRejectsChecksumMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, settings, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	payload := envelope["payload"].(map[string]any)
	payload["locale"] = "en"
	tampered, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = OpenSettingsStore(path)
	if !errors.Is(err, ErrSettingsRecoveryRequired) {
		t.Fatalf("tampered settings error = %v", err)
	}
}

func TestSettingsStoreMigratesRetiredTargetSecurityFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	settings := defaultSettings()
	settings.Applications.Profiles = []InstalledApplicationSettings{{
		Slot: "demo", Label: "Demo", Executable: filepath.Join(t.TempDir(), "demo.exe"), Arguments: []string{"--run"},
	}}
	settings.Network.HTTPOrigins = []HTTPOriginSettings{{
		Slot: "api", Label: "API", Origin: "http://127.0.0.1:8080",
		ResponseByteLimit: 4096, TimeoutMilliseconds: 5000,
	}}
	payload, err := artifact.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var legacy map[string]any
	if err := json.Unmarshal(payload, &legacy); err != nil {
		t.Fatal(err)
	}
	applications := legacy["applications"].(map[string]any)["profiles"].([]any)
	applications[0].(map[string]any)["executableDigest"] = "sha256:" + strings.Repeat("1", 64)
	network := legacy["network"].(map[string]any)["httpOrigins"].([]any)
	network[0].(map[string]any)["allowPrivateNetwork"] = true
	legacyPayload, err := artifact.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	checksum, err := artifact.Sum(settingsPayloadDomain, legacyPayload)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(settingsEnvelope{
		Format: SettingsFormat, Version: SettingsSchemaVersion,
		Generation: 7, Checksum: checksum, Payload: legacyPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	store, loaded, err := OpenSettingsStore(path)
	if err != nil {
		t.Fatalf("OpenSettingsStore legacy target configuration: %v", err)
	}
	if store.Generation() != 8 || len(loaded.Applications.Profiles) != 1 || len(loaded.Network.HTTPOrigins) != 1 {
		t.Fatalf("migrated settings = generation %d, applications=%d, network=%d",
			store.Generation(), len(loaded.Applications.Profiles), len(loaded.Network.HTTPOrigins))
	}
	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(rewritten, []byte("executableDigest")) || bytes.Contains(rewritten, []byte("allowPrivateNetwork")) {
		t.Fatalf("rewritten settings retained retired target fields: %s", rewritten)
	}
}

func writeUncheckedSettingsEnvelope(t *testing.T, path string, settings *Settings, generation uint64) {
	t.Helper()
	payload, err := artifact.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	checksum, err := artifact.Sum(settingsPayloadDomain, payload)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(settingsEnvelope{
		Format: SettingsFormat, Version: SettingsSchemaVersion,
		Generation: generation, Checksum: checksum, Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}
