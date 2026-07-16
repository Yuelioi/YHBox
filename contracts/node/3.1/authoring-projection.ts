/* Generated from Node Authoring Projection 3.1 Go types. Do not edit. */

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

export interface YottaNodeAuthoringProjection {
  body: Body
  format: 'yotta.node-authoring-projection'
  projectionDigest: string
  version: '3.1'
}
export interface Body {
  catalogHash: string
  generatorVersion: string
  nodes: NodeProjection[]
  types: TypeProjection[]
}
export interface NodeProjection {
  availability: 'portable' | 'host-required' | 'target-required' | 'host-and-target-required'
  capabilities: CapabilityProjection[]
  category?: string
  configFields: FieldProjection[]
  dataInputs: PortProjection[]
  dataOutputs: PortProjection[]
  descriptionKey?: string
  editorAdapter?: string
  errors: ErrorSpec[]
  execution: ExecutionSpec
  /**
   * @maxItems 256
   */
  hostFeatureRequirements: HostFeatureRequirement[]
  icon?: string
  instruction: InstructionSpec
  nodeRef: NodeRef
  signals: SignalProjection[]
  stateAccesses: StateAccessProjection[]
  statusEvents: StatusEventSpec[]
  tags: string[]
  titleKey?: string
}
export interface CapabilityProjection {
  capability: Ref
  consent: 'none' | 'once' | 'every-run'
  credential: 'none' | 'required'
  credentialSlot?: string
  credentialSlotConfigKey?: string
  operations: string[]
  requirementId: string
  risk: 'low' | 'sensitive' | 'dangerous'
  scope: any
  targetKinds: string[]
  targetSlot: string
  targetSlotConfigKey?: string
}
export interface Ref {
  capabilityId: string
  semanticDigest: string
}
export interface FieldProjection {
  additionalProperties?: never
  constraints: FieldConstraints
  control: 'text' | 'code' | 'number' | 'integer' | 'toggle' | 'select' | 'object' | 'list' | 'json' | 'state-variable'
  default?: any
  deprecated: boolean
  description?: string
  descriptionKey?: string
  examples: any[]
  hasDefault: boolean
  id: string
  items?: FieldProjection
  properties: FieldProjection[]
  readOnly: boolean
  required: boolean
  title?: string
  titleKey?: string
}
export interface FieldConstraints {
  enum: any[]
  maxItems?: number
  maxLength?: number
  maximum?: any
  minItems?: number
  minLength?: number
  minimum?: any
  pattern?: string
}
export interface PortProjection {
  binding: 'required' | 'optional' | 'default-available' | 'output'
  carrier: 'durable' | 'runtime'
  default?: any
  descriptionKey?: string
  editorAdapter?: string
  hasDefault: boolean
  id: string
  resourceLease?: ResourceLeaseBinding
  titleKey?: string
  type: TypeUse
}
export interface ResourceLeaseBinding {
  /**
   * @minItems 1
   * @maxItems 64
   */
  operations: [string, ...string[]]
  requirementId: string
}
export interface TypeUse {
  color?: string
  constraints: FieldConstraints
  control: 'text' | 'number' | 'integer' | 'toggle' | 'select' | 'object' | 'list' | 'json'
  descriptionKey?: string
  editorAdapter?: string
  examples: any[]
  expression: TypeExpression
  label: string
  lifecycle: 'durable' | 'runtime-only' | 'durable-or-runtime' | 'resolved-at-compile'
  representations: RepresentationSpec[]
  titleKey?: string
  typeIds: string[]
}
export interface TypeRef {
  semanticDigest: string
  typeId: string
}
export interface RepresentationSpec {
  codec: string
  kind: 'inline-json' | 'blob-ref' | 'stream-ref' | 'handle-ref'
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
export interface HostFeatureRequirement {
  featureId: string
  id: string
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
export interface NodeRef {
  nodeTypeId: string
  semanticDigest: string
  version: string
}
export interface SignalProjection {
  channel: 'exec' | 'error'
  direction: 'input' | 'output'
  id: string
}
export interface StateAccessProjection {
  id: string
  mode: 'read' | 'write'
  slotConfigKey: string
  type: TypeUse
}
export interface StatusEventSpec {
  category: 'progress' | 'waiting' | 'connection'
  code: string
}
export interface TypeProjection {
  color?: string
  constraints: FieldConstraints
  control: 'text' | 'number' | 'integer' | 'toggle' | 'select' | 'object' | 'list' | 'json'
  descriptionKey?: string
  editorAdapter?: string
  examples: any[]
  icon?: string
  lifecycle: 'durable' | 'runtime-only' | 'durable-or-runtime' | 'resolved-at-compile'
  representations: RepresentationSpec[]
  schemaRoot: string
  titleKey?: string
  typeRef: TypeRef
}
