/* Generated from WorkflowSource Go types. Do not edit. */

export type Capability = string

export interface YottaWorkflowSourceV3 {
  entryGraph: string
  format: 'yotta.workflow'
  /**
   * @minItems 1
   */
  graphs: [Graph, ...Graph[]]
  requestedCapabilities: Capability[]
  revision: number
  secretRefs: SecretRef[]
  variables: Variable[]
  version: 3
  workflow: Workflow
}
export interface Graph {
  edges: Edge[]
  id: string
  inputs: GraphPort[]
  kind: 'main' | 'subgraph'
  nodes: Node[]
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
