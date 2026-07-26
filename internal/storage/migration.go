package storage

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/yottaapp/yotta/internal/artifact"
)

var (
	ErrNoMigrationPath = errors.New("storage layout has no registered migration path")
	ErrFutureLayout    = errors.New("storage layout is newer than this application")
)

var migrationIDPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,127}$`)

// MigrationStep is the immutable release identity of one root-layout upgrade.
// Once a step ships, its ID, endpoints, and checksum must never be changed.
// The checksum identifies the reviewed implementation and fixtures rather than
// user data.
type MigrationStep struct {
	ID       string
	From     string
	To       string
	Checksum artifact.Digest
}

// MigrationRegistry owns deterministic upgrade routing. It deliberately does
// not execute steps: preflight, backup, journaled execution, and verification
// are a separate lifecycle boundary and must never happen as a Store side
// effect.
type MigrationRegistry struct {
	current string
	byFrom  map[string]MigrationStep
}

func NewMigrationRegistry(current string, steps []MigrationStep) (MigrationRegistry, error) {
	currentNumber, err := parseLayoutVersion(current)
	if err != nil {
		return MigrationRegistry{}, fmt.Errorf("current layout version: %w", err)
	}
	registry := MigrationRegistry{current: current, byFrom: make(map[string]MigrationStep, len(steps))}
	ids := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		if !migrationIDPattern.MatchString(step.ID) {
			return MigrationRegistry{}, fmt.Errorf("invalid migration step ID %q", step.ID)
		}
		if _, exists := ids[step.ID]; exists {
			return MigrationRegistry{}, fmt.Errorf("duplicate migration step ID %q", step.ID)
		}
		ids[step.ID] = struct{}{}
		from, err := parseLayoutVersion(step.From)
		if err != nil {
			return MigrationRegistry{}, fmt.Errorf("migration %s from version: %w", step.ID, err)
		}
		to, err := parseLayoutVersion(step.To)
		if err != nil {
			return MigrationRegistry{}, fmt.Errorf("migration %s to version: %w", step.ID, err)
		}
		if from >= to {
			return MigrationRegistry{}, fmt.Errorf("migration %s must advance its layout version", step.ID)
		}
		if to > currentNumber {
			return MigrationRegistry{}, fmt.Errorf("migration %s targets layout %s beyond current %s", step.ID, step.To, current)
		}
		if !step.Checksum.Valid() {
			return MigrationRegistry{}, fmt.Errorf("migration %s has an invalid implementation checksum", step.ID)
		}
		if previous, exists := registry.byFrom[step.From]; exists {
			return MigrationRegistry{}, fmt.Errorf("ambiguous migrations from layout %s: %s and %s", step.From, previous.ID, step.ID)
		}
		registry.byFrom[step.From] = step
	}
	for _, step := range steps {
		if _, err := registry.Plan(step.From); err != nil {
			return MigrationRegistry{}, fmt.Errorf("migration %s is not connected to current layout: %w", step.ID, err)
		}
	}
	return registry, nil
}

func (r MigrationRegistry) Current() string { return r.current }

// Plan returns the only registered chain from a released layout to the current
// layout. A newer root or a gap is rejected instead of being guessed.
func (r MigrationRegistry) Plan(from string) ([]MigrationStep, error) {
	fromNumber, err := parseLayoutVersion(from)
	if err != nil {
		return nil, err
	}
	currentNumber, err := parseLayoutVersion(r.current)
	if err != nil {
		return nil, errors.New("migration registry is not initialized")
	}
	if fromNumber > currentNumber {
		return nil, fmt.Errorf("%w: found %s, current %s", ErrFutureLayout, from, r.current)
	}
	if from == r.current {
		return []MigrationStep{}, nil
	}
	plan := make([]MigrationStep, 0, currentNumber-fromNumber)
	seen := make(map[string]struct{}, currentNumber-fromNumber)
	next := from
	for next != r.current {
		if _, exists := seen[next]; exists {
			return nil, fmt.Errorf("%w: cycle at layout %s", ErrNoMigrationPath, next)
		}
		seen[next] = struct{}{}
		step, exists := r.byFrom[next]
		if !exists {
			return nil, fmt.Errorf("%w: %s -> %s", ErrNoMigrationPath, from, r.current)
		}
		plan = append(plan, step)
		next = step.To
	}
	return plan, nil
}

func parseLayoutVersion(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 || strconv.Itoa(number) != value {
		return 0, fmt.Errorf("layout version must be a positive canonical integer, got %q", value)
	}
	return number, nil
}
