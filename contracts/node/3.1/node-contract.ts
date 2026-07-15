/* Generated from Node Contract 3.1 Go types. Do not edit. */

export type InstructionSpec =
  | {
      invoke: InvokeInstruction
      kind: 'invoke'
    }
  | {
      kind: 'run-root'
      runRoot: RunRootInstruction
    }
  | {
      countedLoop: CountedLoopInstruction
      kind: 'counted-loop'
    }
  | {
      forEach: ForEachInstruction
      kind: 'for-each'
    }
  | {
      kind: 'retry'
      retry: RetryInstruction
    }
export type TypeExpression =
  | {
      kind: 'ref'
      ref: TypeRef
    }
  | {
      element: TypeExpression
      kind: 'list'
    }
  | {
      kind: 'union'
      /**
       * @minItems 2
       * @maxItems 64
       */
      members: [TypeExpression, TypeExpression, ...TypeExpression[]]
    }
  | {
      /**
       * @maxItems 32
       */
      constraints?: string[]
      kind: 'variable'
      variable: string
    }

export interface YottaNodeContract31 {
  authoring: Authoring
  format: 'yotta.node-contract'
  nodeRef: NodeRef
  semantic: MachineContract
  version: '3.1'
}
export interface Authoring {
  category?: string
  descriptionKey?: string
  editorAdapter?: string
  icon?: string
  /**
   * @maxItems 4096
   */
  tags: string[]
  titleKey?: string
}
export interface NodeRef {
  nodeTypeId: string
  semanticDigest: string
}
export interface MachineContract {
  /**
   * @maxItems 4096
   */
  capabilityRequirements: Requirement[]
  /**
   * @minItems 1
   * @maxItems 256
   */
  configSchemaBundle: [Resource, ...Resource[]]
  configSchemaRoot: string
  /**
   * @maxItems 4096
   */
  configValidators: ConfigValidatorSpec[]
  /**
   * @maxItems 4096
   */
  errors: ErrorSpec[]
  execution: ExecutionSpec
  /**
   * @minItems 1
   */
  implementationABI: [ABIRequirement, ...ABIRequirement[]]
  instanceResolver?: InstanceResolver
  instruction: InstructionSpec
  nodeTypeId: string
  ports: PortSet
  /**
   * @maxItems 4096
   */
  requirementBindings: RequirementBindingSpec[]
  /**
   * @maxItems 4096
   */
  stateAccesses: StateAccessSpec[]
  /**
   * @maxItems 4096
   */
  statusEvents: StatusEventSpec[]
}
export interface Requirement {
  capability: Ref
  credentialSlot?: string
  id: string
  operations: string[]
  scope: any
  targetSlot: string
}
export interface Ref {
  capabilityId: string
  semanticDigest: string
}
export interface Resource {
  id: string
  schema: {
    [k: string]: any
  }
}
export interface ConfigValidatorSpec {
  configKey: string
  id: string
  semanticDigest: string
  validatorId: string
}
export interface ErrorSpec {
  category: string
  code: string
  retryHint: boolean
}
export interface ExecutionSpec {
  cache: 'none' | 'per-run'
  cancellation: 'cooperative' | 'immediate' | 'unsupported'
  class: 'pure-data' | 'effect' | 'control' | 'event' | 'region' | 'marker' | 'visual'
  determinism: 'deterministic' | 'recorded' | 'nondeterministic'
  /**
   * @maxItems 4096
   */
  effects: string[]
  evaluation: 'pull' | 'push'
  retry: 'never' | 'idempotent' | 'operation-id'
  timeout: 'none' | 'required' | 'optional'
}
export interface ABIRequirement {
  kind: 'builtin' | 'host-instruction' | 'wit' | 'process'
  version: string
}
export interface InstanceResolver {
  maxPorts: number
  resolverId: string
  semanticDigest: string
}
export interface InvokeInstruction {}
export interface RunRootInstruction {
  output: string
}
export interface CountedLoopInstruction {
  bodyOutput: string
  breakInput: string
  completedOutput: string
  continueInput: string
  countInput: string
  entryInput: string
  indexOutput: string
  maxIterations: number
  ordinalType: TypeRef
}
export interface TypeRef {
  semanticDigest: string
  typeId: string
}
export interface ForEachInstruction {
  bodyOutput: string
  breakInput: string
  completedOutput: string
  continueInput: string
  entryInput: string
  indexOutput: string
  itemOutput: string
  itemsInput: string
  maxItems: number
  ordinalType: TypeRef
}
export interface RetryInstruction {
  attemptOutput: string
  attemptsInput: string
  bodyOutput: string
  completedOutput: string
  entryInput: string
  exhaustedOutput: string
  maxAttempts: number
  ordinalType: TypeRef
  retryInput: string
}
export interface PortSet {
  /**
   * @maxItems 4096
   */
  dataInputs: DataInputPort[]
  /**
   * @maxItems 4096
   */
  dataOutputs: DataOutputPort[]
  /**
   * @maxItems 4096
   */
  errorOutputs: SignalPort[]
  /**
   * @maxItems 4096
   */
  execInputs: SignalPort[]
  /**
   * @maxItems 4096
   */
  execOutputs: SignalPort[]
}
export interface DataInputPort {
  default?: any
  id: string
  required: boolean
  resourceLease?: ResourceLeaseBinding
  type: TypeExpression
}
export interface ResourceLeaseBinding {
  /**
   * @minItems 1
   * @maxItems 64
   */
  operations: [string, ...string[]]
  requirementId: string
}
export interface DataOutputPort {
  id: string
  resourceLease?: ResourceLeaseBinding
  type: TypeExpression
}
export interface SignalPort {
  id: string
}
export interface RequirementBindingSpec {
  credentialSlotConfigKey?: string
  requirementId: string
  targetSlotConfigKey?: string
}
export interface StateAccessSpec {
  id: string
  mode: 'read' | 'write'
  slotConfigKey: string
  type: TypeExpression
}
export interface StatusEventSpec {
  category: 'progress' | 'waiting' | 'connection'
  code: string
}
