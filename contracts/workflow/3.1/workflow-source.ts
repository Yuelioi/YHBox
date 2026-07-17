/* Generated from WorkflowSource Go types. Do not edit. */

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

export interface YottaWorkflowSource {
  entryGraph: string
  format: 'yotta.workflow'
  /**
   * @minItems 1
   * @maxItems 256
   */
  graphs: [Graph, ...Graph[]]
  revision: number
  /**
   * @maxItems 4096
   */
  secretRefs: SecretRef[]
  /**
   * @maxItems 4096
   */
  variables: Variable[]
  version: '3.1'
  workflow: Workflow
}
export interface Graph {
  /**
   * @maxItems 4096
   */
  annotations?: Annotation[]
  /**
   * @maxItems 4096
   */
  calls?: GraphCall[]
  /**
   * @maxItems 16384
   */
  edges: Edge[]
  /**
   * @maxItems 64
   */
  entries?: Endpoint[]
  /**
   * @maxItems 64
   */
  exits?: GraphExit[]
  id: string
  /**
   * @maxItems 4096
   */
  inputs: GraphPort[]
  kind: 'main' | 'subgraph'
  name?: string
  /**
   * @maxItems 4096
   */
  nodes: Node[]
  /**
   * @maxItems 4096
   */
  outputs: GraphPort[]
}
export interface Annotation {
  color?: string
  id: string
  position: Position
  size: Size
  text: string
}
export interface Position {
  x: number
  y: number
}
export interface Size {
  height: number
  width: number
}
export interface GraphCall {
  bindings: {
    [k: string]: InputBinding
  }
  graphId: string
  id: string
  label?: string
  position: Position
}
export interface InputBinding {
  blob?: BlobRef
  kind: 'value' | 'default' | 'blob'
  value?: any
}
export interface BlobRef {
  digest: string
  mediaType: string
  size: number
}
export interface Edge {
  channel: 'data' | 'exec' | 'error'
  from: Endpoint
  presentation?: EdgePresentation
  to: Endpoint
}
export interface Endpoint {
  nodeId: string
  portId: string
}
export interface EdgePresentation {
  /**
   * @maxItems 64
   */
  reroutes?: Position[]
}
export interface GraphExit {
  channel: 'exec' | 'error'
  endpoint: Endpoint
  id: string
}
export interface GraphPort {
  id: string
  nodeId: string
  portId: string
  type: TypeExpression
}
export interface TypeRef {
  semanticDigest: string
  typeId: string
}
export interface Node {
  bindings: {
    [k: string]: InputBinding
  }
  config: {
    [k: string]: any
  }
  disabled?: boolean
  id: string
  label?: string
  nodeRef: NodeRef
  position: Position
}
export interface NodeRef {
  nodeTypeId: string
  semanticDigest: string
  version: string
}
export interface SecretRef {
  id: string
  purpose: string
}
export interface Variable {
  default: any
  name: string
  type: TypeExpression
}
export interface Workflow {
  id: string
  name: string
}
