// Package configvalidator owns the trusted registry of pure, compile-time
// config validators. Node contracts pin validator identities and exact
// implementation digests; composition decides which implementations exist.
package configvalidator

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

type ValidateFunc func(any) error

type Descriptor struct {
	ID             string
	SemanticDigest artifact.Digest
	Validate       ValidateFunc
}

type registryState struct {
	byID map[string]Descriptor
}

type Registry struct{ state *registryState }

var versionedPathPattern = regexp.MustCompile(`/v[1-9][0-9]*$`)

func Seal(descriptors []Descriptor) (Registry, error) {
	ordered := append([]Descriptor(nil), descriptors...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	byID := make(map[string]Descriptor, len(ordered))
	previous := ""
	for _, descriptor := range ordered {
		parsed, err := url.ParseRequestURI(descriptor.ID)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || !versionedPathPattern.MatchString(parsed.Path) ||
			descriptor.ID <= previous || !descriptor.SemanticDigest.Valid() || descriptor.Validate == nil {
			return Registry{}, errors.New("invalid or duplicate config validator descriptor")
		}
		previous = descriptor.ID
		byID[descriptor.ID] = descriptor
	}
	return Registry{state: &registryState{byID: byID}}, nil
}

func (r Registry) Valid() bool { return r.state != nil }

func (r Registry) Validate(machine nodecontract.MachineContract, config map[string]any) error {
	if !r.Valid() {
		return errors.New("config validator registry is invalid")
	}
	for _, declaration := range machine.ConfigValidators {
		descriptor, ok := r.state.byID[declaration.ValidatorID]
		if !ok {
			return fmt.Errorf("config validator %q is not installed", declaration.ValidatorID)
		}
		if descriptor.SemanticDigest != declaration.SemanticDigest {
			return fmt.Errorf("config validator %q implementation digest does not match its contract", declaration.ValidatorID)
		}
		value, ok := config[declaration.ConfigKey]
		if !ok {
			return fmt.Errorf("config validator %q references missing config key %q", declaration.ID, declaration.ConfigKey)
		}
		if err := descriptor.Validate(value); err != nil {
			return fmt.Errorf("config validator %q rejected %q: %w", declaration.ID, declaration.ConfigKey, err)
		}
	}
	return nil
}
