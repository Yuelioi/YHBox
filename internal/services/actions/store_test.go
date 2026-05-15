package actions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_LoadMissingDir(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "actions"))
	if got := s.List(); len(got) != 0 {
		t.Errorf("空目录应返空列表，got %d", len(got))
	}
}

func TestStore_CreateLoadRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "actions")
	s := NewStore(dir)
	a := Action{
		ID:    "test-id-1",
		Name:  "测试",
		Steps: []Step{{Kind: StepSleep, DurationMs: 100}},
	}
	NormalizeAction(&a)
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(&a); err != nil {
		t.Fatal(err)
	}

	// 文件应在 dir/test-id-1/action.json
	if _, err := os.Stat(filepath.Join(dir, "test-id-1", "action.json")); err != nil {
		t.Errorf("Create 后文件应存在: %v", err)
	}

	s2 := NewStore(dir)
	got := s2.List()
	if len(got) != 1 || got[0].Name != "测试" {
		t.Errorf("往返丢数据 %+v", got)
	}
}

func TestStore_CorruptFolderSkipped(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "actions")
	if err := os.MkdirAll(filepath.Join(dir, "bad"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad", "action.json"), []byte("{broken json"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewStore(dir)
	if len(s.List()) != 0 {
		t.Error("损坏 action.json 应跳过当作空")
	}
}

func TestStore_DeleteRemovesFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "actions")
	s := NewStore(dir)
	a := Action{ID: "d1", Name: "del"}
	NormalizeAction(&a)
	_ = s.Create(&a)
	actionDir := filepath.Join(dir, "d1")
	if _, err := os.Stat(actionDir); err != nil {
		t.Fatalf("文件夹预期存在: %v", err)
	}
	if err := s.Delete("d1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(actionDir); !os.IsNotExist(err) {
		t.Errorf("Delete 后文件夹应消失，got err=%v", err)
	}
}

func TestStore_UpdateUsesMutator(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "actions")
	s := NewStore(dir)
	a := Action{ID: "u1", Name: "old"}
	NormalizeAction(&a)
	_ = s.Create(&a)
	if err := s.Update("u1", func(x *Action) error {
		x.Name = "new"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("u1")
	if got.Name != "new" {
		t.Errorf("name 未生效: %q", got.Name)
	}
}

func TestStore_PathHelpers(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "actions")
	s := NewStore(dir)
	if got, want := s.ActionDir("x"), filepath.Join(dir, "x"); got != want {
		t.Errorf("ActionDir: got %q want %q", got, want)
	}
	if got, want := s.TemplatesDir("x"), filepath.Join(dir, "x", "templates"); got != want {
		t.Errorf("TemplatesDir: got %q want %q", got, want)
	}
	if got, want := s.ReadmePath("x"), filepath.Join(dir, "x", "README.md"); got != want {
		t.Errorf("ReadmePath: got %q want %q", got, want)
	}
}

func TestWipeV1Data_DetectAndWipe(t *testing.T) {
	dir := t.TempDir()
	v1JSON := `{
  "schemaVersion": 2,
  "id": "old-1",
  "name": "v1 action",
  "hotkey": "Ctrl+Shift+A",
  "loopMode": "infinite",
  "category": "战斗"
}`
	subdir := filepath.Join(dir, "old-1")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "action.json"), []byte(v1JSON), 0o644); err != nil {
		t.Fatal(err)
	}
	tplDir := filepath.Join(subdir, "templates")
	_ = os.MkdirAll(tplDir, 0o755)
	_ = os.WriteFile(filepath.Join(tplDir, "x.png"), []byte("png"), 0o644)

	wiped, err := WipeV1Data(dir)
	if err != nil {
		t.Fatalf("WipeV1Data: %v", err)
	}
	if !wiped {
		t.Error("expected wipe=true")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected empty dir, got %d entries", len(entries))
	}
}

func TestWipeV1Data_CleanV2NoWipe(t *testing.T) {
	dir := t.TempDir()
	v2JSON := `{
  "schemaVersion": 2,
  "id": "new-1",
  "name": "v2 action",
  "steps": []
}`
	subdir := filepath.Join(dir, "new-1")
	_ = os.MkdirAll(subdir, 0o755)
	_ = os.WriteFile(filepath.Join(subdir, "action.json"), []byte(v2JSON), 0o644)

	wiped, err := WipeV1Data(dir)
	if err != nil {
		t.Fatalf("WipeV1Data: %v", err)
	}
	if wiped {
		t.Error("expected wipe=false (clean v2)")
	}
	if _, err := os.Stat(filepath.Join(subdir, "action.json")); err != nil {
		t.Error("v2 file should still exist")
	}
}

func TestWipeV1Data_MissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	wiped, err := WipeV1Data(root)
	if err != nil {
		t.Fatalf("expected nil err on missing root, got %v", err)
	}
	if wiped {
		t.Error("expected wipe=false when root missing")
	}
}
