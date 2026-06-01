package container

import "testing"

func TestService_ResolveWindow_NotFound(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	svc := NewService(s)

	_, err := svc.ResolveWindow("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent containerID, got nil")
	}
}
