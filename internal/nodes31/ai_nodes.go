package nodes31

import (
	"encoding/json"
	"fmt"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/configvalidator"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const aiImplementationVersion = "v1"

func sealBuiltinConfigValidators() (configvalidator.Registry, error) {
	digest, err := ai.StrictSchemaValidatorDigest()
	if err != nil {
		return configvalidator.Registry{}, err
	}
	return configvalidator.Seal([]configvalidator.Descriptor{{
		ID: ai.StrictSchemaValidatorID, SemanticDigest: digest,
		Validate: func(value any) error {
			text, ok := value.(string)
			if !ok {
				return fmt.Errorf("AI structured output schema must be a string")
			}
			_, err := ai.CompileStructuredOutput("result", json.RawMessage(text))
			return err
		},
	}})
}

func sealAIGenerationCapability() (capability.Definition, error) {
	const scopeID = AIGenerationCapabilityID + "/scope"
	return capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID:    AIGenerationCapabilityID,
		Operations:      []string{ai.OperationGenerate, ai.OperationGenerateStructured},
		TargetKinds:     []string{"ai-model"},
		ScopeSchemaRoot: scopeID,
		ScopeSchemaBundle: []datatype.SchemaResource{{ID: scopeID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","properties":{
				"retention":{"type":"string","enum":["provider-default","no-application-state","zero-retention-required"]},
				"structured":{"type":"boolean"}
			},"required":["retention","structured"],"additionalProperties":false
		}`, scopeID))}},
		Credential: capability.CredentialRequired,
		Risk:       capability.RiskSensitive, Consent: capability.ConsentOnce,
		ProviderABI: ai.ProviderABI,
	})
}

func defineAINodes(stringRef, jsonRef datatype.TypeRef, generation capability.Definition) ([]BuiltinDefinition, nodecontract.Contract, nodecontract.Contract, error) {
	generate, err := sealAINode(AIGenerateNodeID, stringRef, stringRef, generation, false)
	if err != nil {
		return nil, nodecontract.Contract{}, nodecontract.Contract{}, err
	}
	extract, err := sealAINode(AIExtractNodeID, stringRef, jsonRef, generation, true)
	if err != nil {
		return nil, nodecontract.Contract{}, nodecontract.Contract{}, err
	}
	generateDefinition, err := defineBuiltin(generate, "ai.generate", aiImplementationVersion, "provider-native-text-generation/v1", nil)
	if err != nil {
		return nil, nodecontract.Contract{}, nodecontract.Contract{}, err
	}
	extractDefinition, err := defineBuiltin(extract, "ai.extract", aiImplementationVersion, "provider-native-strict-structured-output/v1", nil)
	if err != nil {
		return nil, nodecontract.Contract{}, nodecontract.Contract{}, err
	}
	return []BuiltinDefinition{generateDefinition, extractDefinition}, generate, extract, nil
}

func sealAINode(nodeID string, inputRef, outputRef datatype.TypeRef, generation capability.Definition, structured bool) (nodecontract.Contract, error) {
	schemaID := nodeID + "/config"
	titlePrefix := "node.ai.generate"
	operation := ai.OperationGenerate
	effectID := nodecontract.EffectID(AIGenerateEffectID)
	if structured {
		titlePrefix = "node.ai.extract"
		operation = ai.OperationGenerateStructured
		effectID = nodecontract.EffectID(AIExtractEffectID)
	}
	properties := `
		"slot":{"type":"string","minLength":1,"maxLength":128,"pattern":"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$",
			"x-yotta-title-key":"node.ai.config.slot.title","x-yotta-description-key":"node.ai.config.slot.description"},
		"instructions":{"type":"string","maxLength":65536,
			"x-yotta-title-key":"node.ai.config.instructions.title","x-yotta-description-key":"node.ai.config.instructions.description"},
		"temperature":{"type":"number","minimum":0,"maximum":2,
			"x-yotta-title-key":"node.ai.config.temperature.title","x-yotta-description-key":"node.ai.config.temperature.description"},
		"maxOutputTokens":{"type":"integer","minimum":1,"maximum":1000000,
			"x-yotta-title-key":"node.ai.config.maxOutputTokens.title","x-yotta-description-key":"node.ai.config.maxOutputTokens.description"}`
	required := `"slot"`
	if structured {
		properties += `,
		"schema":{"type":"string","minLength":2,"maxLength":65536,
			"x-yotta-title-key":"node.ai.extract.config.schema.title","x-yotta-description-key":"node.ai.extract.config.schema.description"}`
		required += `,"schema"`
	}
	credentialSlot := "model-credential"
	configValidators := []nodecontract.ConfigValidatorSpec{}
	if structured {
		validatorDigest, err := ai.StrictSchemaValidatorDigest()
		if err != nil {
			return nodecontract.Contract{}, err
		}
		configValidators = []nodecontract.ConfigValidatorSpec{{
			ID: "output-schema", ConfigKey: "schema", ValidatorID: ai.StrictSchemaValidatorID, SemanticDigest: validatorDigest,
		}}
	}
	return nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: nodeID, ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(fmt.Sprintf(`{
			"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","properties":{%s},"required":[%s],"additionalProperties":false
		}`, schemaID, properties, required))}},
		Ports: nodecontract.PortSet{
			DataInputs:  []nodecontract.DataInputPort{{ID: "prompt", Type: datatype.RefExpression(inputRef), Required: true}},
			DataOutputs: []nodecontract.DataOutputPort{{ID: "result", Type: datatype.RefExpression(outputRef)}},
			ExecInputs:  []nodecontract.SignalPort{{ID: "in"}}, ExecOutputs: []nodecontract.SignalPort{{ID: "completed"}},
			ErrorOutputs: []nodecontract.SignalPort{{ID: "failed"}},
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionEffect, Effects: []nodecontract.EffectID{effectID}, Determinism: nodecontract.Recorded,
			Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutRequired,
		},
		Instruction: nodecontract.Invoke(),
		CapabilityRequirements: []capability.Requirement{{
			ID: "model", Capability: generation.Ref(), Operations: []string{operation}, TargetSlot: "model", CredentialSlot: credentialSlot,
			Scope: json.RawMessage(fmt.Sprintf(`{"retention":"no-application-state","structured":%t}`, structured)),
		}},
		RequirementBindings: []nodecontract.RequirementBindingSpec{{
			RequirementID: "model", TargetSlotConfigKey: "slot", CredentialSlotConfigKey: "slot",
		}},
		ConfigValidators:  configValidators,
		Errors:            []nodecontract.ErrorSpec{{Code: "ai.generation_failed", Category: "provider", RetryHint: false}},
		StatusEvents:      []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring: nodecontract.Authoring{
			TitleKey: titlePrefix + ".title", DescriptionKey: titlePrefix + ".description", Category: "ai",
			Tags: []string{"ai", "model", "generation"}, Icon: "sparkles",
		},
	})
}
