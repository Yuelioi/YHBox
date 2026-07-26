import type { Connection, Edge as FlowEdge, Node as FlowNode } from '@vue-flow/core'
import type {
  Edge,
  Endpoint,
  Graph,
  GraphExit,
  GraphPort,
} from '../../../../contracts/workflow/current/workflow-source'
import { graphHandle, parseGraphHandle } from './graphHandles'

export type GraphBoundaryRole = 'entry' | 'exit' | 'output'

export interface GraphBoundaryNodeData {
  role: GraphBoundaryRole
  graphId: string
  inputs: GraphPort[]
  outputs: GraphPort[]
  exit?: GraphExit
}

export type GraphBoundaryBinding =
  | { kind: 'entry'; endpoint: Endpoint }
  | { kind: 'input'; boundaryId: string; endpoint: Endpoint }
  | { kind: 'exit'; boundaryId: string; endpoint: Endpoint }
  | { kind: 'output'; boundaryId: string; endpoint: Endpoint }

export type GraphBoundaryKey =
  | { kind: 'entry' }
  | { kind: 'input'; boundaryId: string }
  | { kind: 'exit'; boundaryId: string }
  | { kind: 'output'; boundaryId: string }

export interface GraphBoundaryProjection {
  nodes: FlowNode<GraphBoundaryNodeData, Record<string, never>, 'graph-boundary'>[]
  edges: FlowEdge[]
}

export interface CollapseBoundaryIssue {
  kind: 'incoming-error' | 'multiple-entry'
  edges: Edge[]
}

const prefix = '__yotta_authoring_boundary__'

export function projectGraphBoundaries(graph: Graph): GraphBoundaryProjection {
  if (graph.kind !== 'subgraph') return { nodes: [], edges: [] }
  const entryId = graphEntryBoundaryNodeId(graph.id)
  const outputId = graphOutputBoundaryNodeId(graph.id)
  const bounds = elementBounds(graph)
  const entryY = averageEndpointY(graph, graph.entries ?? [], bounds.minY)
  const nodes: GraphBoundaryProjection['nodes'] = [
    boundaryNode(
      entryId,
      { role: 'entry', graphId: graph.id, inputs: graph.inputs, outputs: [] },
      {
        x: bounds.minX - 230,
        y: entryY,
      },
    ),
  ]
  const rightPositions = stackRightBoundaryPositions(graph, bounds.minY)
  for (const exit of graph.exits ?? []) {
    nodes.push(
      boundaryNode(
        graphExitBoundaryNodeId(graph.id, exit.id),
        { role: 'exit', graphId: graph.id, inputs: [], outputs: [], exit },
        { x: bounds.maxX + 280, y: rightPositions.get(`exit:${exit.id}`) ?? bounds.minY },
      ),
    )
  }
  if (graph.outputs.length) {
    nodes.push(
      boundaryNode(
        outputId,
        { role: 'output', graphId: graph.id, inputs: [], outputs: graph.outputs },
        { x: bounds.maxX + 280, y: rightPositions.get('output') ?? bounds.minY },
      ),
    )
  }

  const edges: FlowEdge[] = []
  for (const [index, endpoint] of (graph.entries ?? []).entries()) {
    edges.push(
      boundaryEdge(
        `${prefix}:edge:entry:${graph.id}:${index}`,
        entryId,
        graphHandle('exec', 'output', 'in'),
        endpoint.nodeId,
        graphHandle('exec', 'input', endpoint.portId),
        { kind: 'entry' },
        'var(--ui-primary)',
      ),
    )
  }
  for (const input of graph.inputs) {
    edges.push(
      boundaryEdge(
        `${prefix}:edge:input:${graph.id}:${input.id}`,
        entryId,
        graphHandle('data', 'output', input.id),
        input.nodeId,
        graphHandle('data', 'input', input.portId),
        { kind: 'input', boundaryId: input.id },
        'var(--ui-primary)',
      ),
    )
  }
  for (const exit of graph.exits ?? []) {
    edges.push(
      boundaryEdge(
        `${prefix}:edge:exit:${graph.id}:${exit.id}`,
        exit.endpoint.nodeId,
        graphHandle(exit.channel, 'output', exit.endpoint.portId),
        graphExitBoundaryNodeId(graph.id, exit.id),
        graphHandle(exit.channel, 'input', 'in'),
        { kind: 'exit', boundaryId: exit.id },
        exit.channel === 'error' ? 'var(--ui-error)' : 'var(--ui-success)',
      ),
    )
  }
  for (const output of graph.outputs) {
    edges.push(
      boundaryEdge(
        `${prefix}:edge:output:${graph.id}:${output.id}`,
        output.nodeId,
        graphHandle('data', 'output', output.portId),
        outputId,
        graphHandle('data', 'input', output.id),
        { kind: 'output', boundaryId: output.id },
        'var(--ui-primary)',
      ),
    )
  }
  return { nodes, edges }
}

export function graphBoundaryBindingFromConnection(
  connection: Pick<Connection, 'source' | 'sourceHandle' | 'target' | 'targetHandle'>,
  graph: Graph,
): GraphBoundaryBinding | null {
  if (graph.kind !== 'subgraph') return null
  const source = parseGraphHandle(connection.sourceHandle)
  const target = parseGraphHandle(connection.targetHandle)
  if (!source || !target || source.direction !== 'output' || target.direction !== 'input') {
    return null
  }
  if (connection.source === graphEntryBoundaryNodeId(graph.id)) {
    if (source.channel === 'exec' && source.portId === 'in' && target.channel === 'exec') {
      return { kind: 'entry', endpoint: { nodeId: connection.target, portId: target.portId } }
    }
    if (source.channel === 'data' && target.channel === 'data') {
      const input = graph.inputs.find((port) => port.id === source.portId)
      return input
        ? {
            kind: 'input',
            boundaryId: input.id,
            endpoint: { nodeId: connection.target, portId: target.portId },
          }
        : null
    }
  }
  const exit = (graph.exits ?? []).find(
    (item) => connection.target === graphExitBoundaryNodeId(graph.id, item.id),
  )
  if (exit && source.channel === exit.channel && target.channel === exit.channel) {
    return {
      kind: 'exit',
      boundaryId: exit.id,
      endpoint: { nodeId: connection.source, portId: source.portId },
    }
  }
  if (
    connection.target === graphOutputBoundaryNodeId(graph.id) &&
    source.channel === 'data' &&
    target.channel === 'data'
  ) {
    const output = graph.outputs.find((port) => port.id === target.portId)
    return output
      ? {
          kind: 'output',
          boundaryId: output.id,
          endpoint: { nodeId: connection.source, portId: source.portId },
        }
      : null
  }
  return null
}

export function graphBoundaryKeyFromEdge(edge: FlowEdge): GraphBoundaryKey | null {
  const value = (edge.data as { boundaryKey?: unknown } | undefined)?.boundaryKey
  if (!value || typeof value !== 'object') return null
  const candidate = value as Record<string, unknown>
  if (candidate.kind === 'entry') return { kind: 'entry' }
  if (
    (candidate.kind === 'input' || candidate.kind === 'exit' || candidate.kind === 'output') &&
    typeof candidate.boundaryId === 'string'
  ) {
    return { kind: candidate.kind, boundaryId: candidate.boundaryId } as GraphBoundaryKey
  }
  return null
}

export function isGraphBoundaryNodeId(nodeId: string): boolean {
  return nodeId.startsWith(`${prefix}:node:`)
}

export function analyzeCollapseBoundary(
  graph: Graph,
  selected: ReadonlySet<string>,
): CollapseBoundaryIssue | null {
  const incoming = graph.edges.filter(
    (edge) => !selected.has(edge.from.nodeId) && selected.has(edge.to.nodeId),
  )
  const incomingError = incoming.filter((edge) => edge.channel === 'error')
  if (incomingError.length) return { kind: 'incoming-error', edges: incomingError }
  const executionEntries = incoming.filter((edge) => edge.channel === 'exec')
  const endpoints = new Set(executionEntries.map((edge) => `${edge.to.nodeId}\0${edge.to.portId}`))
  return endpoints.size > 1 ? { kind: 'multiple-entry', edges: executionEntries } : null
}

export function graphEntryBoundaryNodeId(graphId: string): string {
  return `${prefix}:node:entry:${encodeURIComponent(graphId)}`
}

function graphExitBoundaryNodeId(graphId: string, exitId: string): string {
  return `${prefix}:node:exit:${encodeURIComponent(graphId)}:${encodeURIComponent(exitId)}`
}

function graphOutputBoundaryNodeId(graphId: string): string {
  return `${prefix}:node:output:${encodeURIComponent(graphId)}`
}

function boundaryNode(
  id: string,
  data: GraphBoundaryNodeData,
  position: { x: number; y: number },
): GraphBoundaryProjection['nodes'][number] {
  return {
    id,
    type: 'graph-boundary',
    position,
    data,
    draggable: false,
    selectable: false,
    connectable: true,
    deletable: false,
  }
}

function boundaryEdge(
  id: string,
  source: string,
  sourceHandle: string,
  target: string,
  targetHandle: string,
  boundaryKey: GraphBoundaryKey,
  stroke: string,
): FlowEdge {
  return {
    id,
    source,
    sourceHandle,
    target,
    targetHandle,
    animated: false,
    data: { boundaryKey },
    style: { stroke, strokeWidth: 2, strokeDasharray: '6 4' },
  }
}

function elementBounds(graph: Graph): { minX: number; maxX: number; minY: number } {
  const positions = [
    ...graph.nodes.map((node) => node.position),
    ...(graph.calls ?? []).map((call) => call.position),
  ]
  if (!positions.length) return { minX: 280, maxX: 280, minY: 80 }
  return {
    minX: Math.min(...positions.map((position) => position.x)),
    maxX: Math.max(...positions.map((position) => position.x)),
    minY: Math.min(...positions.map((position) => position.y)),
  }
}

function endpointY(graph: Graph, endpoint: Endpoint, fallback: number): number {
  return (
    graph.nodes.find((node) => node.id === endpoint.nodeId)?.position.y ??
    graph.calls?.find((call) => call.id === endpoint.nodeId)?.position.y ??
    fallback
  )
}

function averageEndpointY(graph: Graph, endpoints: Endpoint[], fallback: number): number {
  if (!endpoints.length) return fallback
  return (
    endpoints.reduce((sum, endpoint) => sum + endpointY(graph, endpoint, fallback), 0) /
    endpoints.length
  )
}

function stackRightBoundaryPositions(graph: Graph, fallback: number): Map<string, number> {
  const items = (graph.exits ?? [])
    .map((exit) => ({ key: `exit:${exit.id}`, y: endpointY(graph, exit.endpoint, fallback) }))
    .sort((left, right) => left.y - right.y || left.key.localeCompare(right.key))
  if (graph.outputs.length) {
    items.push({
      key: 'output',
      y: averageEndpointY(
        graph,
        graph.outputs.map((port) => ({ nodeId: port.nodeId, portId: port.portId })),
        fallback,
      ),
    })
  }
  const result = new Map<string, number>()
  let previous = Number.NEGATIVE_INFINITY
  for (const item of items) {
    const y = Math.max(item.y, previous + 96)
    result.set(item.key, y)
    previous = y
  }
  return result
}
