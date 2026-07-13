/* Generated from WorkflowSource Go types. Do not edit. */

export type Capability = string

export interface YottaWorkflowSourceV3 {
  entryGraph: string
  format: 'yotta.workflow'
  /**
   * @minItems 1
   * @maxItems 256
   */
  graphs: [Graph, ...Graph[]]
  /**
   * @maxItems 4096
   */
  requestedCapabilities: Capability[]
  revision: number
  /**
   * @maxItems 4096
   */
  secretRefs: SecretRef[]
  /**
   * @maxItems 4096
   */
  variables: Variable[]
  version: 3
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
  from: string
  to: string
}
export interface GraphPort {
  id: string
  name: string
  nodeId: string
  type: string
}
export interface Node {
  config: {
    [k: string]: any
  }
  disabled?: boolean
  id: string
  kind: string
  label?: string
  position: Position
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
  type: string
}
export interface Workflow {
  id: string
  name: string
}
