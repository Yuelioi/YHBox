package installed

import (
	"context"
	"testing"
)

func TestGenerationRetiresAfterLastLease(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	generation := authoringTestGeneration(t, "editor", profile, provider)

	lease, err := generation.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if err := generation.Retire(); err != nil {
		t.Fatal(err)
	}
	if driver.closed != 0 {
		t.Fatal("retirement closed a provider with an active generation lease")
	}
	if closed, _ := generation.Closed(); closed {
		t.Fatal("generation reported closed before its last lease released")
	}
	if _, err := generation.Acquire(); err == nil {
		t.Fatal("retired generation issued a new lease")
	}
	lease.Release()
	if err := generation.WaitClosed(context.Background()); err != nil {
		t.Fatal(err)
	}
	if driver.closed != 1 {
		t.Fatalf("provider close count = %d, want 1", driver.closed)
	}
	if closed, err := generation.Closed(); !closed || err != nil {
		t.Fatalf("generation close state = %v, %v", closed, err)
	}
}

func TestGenerationImmediatelyReclaimsIdleProviders(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	generation := authoringTestGeneration(t, "editor", profile, &provider{profile: profile, driver: driver})

	if err := generation.Retire(); err != nil {
		t.Fatal(err)
	}
	if err := generation.WaitClosed(context.Background()); err != nil {
		t.Fatal(err)
	}
	if driver.closed != 1 {
		t.Fatalf("provider close count = %d, want 1", driver.closed)
	}
}
