package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	RandomDistributionTypeID = "https://schemas.yotta.dev/types/random/distribution/v1"

	RandomIntegerNodeID = "https://schemas.yotta.dev/nodes/random/integer"
	RandomNumberNodeID  = "https://schemas.yotta.dev/nodes/random/number"
	RandomBooleanNodeID = "https://schemas.yotta.dev/nodes/random/boolean"
	RandomChoiceNodeID  = "https://schemas.yotta.dev/nodes/random/choice"
	ObserveTimeNodeID   = "https://schemas.yotta.dev/nodes/time/observe"

	RandomSampleEffectID = "https://schemas.yotta.dev/effects/random/sample/v1"
	TimeObserveEffectID  = "https://schemas.yotta.dev/effects/time/observe/v1"

	RandomEntropyUnavailableCode = "random.entropy_unavailable"
	RandomInvalidRangeCode       = "random.invalid_range"
	RandomInvalidProbabilityCode = "random.invalid_probability"
	RandomEmptyChoiceCode        = "random.empty_choice"
	TimeObservationFailedCode    = "time.observation_failed"
)

func sealRandomDistributionType() (datatype.Definition, error) {
	return sealStructuredType(
		RandomDistributionTypeID,
		json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"string","enum":["uniform","centered"]
		}`, RandomDistributionTypeID+"/schema")),
		datatype.Authoring{
			TitleKey: "type.random.distribution.title", DescriptionKey: "type.random.distribution.description",
			Color: "#ec4899", Icon: "chart-bell-curve",
			Examples: []json.RawMessage{json.RawMessage(`"uniform"`), json.RawMessage(`"centered"`)},
		},
	)
}

func defineRecordedObservationNodes(types primitiveTypes, distribution datatype.TypeRef) ([]BuiltinDefinition, error) {
	integerType := datatype.RefExpression(types.integerRef)
	numberType := datatype.RefExpression(types.numberRef)
	booleanType := datatype.RefExpression(types.booleanRef)
	distributionType := datatype.RefExpression(distribution)
	elementType := datatype.VariableExpression("T")
	listType := datatype.ListExpression(elementType)

	type spec struct {
		id, entrypoint, conformance, key, category, icon string
		inputs                                           []nodecontract.DataInputPort
		output                                           datatype.TypeExpression
		effect                                           nodecontract.EffectID
		errors                                           []nodecontract.ErrorSpec
	}
	defaultValue := func(value string) *json.RawMessage {
		raw := json.RawMessage(value)
		return &raw
	}
	errorSpec := func(code string) nodecontract.ErrorSpec {
		return nodecontract.ErrorSpec{Code: code, Category: "observation", RetryHint: false}
	}
	specs := []spec{
		{
			id: RandomIntegerNodeID, entrypoint: "random.integer", conformance: "recorded-unbiased-safe-integer-sample/v1",
			key: "node.random.integer", category: "random", icon: "numbers",
			inputs: []nodecontract.DataInputPort{
				{ID: "minimum", Type: integerType, Required: true, Default: defaultValue("0")},
				{ID: "maximum", Type: integerType, Required: true, Default: defaultValue("100")},
				{ID: "distribution", Type: distributionType, Required: true, Default: defaultValue(`"uniform"`)},
			},
			output: integerType, effect: RandomSampleEffectID,
			errors: []nodecontract.ErrorSpec{errorSpec(RandomEntropyUnavailableCode), errorSpec(RandomInvalidRangeCode)},
		},
		{
			id: RandomNumberNodeID, entrypoint: "random.number", conformance: "recorded-finite-number-sample/v1",
			key: "node.random.number", category: "random", icon: "decimal",
			inputs: []nodecontract.DataInputPort{
				{ID: "minimum", Type: numberType, Required: true, Default: defaultValue("0")},
				{ID: "maximum", Type: numberType, Required: true, Default: defaultValue("1")},
				{ID: "distribution", Type: distributionType, Required: true, Default: defaultValue(`"uniform"`)},
			},
			output: numberType, effect: RandomSampleEffectID,
			errors: []nodecontract.ErrorSpec{errorSpec(RandomEntropyUnavailableCode), errorSpec(RandomInvalidRangeCode)},
		},
		{
			id: RandomBooleanNodeID, entrypoint: "random.boolean", conformance: "recorded-bernoulli-sample/v1",
			key: "node.random.boolean", category: "random", icon: "adjustments-bolt",
			inputs: []nodecontract.DataInputPort{{ID: "probability", Type: numberType, Required: true, Default: defaultValue("0.5")}},
			output: booleanType, effect: RandomSampleEffectID,
			errors: []nodecontract.ErrorSpec{errorSpec(RandomEntropyUnavailableCode), errorSpec(RandomInvalidProbabilityCode)},
		},
		{
			id: RandomChoiceNodeID, entrypoint: "random.choice", conformance: "recorded-unbiased-list-element-sample/v1",
			key: "node.random.choice", category: "random", icon: "list-random",
			inputs: []nodecontract.DataInputPort{{ID: "list", Type: listType, Required: true}},
			output: elementType, effect: RandomSampleEffectID,
			errors: []nodecontract.ErrorSpec{errorSpec(RandomEntropyUnavailableCode), errorSpec(RandomEmptyChoiceCode)},
		},
		{
			id: ObserveTimeNodeID, entrypoint: "time.observe", conformance: "recorded-invocation-unix-milliseconds/v1",
			key: "node.time.observe", category: "time", icon: "clock",
			inputs: []nodecontract.DataInputPort{}, output: integerType, effect: TimeObserveEffectID,
			errors: []nodecontract.ErrorSpec{errorSpec(TimeObservationFailedCode)},
		},
	}

	definitions := make([]BuiltinDefinition, 0, len(specs))
	for _, item := range specs {
		configID := item.id + "/config"
		contract, err := nodecontract.Seal(nodecontract.Draft{Version: BuiltinNodeVersion,
			NodeTypeID: item.id, ConfigSchemaRoot: configID, ConfigSchemaBundle: emptyConfigSchema(configID),
			Ports: nodecontract.PortSet{
				DataInputs: item.inputs, DataOutputs: []nodecontract.DataOutputPort{{ID: "result", Type: item.output}},
				ExecInputs: []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
			},
			Execution: conversionExecution(item.effect), Instruction: nodecontract.Invoke(), CapabilityRequirements: []capability.Requirement{}, Errors: item.errors,
			StatusEvents:      []nodecontract.StatusEventSpec{},
			ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
			Authoring: nodecontract.Authoring{
				TitleKey: item.key + ".title", DescriptionKey: item.key + ".description", Category: item.category,
				Tags: []string{item.category, "recorded", "observation"}, Icon: item.icon,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("seal built-in %s: %w", item.id, err)
		}
		definition, err := defineBuiltin(contract, item.entrypoint, "v1", item.conformance, nil)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}
