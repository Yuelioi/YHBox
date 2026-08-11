package services

import "testing"

func TestAppInfoUsesCanonicalRepository(t *testing.T) {
	const want = "https://github.com/yuelioi/yotta"
	if got := NewAppInfoService().Info().Repo; got != want {
		t.Fatalf("repository URL = %q, want %q", got, want)
	}
}
