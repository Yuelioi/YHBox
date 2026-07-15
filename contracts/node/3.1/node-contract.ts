/* Generated from Node Contract 3.1 Go types. Do not edit. */

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
  errors: ErrorSpec[]
  execution: ExecutionSpec
  /**
   * @minItems 1
   */
  implementationABI: [ABIRequirement, ...ABIRequirement[]]
  instanceResolver?: InstanceResolver
  nodeTypeId: string
  ports: PortSet
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
  kind: 'builtin' | 'wit' | 'process'
  version: string
}
export interface InstanceResolver {
  maxPorts: number
  resolverId: string
  semanticDigest: string
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
  /**
   * @maxItems 4096
   */
  statusOutputs: SignalPort[]
}
export interface DataInputPort {
  default?: any
  id: string
  required: boolean
  type: TypeExpression
}
export interface TypeRef {
  semanticDigest: string
  typeId: string
}
export interface DataOutputPort {
  id: string
  type: TypeExpression
}
export interface SignalPort {
  id: string
}
