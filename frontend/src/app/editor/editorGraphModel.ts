import type {
  Edge,
  Graph,
  YottaWorkflowSource,
} from '../../../../contracts/workflow/current/workflow-source'
import type {
  NodeProjection,
  PortProjection,
  TypeExpression,
} from '../../../../contracts/node/current/authoring-projection'
import type { EditorCommand } from './EditorSession'
import { requireProjection, resolveConfigDependentProjection } from './editorTypeProjection'

export function graphAt(source: YottaWorkflowSource, path: string[]): Graph {
  const graphId = path.at(-1) ?? source.entryGraph
  const graph = source.graphs.find((candidate) => candidate.id === graphId)
  if (!graph) throw new Error(`graph ${graphId} does not exist`)
  return graph
}

export function uniqueNodeId(graph: Graph, idFactory: () => string): string {
  for (let attempt = 0; attempt < 32; attempt += 1) {
    const candidate = idFactory()
    if (/^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(candidate) && !graphElementExists(graph, candidate)) {
      return candidate
    }
  }
  throw new Error('could not allocate a unique node ID')
}

export function uniqueElementId(graph: Graph, idFactory: () => string): string {
  return uniqueNodeId(graph, idFactory)
}

export function uniqueGraphId(source: YottaWorkflowSource, idFactory: () => string): string {
  for (let attempt = 0; attempt < 32; attempt += 1) {
    const candidate = idFactory()
    if (
      /^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(candidate) &&
      !source.graphs.some((graph) => graph.id === candidate)
    )
      return candidate
  }
  throw new Error('could not allocate a unique graph ID')
}

export function normalizeGraph(graph: Graph): Graph {
  graph.calls ??= []
  graph.entries ??= []
  graph.exits ??= []
  graph.annotations ??= []
  return graph
}

export function graphElementExists(graph: Graph, id: string): boolean {
  return (
    graph.nodes.some((node) => node.id === id) ||
    graph.calls!.some((call) => call.id === id) ||
    graph.annotations!.some((annotation) => annotation.id === id)
  )
}

export function collapseGraphSelection(
  source: YottaWorkflowSource,
  graph: Graph,
  command: Extract<EditorCommand, { kind: 'collapse-selection' }>,
  projections: Map<string, NodeProjection>,
): void {
  const selected = new Set(command.nodeIds)
  const nodes = graph.nodes.filter((node) => selected.has(node.id))
  const calls = graph.calls!.filter((call) => selected.has(call.id))
  if (
    nodes.length + calls.length !== selected.size ||
    source.graphs.some((candidate) => candidate.id === command.subgraphId) ||
    graphElementExists(graph, command.callId)
  )
    throw new Error('collapse selection is invalid')
  const subgraph: Graph = normalizeGraph({
    id: command.subgraphId,
    name: command.name.trim() || 'Subgraph',
    kind: 'subgraph',
    nodes: clone(nodes),
    calls: clone(calls),
    edges: [],
    inputs: [],
    outputs: [],
  })
  const parentEdges: Edge[] = []
  const inputs = new Map<string, string>()
  const outputs = new Map<string, string>()
  const exits = new Map<string, string>()
  for (const edge of graph.edges) {
    const fromSelected = selected.has(edge.from.nodeId)
    const toSelected = selected.has(edge.to.nodeId)
    if (fromSelected && toSelected) {
      subgraph.edges.push(clone(edge))
      continue
    }
    if (!fromSelected && !toSelected) {
      parentEdges.push(clone(edge))
      continue
    }
    const copyEdge = clone(edge)
    if (!fromSelected && toSelected) {
      if (edge.channel === 'error') throw new Error('selection has an incoming error route')
      if (edge.channel === 'exec') {
        if (
          subgraph.entries!.length &&
          (subgraph.entries![0].nodeId !== edge.to.nodeId ||
            subgraph.entries![0].portId !== edge.to.portId)
        )
          throw new Error('selection has multiple execution entries')
        if (!subgraph.entries!.length) subgraph.entries = [clone(edge.to)]
        copyEdge.to = { nodeId: command.callId, portId: 'in' }
      } else {
        const key = `${edge.to.nodeId}\0${edge.to.portId}`
        let portId = inputs.get(key)
        if (!portId) {
          const type = graphEndpointType(source, graph, edge.to, false, projections)
          if (!type) throw new Error('selection input type is unavailable')
          portId = boundaryId('input', edge.to.portId, subgraph.inputs.length + 1)
          inputs.set(key, portId)
          subgraph.inputs.push({
            id: portId,
            name: edge.to.portId,
            type: clone(type),
            nodeId: edge.to.nodeId,
            portId: edge.to.portId,
          })
        }
        copyEdge.to = { nodeId: command.callId, portId }
      }
      parentEdges.push(copyEdge)
      continue
    }
    if (edge.channel === 'data') {
      const key = `${edge.from.nodeId}\0${edge.from.portId}`
      let portId = outputs.get(key)
      if (!portId) {
        const type = graphEndpointType(source, graph, edge.from, true, projections)
        if (!type) throw new Error('selection output type is unavailable')
        portId = boundaryId('output', edge.from.portId, subgraph.outputs.length + 1)
        outputs.set(key, portId)
        subgraph.outputs.push({
          id: portId,
          name: edge.from.portId,
          type: clone(type),
          nodeId: edge.from.nodeId,
          portId: edge.from.portId,
        })
      }
      copyEdge.from = { nodeId: command.callId, portId }
    } else {
      const key = `${edge.channel}\0${edge.from.nodeId}\0${edge.from.portId}`
      let exitId = exits.get(key)
      if (!exitId) {
        exitId = boundaryId('exit', edge.from.portId, subgraph.exits!.length + 1)
        exits.set(key, exitId)
        subgraph.exits!.push({
          id: exitId,
          name: edge.from.portId,
          channel: edge.channel,
          endpoint: clone(edge.from),
        })
      }
      copyEdge.from = { nodeId: command.callId, portId: exitId }
    }
    parentEdges.push(copyEdge)
  }
  const hasIncoming = (nodeId: string, portId: string, channel: Edge['channel']) =>
    graph.edges.some(
      (edge) => edge.channel === channel && edge.to.nodeId === nodeId && edge.to.portId === portId,
    )
  const hasOutgoing = (nodeId: string, portId: string, channel: Edge['channel']) =>
    graph.edges.some(
      (edge) =>
        edge.channel === channel && edge.from.nodeId === nodeId && edge.from.portId === portId,
    )
  const addOpenSignals = (
    nodeId: string,
    signals: Array<{
      id: string
      channel: 'exec' | 'error'
      direction: 'input' | 'output'
    }>,
  ) => {
    for (const signal of signals) {
      const endpoint = { nodeId, portId: signal.id }
      if (
        signal.direction === 'input' &&
        signal.channel === 'exec' &&
        !hasIncoming(nodeId, signal.id, signal.channel)
      ) {
        if (
          subgraph.entries!.length &&
          (subgraph.entries![0].nodeId !== nodeId || subgraph.entries![0].portId !== signal.id)
        )
          throw new Error('selection has multiple execution entries')
        if (!subgraph.entries!.length) subgraph.entries = [endpoint]
      }
      if (signal.direction === 'output' && !hasOutgoing(nodeId, signal.id, signal.channel)) {
        subgraph.exits!.push({
          id: boundaryId('exit', signal.id, subgraph.exits!.length + 1),
          name: signal.id,
          channel: signal.channel,
          endpoint,
        })
      }
    }
  }
  for (const node of nodes) {
    const projection = resolveConfigDependentProjection(
      requireProjection(node, projections),
      node.config,
    )
    addOpenSignals(node.id, projection.signals)
  }
  for (const call of calls) {
    const callee = source.graphs.find((candidate) => candidate.id === call.graphId)
    if (!callee) throw new Error(`graph ${call.graphId} does not exist`)
    addOpenSignals(call.id, [
      { id: 'in', channel: 'exec', direction: 'input' },
      ...(callee.exits ?? []).map((exit) => ({
        id: exit.id,
        channel: exit.channel,
        direction: 'output' as const,
      })),
    ])
  }
  if (!subgraph.entries!.length || !subgraph.exits!.length)
    throw new Error('selection needs one execution entry and at least one signal exit')
  graph.nodes = graph.nodes.filter((node) => !selected.has(node.id))
  graph.calls = graph.calls!.filter((call) => !selected.has(call.id))
  graph.edges = parentEdges
  graph.calls!.push({
    id: command.callId,
    graphId: command.subgraphId,
    label: subgraph.name,
    position: clone(command.position),
    bindings: {},
  })
  source.graphs.push(subgraph)
}

export function graphEndpointType(
  source: YottaWorkflowSource,
  graph: Graph,
  endpoint: Edge['from'],
  output: boolean,
  projections: Map<string, NodeProjection>,
): TypeExpression | undefined {
  const node = graph.nodes.find((candidate) => candidate.id === endpoint.nodeId)
  if (node) {
    const projection = requireProjection(node, projections)
    const ports = output ? projection.dataOutputs : projection.dataInputs
    return ports.find((port) => port.id === endpoint.portId)?.type.expression
  }
  const call = graph.calls!.find((candidate) => candidate.id === endpoint.nodeId)
  const callee = source.graphs.find((candidate) => candidate.id === call?.graphId)
  const ports = output ? callee?.outputs : callee?.inputs
  return ports?.find((port) => port.id === endpoint.portId)?.type
}

export function graphSignalEndpointValid(
  source: YottaWorkflowSource,
  graph: Graph,
  endpoint: Edge['from'],
  output: boolean,
  channel: 'exec' | 'error',
  projections: Map<string, NodeProjection>,
): boolean {
  const node = graph.nodes.find((candidate) => candidate.id === endpoint.nodeId)
  if (node) {
    return Boolean(
      projections
        .get(node.nodeRef.nodeTypeId)
        ?.signals.some(
          (signal) =>
            signal.id === endpoint.portId &&
            signal.direction === (output ? 'output' : 'input') &&
            signal.channel === channel,
        ),
    )
  }
  const call = graph.calls!.find((candidate) => candidate.id === endpoint.nodeId)
  const callee = source.graphs.find((candidate) => candidate.id === call?.graphId)
  return output
    ? Boolean(
        callee?.exits?.some((exit) => exit.id === endpoint.portId && exit.channel === channel),
      )
    : Boolean(call && endpoint.portId === 'in' && channel === 'exec')
}

export function resolveGraphInputProjection(
  source: YottaWorkflowSource,
  graph: Graph,
  endpoint: Edge['to'],
  projections: Map<string, NodeProjection>,
  visited: Set<string>,
): PortProjection | undefined {
  const visitKey = `${graph.id}\0${endpoint.nodeId}\0${endpoint.portId}`
  if (visited.has(visitKey)) return undefined
  visited.add(visitKey)
  const node = graph.nodes.find((candidate) => candidate.id === endpoint.nodeId)
  if (node) {
    return projections
      .get(node.nodeRef.nodeTypeId)
      ?.dataInputs.find((port) => port.id === endpoint.portId)
  }
  const call = graph.calls!.find((candidate) => candidate.id === endpoint.nodeId)
  const callee = source.graphs.find((candidate) => candidate.id === call?.graphId)
  const port = callee?.inputs.find((candidate) => candidate.id === endpoint.portId)
  return callee && port
    ? resolveGraphInputProjection(
        source,
        callee,
        { nodeId: port.nodeId, portId: port.portId },
        projections,
        visited,
      )
    : undefined
}

function boundaryId(prefix: string, portId: string, index: number): string {
  const clean = portId.replace(/[^A-Za-z0-9_-]+/g, '_') || prefix
  return `${prefix}_${clean}_${index}`
}

export function defaultNodeId(): string {
  return `node_${crypto.randomUUID().replaceAll('-', '')}`
}

export function sameEdge(left: Edge, right: Edge): boolean {
  return (
    left.channel === right.channel &&
    left.from.nodeId === right.from.nodeId &&
    left.from.portId === right.from.portId &&
    left.to.nodeId === right.to.nodeId &&
    left.to.portId === right.to.portId
  )
}

function clone<T>(value: T): T {
  return structuredClone(value)
}
