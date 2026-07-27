package compiler

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

// StateBinding is invocation-scoped attenuated authority. A node adapter can
// only access the slot and operation declared by its immutable Node Contract.
type stateBinding struct {
	mode nodecontract.StateAccessMode
	slot *runStateSlot
}

func (b stateBinding) Read() (nodeadapter.StateSnapshot, error) {
	if b.slot == nil || b.mode != nodecontract.StateRead {
		return nodeadapter.StateSnapshot{}, errors.New("state binding does not grant read access")
	}
	return b.slot.read(), nil
}

func (b stateBinding) Write(value datatype.ValueEnvelope) (nodeadapter.StateSnapshot, error) {
	if b.slot == nil || b.mode != nodecontract.StateWrite {
		return nodeadapter.StateSnapshot{}, errors.New("state binding does not grant write access")
	}
	return b.slot.write(value)
}

// Update performs one read-modify-write transaction while the slot is
// exclusively locked. It deliberately shares the existing write authority:
// contracts do not gain a second, subtly different state permission, while
// built-in convenience nodes avoid a racy Read followed by Write sequence.
func (b stateBinding) Update(transform func(datatype.ValueEnvelope) (datatype.ValueEnvelope, error)) (nodeadapter.StateSnapshot, error) {
	if b.slot == nil || b.mode != nodecontract.StateWrite {
		return nodeadapter.StateSnapshot{}, errors.New("state binding does not grant write access")
	}
	if transform == nil {
		return nodeadapter.StateSnapshot{}, errors.New("state update transform is missing")
	}
	return b.slot.update(transform)
}

type runState struct {
	slots map[string]*runStateSlot
}

type runStateSlot struct {
	mu        sync.RWMutex
	typeRef   datatype.ResolvedType
	value     datatype.ValueEnvelope
	revision  int64
	changedAt time.Time
	now       func() time.Time
}

func newRunState(slots []programStateSlot, catalog nodecatalog.Snapshot, now func() time.Time) (*runState, error) {
	if !catalog.Valid() || now == nil {
		return nil, errors.New("run state requires a trusted Catalog and clock")
	}
	result := &runState{slots: make(map[string]*runStateSlot, len(slots))}
	initializedAt := now().UTC()
	for _, slot := range slots {
		envelope, err := datatype.OpenValueEnvelope(catalog, slot.Initial)
		if err != nil {
			return nil, fmt.Errorf("open initial state %q: %w", slot.Name, err)
		}
		if !reflect.DeepEqual(envelope.Type(), slot.Type) || envelope.Representation() != datatype.RepresentationInlineJSON {
			return nil, fmt.Errorf("initial state %q violates its frozen type", slot.Name)
		}
		result.slots[slot.Name] = &runStateSlot{
			typeRef: slot.Type, value: envelope, revision: 0, changedAt: initializedAt, now: now,
		}
	}
	return result, nil
}

func (s *runState) bindings(machine nodecontract.MachineContract, config map[string]any) (map[string]nodeadapter.StateBinding, error) {
	result := make(map[string]nodeadapter.StateBinding, len(machine.StateAccesses))
	for _, access := range machine.StateAccesses {
		name, ok := config[access.SlotConfigKey].(string)
		slot := s.slots[name]
		if !ok || slot == nil {
			return nil, fmt.Errorf("state access %q has no bound slot", access.ID)
		}
		result[access.ID] = stateBinding{mode: access.Mode, slot: slot}
	}
	return result, nil
}

func (s *runStateSlot) read() nodeadapter.StateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return nodeadapter.StateSnapshot{Value: s.value, Revision: s.revision, ChangedAt: s.changedAt}
}

func (s *runStateSlot) write(value datatype.ValueEnvelope) (nodeadapter.StateSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeLocked(value)
}

func (s *runStateSlot) update(transform func(datatype.ValueEnvelope) (datatype.ValueEnvelope, error)) (nodeadapter.StateSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := transform(s.value)
	if err != nil {
		return nodeadapter.StateSnapshot{}, err
	}
	return s.writeLocked(value)
}

func (s *runStateSlot) writeLocked(value datatype.ValueEnvelope) (nodeadapter.StateSnapshot, error) {
	if !value.Valid() || !value.Durable() || value.Representation() != datatype.RepresentationInlineJSON || !reflect.DeepEqual(value.Type(), s.typeRef) {
		return nodeadapter.StateSnapshot{}, errors.New("state write value violates the frozen slot type")
	}
	if s.revision >= schema.MaxRevision {
		return nodeadapter.StateSnapshot{}, errors.New("state revision budget exceeded")
	}
	s.value = value
	s.revision++
	s.changedAt = s.now().UTC()
	return nodeadapter.StateSnapshot{Value: s.value, Revision: s.revision, ChangedAt: s.changedAt}, nil
}
