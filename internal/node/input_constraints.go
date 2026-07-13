package node

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
)

const MaxInputConstraints = 16

const CodeInputConstraintViolation = "INPUT_CONSTRAINT_VIOLATION"

// NormalizeInputConstraints validates constraints and returns their stable contract form.
func NormalizeInputConstraints(input InputSpec) ([]InputConstraint, error) {
	if len(input.Constraints) > MaxInputConstraints {
		return nil, fmt.Errorf("input %q exceeds constraint budget", input.Name)
	}
	normalized := append([]InputConstraint(nil), input.Constraints...)
	seen := make(map[InputConstraintKind]bool, len(input.Constraints))
	for index := range normalized {
		constraint := &normalized[index]
		if seen[constraint.Kind] {
			return nil, fmt.Errorf("input %q contains duplicate constraint %q", input.Name, constraint.Kind)
		}
		seen[constraint.Kind] = true
		switch constraint.Kind {
		case InputConstraintNonBlank:
			if input.Type != "String" || constraint.Threshold != "" {
				return nil, fmt.Errorf("input %q contains invalid non-blank constraint", input.Name)
			}
		case InputConstraintNumberGreaterThan:
			if !numericConstraintType(input.Type) || len(constraint.Threshold) > 128 {
				return nil, fmt.Errorf("input %q contains invalid numeric constraint", input.Name)
			}
			if _, ok := constraintNumber(constraint.Threshold); !ok {
				return nil, fmt.Errorf("input %q contains invalid numeric threshold", input.Name)
			}
			canonical, err := artifact.Canonicalize([]byte("[" + constraint.Threshold + "]"))
			if err != nil {
				return nil, fmt.Errorf("input %q contains non-canonicalizable numeric threshold: %w", input.Name, err)
			}
			constraint.Threshold = string(canonical[1 : len(canonical)-1])
		default:
			return nil, fmt.Errorf("input %q contains unknown constraint %q", input.Name, constraint.Kind)
		}
	}
	if input.Default != nil {
		for _, constraint := range normalized {
			if InputConstraintViolated(constraint, input.Default) {
				return nil, fmt.Errorf("input %q default violates constraint %q", input.Name, constraint.Kind)
			}
		}
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Kind < normalized[j].Kind })
	return normalized, nil
}

// InputConstraintViolated evaluates one structurally valid constraint.
func InputConstraintViolated(constraint InputConstraint, value any) bool {
	switch constraint.Kind {
	case InputConstraintNonBlank:
		text, ok := value.(string)
		return !ok || strings.TrimSpace(text) == ""
	case InputConstraintNumberGreaterThan:
		actual, actualOK := inputNumber(value)
		threshold, thresholdOK := constraintNumber(constraint.Threshold)
		return !actualOK || !thresholdOK || actual.Cmp(threshold) <= 0
	default:
		return true
	}
}

func validateDeclarativeConstraints(spec *Spec, inputs Inputs) []ValidationError {
	var violations []ValidationError
	for _, input := range spec.Inputs {
		if !inputs.Has(input.Name) {
			continue
		}
		for _, constraint := range input.Constraints {
			if !InputConstraintViolated(constraint, inputs.Raw(input.Name)) {
				continue
			}
			params := map[string]any{"constraint": string(constraint.Kind)}
			if constraint.Threshold != "" {
				params["threshold"] = constraint.Threshold
			}
			violations = append(violations, ValidationError{
				Code: CodeInputConstraintViolation, Message: fmt.Sprintf("input %q violates %q", input.Name, constraint.Kind),
				Field: input.Name, Params: params,
			})
		}
	}
	return violations
}

func numericConstraintType(inputType string) bool {
	switch inputType {
	case "Number", "Integer", "Int", "Duration":
		return true
	default:
		return false
	}
}

func constraintNumber(value string) (*big.Rat, bool) {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, false
	}
	number, ok := decoded.(json.Number)
	if !ok {
		return nil, false
	}
	rational, ok := new(big.Rat).SetString(number.String())
	return rational, ok
}

func inputNumber(value any) (*big.Rat, bool) {
	switch number := value.(type) {
	case json.Number:
		return constraintNumber(number.String())
	case string:
		return constraintNumber(number)
	case float64:
		return constraintNumber(strconv.FormatFloat(number, 'g', -1, 64))
	case float32:
		return constraintNumber(strconv.FormatFloat(float64(number), 'g', -1, 32))
	case int:
		return new(big.Rat).SetInt64(int64(number)), true
	case int8:
		return new(big.Rat).SetInt64(int64(number)), true
	case int16:
		return new(big.Rat).SetInt64(int64(number)), true
	case int32:
		return new(big.Rat).SetInt64(int64(number)), true
	case int64:
		return new(big.Rat).SetInt64(number), true
	case uint:
		return new(big.Rat).SetUint64(uint64(number)), true
	case uint8:
		return new(big.Rat).SetUint64(uint64(number)), true
	case uint16:
		return new(big.Rat).SetUint64(uint64(number)), true
	case uint32:
		return new(big.Rat).SetUint64(uint64(number)), true
	case uint64:
		return new(big.Rat).SetUint64(number), true
	default:
		return nil, false
	}
}
