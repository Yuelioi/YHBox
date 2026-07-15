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

export interface YottaNodeAuthoringProjection31 {
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
  availability: 'portable' | 'target-required'
  capabilities: CapabilityProjection[]
  category?: string
  configFields: FieldProjection[]
  dataInputs: PortProjection[]
  dataOutputs: PortProjection[]
  descriptionKey?: string
  editorAdapter?: string
  errors: ErrorSpec[]
  execution: ExecutionSpec
  icon?: string
  nodeRef: NodeRef
  signals: SignalProjection[]
  statusEvents: StatusEventSpec[]
  tags: string[]
  titleKey?: string
}
export interface CapabilityProjection {
  capability: Ref
  consent: 'none' | 'once' | 'every-run'
  credential: 'none' | 'required'
  credentialSlot?: string
  operations: string[]
  requirementId: string
  risk: 'low' | 'sensitive' | 'dangerous'
  scope: any
  targetKinds: string[]
  targetSlot: string
}
export interface Ref {
  capabilityId: string
  semanticDigest: string
}
export interface FieldProjection {
  additionalProperties?: never
  constraints: FieldConstraints
  control: 'text' | 'number' | 'integer' | 'toggle' | 'select' | 'object' | 'list' | 'json'
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
  hasDefault: boolean
  id: string
  resourceLease?: ResourceLeaseBinding
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
export interface NodeRef {
  nodeTypeId: string
  semanticDigest: string
}
export interface SignalProjection {
  channel: 'exec' | 'error'
  direction: 'input' | 'output'
  id: string
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
