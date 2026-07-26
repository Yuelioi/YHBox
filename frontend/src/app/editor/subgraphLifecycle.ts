import type {
  Annotation,
  Edge,
  Graph,
  GraphCall,
  Node,
  YottaWorkflowSource,
} from '../../../../contracts/workflow/current/workflow-source'

export interface GraphCallSite {
  parentGraphId: string
  callId: string
}

export interface ExpandedGraphCall {
  callId: string
  nodes: Node[]
  calls: GraphCall[]
  annotations: Annotation[]
  edges: Edge[]
}

export function duplicateGraphCall(
  graph: Graph,
  callId: string,
  newCallId: string,
  offset = { x: 32, y: 32 },
): GraphCall {
  const call = graph.calls?.find((candidate) => candidate.id === callId)
  if (!call) throw new Error(`graph call ${callId} does not exist`)
  if (graphElementIDs(graph).has(newCallId))
    throw new Error(`graph element ${newCallId} already exists`)
  return {
    ...structuredClone(call),
    id: newCallId,
    position: {
      x: call.position.x + offset.x,
      y: call.position.y + offset.y,
    },
  }
}

export function duplicateGraphDefinition(
  source: Pick<YottaWorkflowSource, 'graphs'>,
  graphId: string,
  newGraphId: string,
  name: string,
): Graph {
  const graph = source.graphs.find((candidate) => candidate.id === graphId)
  if (!graph || graph.kind !== 'subgraph') throw new Error(`subgraph ${graphId} does not exist`)
  if (source.graphs.some((candidate) => candidate.id === newGraphId))
    throw new Error(`graph ${newGraphId} already exists`)
  return {
    ...structuredClone(graph),
    id: newGraphId,
    name: name.trim() || `${graph.name?.trim() || graph.id} Copy`,
  }
}

export function graphCallSites(
  source: Pick<YottaWorkflowSource, 'graphs'>,
  graphId: string,
): GraphCallSite[] {
  return source.graphs.flatMap((graph) =>
    (graph.calls ?? [])
      .filter((call) => call.graphId === graphId)
      .map((call) => ({ parentGraphId: graph.id, callId: call.id })),
  )
}

export function expandGraphCall(
  source: Pick<YottaWorkflowSource, 'graphs'>,
  parentGraphId: string,
  callId: string,
  idFactory: () => string,
): ExpandedGraphCall {
  const parent = source.graphs.find((graph) => graph.id === parentGraphId)
  const call = parent?.calls?.find((candidate) => candidate.id === callId)
  const callee = source.graphs.find((graph) => graph.id === call?.graphId)
  if (!parent || !call || !callee || callee.kind !== 'subgraph')
    throw new Error(`graph call ${callId} does not exist`)

  const used = graphElementIDs(parent)
  const remapped = new Map<string, string>()
  for (const element of [...callee.nodes, ...(callee.calls ?? []), ...(callee.annotations ?? [])]) {
    const id = allocateElementID(used, idFactory, element.id)
    used.add(id)
    remapped.set(element.id, id)
  }
  const origin = graphOrigin(callee)
  const offset = { x: call.position.x - origin.x, y: call.position.y - origin.y }
  const nodes = callee.nodes.map((node) => ({
    ...structuredClone(node),
    id: requireRemappedID(remapped, node.id),
    position: {
      x: node.position.x + offset.x,
      y: node.position.y + offset.y,
    },
  }))
  const calls = (callee.calls ?? []).map((nested) => ({
    ...structuredClone(nested),
    id: requireRemappedID(remapped, nested.id),
    position: {
      x: nested.position.x + offset.x,
      y: nested.position.y + offset.y,
    },
  }))
  const annotations = (callee.annotations ?? []).map((annotation) => ({
    ...structuredClone(annotation),
    id: requireRemappedID(remapped, annotation.id),
    position: {
      x: annotation.position.x + offset.x,
      y: annotation.position.y + offset.y,
    },
  }))
  const edges = callee.edges.map((edge) => remapEdge(edge, remapped))

  for (const [portId, binding] of Object.entries(call.bindings)) {
    const input = callee.inputs.find((candidate) => candidate.id === portId)
    if (!input) throw new Error(`graph input ${portId} does not exist`)
    const targetID = requireRemappedID(remapped, input.nodeId)
    const target =
      nodes.find((node) => node.id === targetID) ?? calls.find((nested) => nested.id === targetID)
    if (!target) throw new Error(`graph input ${portId} target does not exist`)
    target.bindings[input.portId] = structuredClone(binding)
  }

  for (const edge of parent.edges) {
    if (edge.from.nodeId !== callId && edge.to.nodeId !== callId) continue
    const rewritten = structuredClone(edge)
    if (edge.to.nodeId === callId) {
      if (edge.channel === 'exec' && edge.to.portId === 'in') {
        const entry = callee.entries?.[0]
        if (!entry) throw new Error('subgraph call has no execution entry')
        rewritten.to = remapEndpoint(entry.nodeId, entry.portId, remapped)
      } else if (edge.channel === 'data') {
        const input = callee.inputs.find((candidate) => candidate.id === edge.to.portId)
        if (!input) throw new Error(`graph input ${edge.to.portId} does not exist`)
        rewritten.to = remapEndpoint(input.nodeId, input.portId, remapped)
      } else {
        throw new Error(`graph call input ${edge.to.portId} is invalid`)
      }
    }
    if (edge.from.nodeId === callId) {
      if (edge.channel === 'data') {
        const output = callee.outputs.find((candidate) => candidate.id === edge.from.portId)
        if (!output) throw new Error(`graph output ${edge.from.portId} does not exist`)
        rewritten.from = remapEndpoint(output.nodeId, output.portId, remapped)
      } else {
        const exit = callee.exits?.find(
          (candidate) => candidate.id === edge.from.portId && candidate.channel === edge.channel,
        )
        if (!exit) throw new Error(`graph exit ${edge.from.portId} does not exist`)
        rewritten.from = remapEndpoint(exit.endpoint.nodeId, exit.endpoint.portId, remapped)
      }
    }
    edges.push(rewritten)
  }

  return { callId, nodes, calls, annotations, edges }
}

function graphElementIDs(graph: Graph): Set<string> {
  return new Set([
    ...graph.nodes.map((node) => node.id),
    ...(graph.calls ?? []).map((call) => call.id),
    ...(graph.annotations ?? []).map((annotation) => annotation.id),
  ])
}

function allocateElementID(used: Set<string>, idFactory: () => string, originalID: string): string {
  for (let attempt = 0; attempt < 16; attempt += 1) {
    const candidate = idFactory()
    if (validID(candidate) && !used.has(candidate)) return candidate
  }
  const base = `${originalID}_expanded`.replace(/[^A-Za-z0-9._-]+/g, '_')
  let candidate = validID(base) ? base : 'expanded'
  let suffix = 2
  while (used.has(candidate)) {
    candidate = `${base}_${suffix}`
    suffix += 1
  }
  return candidate
}

function validID(id: string): boolean {
  return /^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(id) && id.length <= 128
}

function requireRemappedID(remapped: Map<string, string>, id: string): string {
  const value = remapped.get(id)
  if (!value) throw new Error(`subgraph element ${id} is missing from the expansion`)
  return value
}

function remapEndpoint(nodeId: string, portId: string, remapped: Map<string, string>) {
  return { nodeId: requireRemappedID(remapped, nodeId), portId }
}

function remapEdge(edge: Edge, remapped: Map<string, string>): Edge {
  return {
    ...structuredClone(edge),
    from: remapEndpoint(edge.from.nodeId, edge.from.portId, remapped),
    to: remapEndpoint(edge.to.nodeId, edge.to.portId, remapped),
  }
}

function graphOrigin(graph: Graph): { x: number; y: number } {
  const positions = [
    ...graph.nodes.map((node) => node.position),
    ...(graph.calls ?? []).map((call) => call.position),
    ...(graph.annotations ?? []).map((annotation) => annotation.position),
  ]
  if (!positions.length) return { x: 0, y: 0 }
  return {
    x: Math.min(...positions.map((position) => position.x)),
    y: Math.min(...positions.map((position) => position.y)),
  }
}
