import type {
  Edge,
  Endpoint,
  Graph,
  GraphExit,
  GraphPort,
  InputBinding,
  TypeExpression,
} from '../../../../contracts/workflow/current/workflow-source'

export type GraphInterfaceItemKind = 'input' | 'output' | 'exit'
export type GraphInterfaceCandidateKind = 'entry' | GraphInterfaceItemKind

export interface GraphInterfaceElementPort {
  id: string
  name?: string
  type: TypeExpression
}

export interface GraphInterfaceElementSignal {
  id: string
  name?: string
  channel: 'exec' | 'error'
  direction: 'input' | 'output'
}

export interface GraphInterfaceElement {
  id: string
  label: string
  dataInputs: GraphInterfaceElementPort[]
  dataOutputs: GraphInterfaceElementPort[]
  signals: GraphInterfaceElementSignal[]
  bindings: Record<string, InputBinding>
}

export interface GraphInterfaceCandidate {
  key: string
  kind: GraphInterfaceCandidateKind
  endpoint: Endpoint
  elementLabel: string
  name: string
  type?: TypeExpression
  channel?: 'exec' | 'error'
  published: boolean
}

export interface GraphInterfaceDraft {
  inputs: GraphPort[]
  outputs: GraphPort[]
  entries: NonNullable<Graph['entries']>
  exits: GraphExit[]
}

export interface GraphInterfaceInferencePreview {
  draft: GraphInterfaceDraft
  added: Array<{ kind: GraphInterfaceCandidateKind; id: string }>
  removed: Array<{ kind: GraphInterfaceCandidateKind; id: string }>
}

export interface GraphInterfaceReference {
  parentGraphId: string
  parentGraphName: string
  callId: string
  callLabel: string
  usage: 'binding' | 'edge'
}

export function graphInterfacePortLabel(port: Pick<GraphPort, 'id' | 'name'>): string {
  return port.name?.trim() || port.id
}

export function graphInterfaceExitLabel(exit: Pick<GraphExit, 'id' | 'name'>): string {
  return exit.name?.trim() || exit.id
}

export function projectGraphInterfaceCandidates(
  graph: Graph,
  elements: GraphInterfaceElement[],
): GraphInterfaceCandidate[] {
  if (graph.kind !== 'subgraph') return []
  const published = publishedEndpoints(graph)
  const candidates: GraphInterfaceCandidate[] = []
  for (const element of elements) {
    for (const signal of element.signals) {
      if (
        signal.direction === 'input' &&
        signal.channel === 'exec' &&
        signal.id === 'in' &&
        !hasIncoming(graph.edges, element.id, signal.id, 'exec')
      ) {
        candidates.push(candidate('entry', element, signal.id, signal.name ?? signal.id, published))
      }
      if (
        signal.direction === 'output' &&
        !hasOutgoing(graph.edges, element.id, signal.id, signal.channel)
      ) {
        candidates.push(
          candidate(
            'exit',
            element,
            signal.id,
            signal.name ?? signal.id,
            published,
            undefined,
            signal.channel,
          ),
        )
      }
    }
    for (const port of element.dataInputs) {
      if (!element.bindings[port.id] && !hasIncoming(graph.edges, element.id, port.id, 'data')) {
        candidates.push(
          candidate('input', element, port.id, port.name ?? port.id, published, port.type),
        )
      }
    }
    for (const port of element.dataOutputs) {
      if (!hasOutgoing(graph.edges, element.id, port.id, 'data')) {
        candidates.push(
          candidate('output', element, port.id, port.name ?? port.id, published, port.type),
        )
      }
    }
  }
  return candidates
}

export function addGraphInterfaceCandidate(
  graph: Graph,
  selected: GraphInterfaceCandidate,
): GraphInterfaceDraft {
  const draft = graphInterfaceDraft(graph)
  if (selected.published) return draft
  if (selected.kind === 'entry') {
    draft.entries = [structuredClone(selected.endpoint)]
    return draft
  }
  const id = uniqueInterfaceID(graph, selected.kind, selected.name)
  const existingNames =
    selected.kind === 'exit'
      ? draft.exits.map(graphInterfaceExitLabel)
      : draft[selected.kind === 'input' ? 'inputs' : 'outputs'].map(graphInterfacePortLabel)
  const name = uniqueDisplayName(existingNames, selected.name)
  if (selected.kind === 'exit') {
    if (!selected.channel) throw new Error('graph exit candidate omitted its channel')
    draft.exits.push({
      id,
      name,
      channel: selected.channel,
      endpoint: structuredClone(selected.endpoint),
    })
    return draft
  }
  if (!selected.type) throw new Error('graph data candidate omitted its type')
  const port: GraphPort = {
    id,
    name,
    type: structuredClone(selected.type),
    nodeId: selected.endpoint.nodeId,
    portId: selected.endpoint.portId,
  }
  draft[selected.kind === 'input' ? 'inputs' : 'outputs'].push(port)
  return draft
}

export function renameGraphInterfaceItem(
  graph: Graph,
  kind: GraphInterfaceItemKind,
  id: string,
  rawName: string,
): GraphInterfaceDraft {
  const draft = graphInterfaceDraft(graph)
  const items = kind === 'exit' ? draft.exits : kind === 'input' ? draft.inputs : draft.outputs
  const item = items.find((candidate) => candidate.id === id)
  if (!item) throw new Error(`graph ${kind} ${id} does not exist`)
  const name = rawName.trim()
  if (name.length > 256) throw new Error('graph interface name exceeds 256 characters')
  const visibleName = name || id
  if (
    items.some(
      (candidate) =>
        candidate.id !== id &&
        normalizedInterfaceName(
          'endpoint' in candidate
            ? graphInterfaceExitLabel(candidate)
            : graphInterfacePortLabel(candidate),
        ) === normalizedInterfaceName(visibleName),
    )
  )
    throw new Error(`graph ${kind} name ${visibleName} is already in use`)
  if (name && name !== id) item.name = name
  else delete item.name
  return draft
}

export function moveGraphInterfaceItem(
  graph: Graph,
  kind: GraphInterfaceItemKind,
  id: string,
  direction: -1 | 1,
): GraphInterfaceDraft {
  const draft = graphInterfaceDraft(graph)
  const items = kind === 'exit' ? draft.exits : kind === 'input' ? draft.inputs : draft.outputs
  const index = items.findIndex((candidate) => candidate.id === id)
  if (index < 0) throw new Error(`graph ${kind} ${id} does not exist`)
  const target = index + direction
  if (target < 0 || target >= items.length) return draft
  ;[items[index], items[target]] = [items[target]!, items[index]!]
  return draft
}

export function removeGraphInterfaceItem(
  graph: Graph,
  kind: GraphInterfaceCandidateKind,
  id = '',
): GraphInterfaceDraft {
  const draft = graphInterfaceDraft(graph)
  if (kind === 'entry') draft.entries = []
  if (kind === 'input') draft.inputs = draft.inputs.filter((port) => port.id !== id)
  if (kind === 'output') draft.outputs = draft.outputs.filter((port) => port.id !== id)
  if (kind === 'exit') draft.exits = draft.exits.filter((exit) => exit.id !== id)
  return draft
}

export function inferGraphInterface(
  graph: Graph,
  candidates: GraphInterfaceCandidate[],
): GraphInterfaceInferencePreview {
  const available = candidates.filter(
    (
      item,
    ): item is GraphInterfaceCandidate & {
      kind: GraphInterfaceItemKind
    } => item.kind !== 'entry',
  )
  const entries = candidates.filter((item) => item.kind === 'entry')
  if (entries.length > 1) throw new Error('subgraph has multiple unconnected execution entries')
  if (!entries.length || !available.some((item) => item.kind === 'exit'))
    throw new Error('subgraph needs an unconnected “in” entry and at least one signal exit')

  const next: GraphInterfaceDraft = { inputs: [], outputs: [], entries: [], exits: [] }
  next.entries = [structuredClone(entries[0]!.endpoint)]
  const usedIDs = new Set<string>()
  const usedNames = new Map<GraphInterfaceItemKind, string[]>([
    ['input', []],
    ['output', []],
    ['exit', []],
  ])
  for (const item of available) {
    const existing = existingItemAtEndpoint(graph, item)
    const id = existing?.id ?? uniqueID(usedIDs, item.kind, item.name)
    const existingName =
      existing && 'endpoint' in existing
        ? graphInterfaceExitLabel(existing)
        : existing
          ? graphInterfacePortLabel(existing)
          : ''
    const name = existingName || uniqueDisplayName(usedNames.get(item.kind)!, item.name)
    usedIDs.add(id)
    usedNames.get(item.kind)!.push(name)
    if (item.kind === 'exit') {
      next.exits.push({
        id,
        name,
        channel: item.channel!,
        endpoint: structuredClone(item.endpoint),
      })
    } else {
      next[item.kind === 'input' ? 'inputs' : 'outputs'].push({
        id,
        name,
        type: structuredClone(item.type!),
        nodeId: item.endpoint.nodeId,
        portId: item.endpoint.portId,
      })
    }
  }

  const before = interfaceIdentities(graphInterfaceDraft(graph))
  const after = interfaceIdentities(next)
  return {
    draft: next,
    added: [...after].filter((identity) => !before.has(identity)).map(interfaceIdentityItem),
    removed: [...before].filter((identity) => !after.has(identity)).map(interfaceIdentityItem),
  }
}

export function graphInterfaceReferences(
  source: { graphs: readonly Graph[] },
  graphId: string,
  kind: GraphInterfaceItemKind,
  id: string,
): GraphInterfaceReference[] {
  const references: GraphInterfaceReference[] = []
  const seen = new Set<string>()
  for (const parent of source.graphs) {
    for (const call of parent.calls ?? []) {
      if (call.graphId !== graphId) continue
      const binding = kind === 'input' && Boolean(call.bindings[id])
      const edge = parent.edges.some((candidate) =>
        referencesInterface(candidate, call.id, kind, id),
      )
      for (const usage of [binding ? 'binding' : '', edge ? 'edge' : ''] as const) {
        if (!usage) continue
        const key = `${parent.id}\0${call.id}\0${usage}`
        if (seen.has(key)) continue
        seen.add(key)
        references.push({
          parentGraphId: parent.id,
          parentGraphName: parent.name?.trim() || parent.id,
          callId: call.id,
          callLabel: call.label?.trim() || call.id,
          usage,
        })
      }
    }
  }
  return references
}

export function graphInterfaceDraft(graph: Graph): GraphInterfaceDraft {
  return {
    inputs: structuredClone(graph.inputs),
    outputs: structuredClone(graph.outputs),
    entries: structuredClone(graph.entries ?? []),
    exits: structuredClone(graph.exits ?? []),
  }
}

function candidate(
  kind: GraphInterfaceCandidateKind,
  element: GraphInterfaceElement,
  portId: string,
  name: string,
  published: Set<string>,
  type?: TypeExpression,
  channel?: 'exec' | 'error',
): GraphInterfaceCandidate {
  const endpoint = { nodeId: element.id, portId }
  return {
    key: `${kind}:${element.id}:${portId}`,
    kind,
    endpoint,
    elementLabel: element.label,
    name,
    type,
    channel,
    published: published.has(interfaceEndpointKey(kind, endpoint)),
  }
}

function publishedEndpoints(graph: Graph): Set<string> {
  return new Set([
    ...(graph.entries ?? []).map((endpoint) => interfaceEndpointKey('entry', endpoint)),
    ...graph.inputs.map((port) =>
      interfaceEndpointKey('input', { nodeId: port.nodeId, portId: port.portId }),
    ),
    ...graph.outputs.map((port) =>
      interfaceEndpointKey('output', { nodeId: port.nodeId, portId: port.portId }),
    ),
    ...(graph.exits ?? []).map((exit) => interfaceEndpointKey('exit', exit.endpoint)),
  ])
}

function interfaceEndpointKey(kind: GraphInterfaceCandidateKind, endpoint: Endpoint): string {
  return `${kind}\0${endpoint.nodeId}\0${endpoint.portId}`
}

function hasIncoming(edges: Edge[], nodeId: string, portId: string, channel: Edge['channel']) {
  return edges.some(
    (edge) => edge.channel === channel && edge.to.nodeId === nodeId && edge.to.portId === portId,
  )
}

function hasOutgoing(edges: Edge[], nodeId: string, portId: string, channel: Edge['channel']) {
  return edges.some(
    (edge) =>
      edge.channel === channel && edge.from.nodeId === nodeId && edge.from.portId === portId,
  )
}

function uniqueInterfaceID(graph: Graph, kind: GraphInterfaceItemKind, name: string): string {
  const used = new Set(
    kind === 'input'
      ? graph.inputs.map((item) => item.id)
      : kind === 'output'
        ? graph.outputs.map((item) => item.id)
        : (graph.exits ?? []).map((item) => item.id),
  )
  return uniqueID(used, kind, name)
}

function uniqueID(used: Set<string>, kind: GraphInterfaceItemKind, name: string): string {
  const clean = name.replace(/[^A-Za-z0-9_-]+/g, '_').replace(/^_+|_+$/g, '') || kind
  let index = 1
  let id = `${kind}_${clean}_${index}`
  while (used.has(id)) {
    index += 1
    id = `${kind}_${clean}_${index}`
  }
  return id
}

function uniqueDisplayName(existing: string[], preferred: string): string {
  const base = preferred.trim() || 'Interface'
  const used = new Set(existing.map(normalizedInterfaceName))
  if (!used.has(normalizedInterfaceName(base))) return base
  let index = 2
  while (used.has(normalizedInterfaceName(`${base} ${index}`))) index += 1
  return `${base} ${index}`
}

function normalizedInterfaceName(value: string): string {
  return value.trim().toLocaleLowerCase()
}

function existingItemAtEndpoint(
  graph: Graph,
  candidate: GraphInterfaceCandidate,
): GraphPort | GraphExit | undefined {
  if (candidate.kind === 'input')
    return graph.inputs.find(
      (port) =>
        port.nodeId === candidate.endpoint.nodeId && port.portId === candidate.endpoint.portId,
    )
  if (candidate.kind === 'output')
    return graph.outputs.find(
      (port) =>
        port.nodeId === candidate.endpoint.nodeId && port.portId === candidate.endpoint.portId,
    )
  if (candidate.kind === 'exit')
    return (graph.exits ?? []).find(
      (exit) =>
        exit.endpoint.nodeId === candidate.endpoint.nodeId &&
        exit.endpoint.portId === candidate.endpoint.portId,
    )
  return undefined
}

function interfaceIdentities(draft: GraphInterfaceDraft): Set<string> {
  return new Set([
    ...(draft.entries.length ? ['entry:in'] : []),
    ...draft.inputs.map((item) => `input:${item.id}`),
    ...draft.outputs.map((item) => `output:${item.id}`),
    ...draft.exits.map((item) => `exit:${item.id}`),
  ])
}

function interfaceIdentityItem(identity: string): {
  kind: GraphInterfaceCandidateKind
  id: string
} {
  const separator = identity.indexOf(':')
  return {
    kind: identity.slice(0, separator) as GraphInterfaceCandidateKind,
    id: identity.slice(separator + 1),
  }
}

function referencesInterface(
  edge: Edge,
  callId: string,
  kind: GraphInterfaceItemKind,
  id: string,
): boolean {
  if (kind === 'input')
    return edge.channel === 'data' && edge.to.nodeId === callId && edge.to.portId === id
  if (kind === 'output')
    return edge.channel === 'data' && edge.from.nodeId === callId && edge.from.portId === id
  return edge.channel !== 'data' && edge.from.nodeId === callId && edge.from.portId === id
}
