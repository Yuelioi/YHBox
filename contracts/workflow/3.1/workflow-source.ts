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

export interface YottaWorkflowSource31 {
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
   * @maxItems 16384
   */
  edges: Edge[]
  id: string
  /**
   * @maxItems 4096
   */
  inputs: GraphPort[]
  kind: 'main' | 'subgraph'
  /**
   * @maxItems 4096
   */
  nodes: Node[]
  /**
   * @maxItems 4096
   */
  outputs: GraphPort[]
}
export interface Edge {
  channel: 'data' | 'exec' | 'error'
  from: Endpoint
  to: Endpoint
}
export interface Endpoint {
  nodeId: string
  portId: string
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
export interface NodeRef {
  nodeTypeId: string
  semanticDigest: string
}
export interface Position {
  x: number
  y: number
}
export interface SecretRef {
  id: string
  purpose: string
}
export interface Variable {
  default?: any
  name: string
  type: TypeExpression
}
export interface Workflow {
  id: string
  name: string
}
