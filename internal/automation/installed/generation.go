package installed

import (
	"context"
	"errors"
	"sync"
)

// Generation owns one immutable installation snapshot. Authoring calls and
// Runs acquire leases from the same owner, so retiring a target can publish
// immediately while native providers close only after old work releases them.
type Generation struct{ state *generationState }

type generationState struct {
	mu            sync.Mutex
	installations Installations
	bySlot        map[string]*provider
	leases        int
	retired       bool
	closed        bool
	finalized     bool
	closeErr      error
	done          chan struct{}
}

type GenerationLease struct {
	state *generationState
	once  sync.Once
}

func NewGeneration(installations Installations) (Generation, error) {
	bySlot, err := providersBySlot(installations)
	if err != nil {
		return Generation{}, err
	}
	return Generation{state: &generationState{
		installations: installations,
		bySlot:        bySlot,
		done:          make(chan struct{}),
	}}, nil
}

func (generation Generation) Valid() bool {
	return generation.state != nil && generation.state.installations.Valid()
}

func (generation Generation) Acquire() (GenerationLease, error) {
	if !generation.Valid() {
		return GenerationLease{}, errors.New("automation target generation is unavailable")
	}
	generation.state.mu.Lock()
	defer generation.state.mu.Unlock()
	if generation.state.retired || generation.state.closed {
		return GenerationLease{}, errors.New("automation target generation is retired")
	}
	generation.state.leases++
	return GenerationLease{state: generation.state}, nil
}

func (lease *GenerationLease) Release() {
	if lease == nil || lease.state == nil {
		return
	}
	lease.once.Do(func() {
		state := lease.state
		state.mu.Lock()
		if state.leases > 0 {
			state.leases--
		}
		closeNow := state.retired && state.leases == 0 && !state.closed
		if closeNow {
			state.closed = true
		}
		state.mu.Unlock()
		if closeNow {
			closeErr := state.installations.Close()
			state.mu.Lock()
			state.closeErr = closeErr
			state.finalized = true
			close(state.done)
			state.mu.Unlock()
		}
	})
}

func (lease *GenerationLease) provider(slot string) (*provider, error) {
	if lease.state == nil {
		return nil, errors.New("automation target generation lease is unavailable")
	}
	provider := lease.state.bySlot[slot]
	if provider == nil {
		return nil, errors.New("automation target slot is not installed in this generation")
	}
	return provider, nil
}

// Retire prevents new leases and closes the native installation snapshot as
// soon as all already-issued authoring and Run leases have been released.
func (generation Generation) Retire() error {
	if !generation.Valid() {
		return nil
	}
	state := generation.state
	state.mu.Lock()
	state.retired = true
	closeNow := state.leases == 0 && !state.closed
	if closeNow {
		state.closed = true
	}
	state.mu.Unlock()
	if closeNow {
		closeErr := state.installations.Close()
		state.mu.Lock()
		state.closeErr = closeErr
		state.finalized = true
		close(state.done)
		state.mu.Unlock()
		return closeErr
	}
	return nil
}

func (generation Generation) WaitClosed(ctx context.Context) error {
	if ctx == nil {
		return errors.New("wait for automation generation context is required")
	}
	if !generation.Valid() {
		return nil
	}
	select {
	case <-generation.state.done:
		generation.state.mu.Lock()
		err := generation.state.closeErr
		generation.state.mu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (generation Generation) Closed() (bool, error) {
	if !generation.Valid() {
		return true, nil
	}
	generation.state.mu.Lock()
	defer generation.state.mu.Unlock()
	return generation.state.finalized, generation.state.closeErr
}
