package configvalidator

import (
	"errors"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestRegistryRequiresExactInstalledValidatorImplementation(t *testing.T) {
	digest := artifact.Digest("sha256:" + strings.Repeat("1", 64))
	registry, err := Seal([]Descriptor{{
		ID: "https://schemas.yotta.dev/config-validators/test/v1", SemanticDigest: digest,
		Validate: func(value any) error {
			if value != "valid" {
				return errors.New("rejected")
			}
			return nil
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	machine := nodecontract.MachineContract{ConfigValidators: []nodecontract.ConfigValidatorSpec{{
		ID: "test", ConfigKey: "value", ValidatorID: "https://schemas.yotta.dev/config-validators/test/v1", SemanticDigest: digest,
	}}}
	if err := registry.Validate(machine, map[string]any{"value": "valid"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Validate(machine, map[string]any{"value": "invalid"}); err == nil {
		t.Fatal("validator rejection was ignored")
	}
	machine.ConfigValidators[0].SemanticDigest = artifact.Digest("sha256:" + strings.Repeat("2", 64))
	if err := registry.Validate(machine, map[string]any{"value": "valid"}); err == nil {
		t.Fatal("validator implementation digest mismatch was ignored")
	}
	if err := (Registry{}).Validate(nodecontract.MachineContract{}, nil); err == nil {
		t.Fatal("invalid registry was treated as an empty trusted registry")
	}
}
