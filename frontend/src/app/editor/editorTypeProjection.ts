import type {
  Edge,
  Graph,
  GraphCall,
  Node,
  YottaWorkflowSource,
} from '../../../../contracts/workflow/current/workflow-source'
import type {
  NodeProjection,
  TypeExpression,
  TypeProjection,
  TypeUse,
} from '../../../../contracts/node/current/authoring-projection'
import { projectedConnectionCompatibility } from './connectionCompatibility'

const PLAY_INPUT_CLIP_NODE_TYPE_ID = 'https://schemas.yotta.dev/nodes/automation/play-input-clip'
const PLAY_INPUT_CLIP_RETRACTED_SCALE_DIGEST =
  'sha256:ff7ea9d0b2ca91cb2062cff30dd5ca8575555ec5363b4c76e746925ee6ae027b'

export function stateTypeHasDefault(type: TypeProjection): boolean {
  return type.examples.length > 0 || type.control !== 'object'
}

export function defaultStateValue(type: TypeProjection): unknown {
  if (type.examples.length) return clone(type.examples[0])
  switch (type.control) {
    case 'text':
      return ''
    case 'number':
    case 'integer':
      return 0
    case 'toggle':
      return false
    case 'select':
      return type.constraints.enum[0] ?? null
    case 'list':
      return []
    case 'object':
      return {}
    default:
      return null
  }
}

export function validateEdge(
  source: YottaWorkflowSource,
  graph: Graph,
  edge: Edge,
  projections: Map<string, NodeProjection>,
  types: Map<string, TypeProjection>,
): void {
  const fromNode = graph.nodes.find((node) => node.id === edge.from.nodeId)
  const toNode = graph.nodes.find((node) => node.id === edge.to.nodeId)
  if (!fromNode || !toNode) {
    const fromCall = graph.calls!.find((call) => call.id === edge.from.nodeId)
    const toCall = graph.calls!.find((call) => call.id === edge.to.nodeId)
    if ((!fromNode && !fromCall) || (!toNode && !toCall))
      throw new Error('edge endpoint does not exist')
    const callee = (call: GraphCall | undefined) =>
      source.graphs.find((candidate) => candidate.id === call?.graphId)
    if (edge.channel === 'data') {
      const fromValid = fromCall
        ? callee(fromCall)?.outputs.some((port) => port.id === edge.from.portId)
        : requireProjection(fromNode!, projections).dataOutputs.some(
            (port) => port.id === edge.from.portId,
          )
      const toValid = toCall
        ? callee(toCall)?.inputs.some((port) => port.id === edge.to.portId)
        : requireProjection(toNode!, projections).dataInputs.some(
            (port) => port.id === edge.to.portId,
          )
      if (fromValid && toValid) return
    } else {
      const fromValid = fromCall
        ? callee(fromCall)?.exits?.some(
            (exit) => exit.id === edge.from.portId && exit.channel === edge.channel,
          )
        : requireProjection(fromNode!, projections).signals.some(
            (signal) =>
              signal.id === edge.from.portId &&
              signal.channel === edge.channel &&
              signal.direction === 'output',
          )
      const toValid = toCall
        ? edge.channel === 'exec' && edge.to.portId === 'in'
        : requireProjection(toNode!, projections).signals.some(
            (signal) => signal.id === edge.to.portId && signal.direction === 'input',
          )
      if (fromValid && toValid) return
    }
    throw new Error('edge is incompatible')
  }
  const from = resolveNodeInstanceProjection(
    source,
    fromNode,
    requireProjection(fromNode, projections),
    projections,
    types,
  )
  const to = resolveNodeInstanceProjection(
    source,
    toNode,
    requireProjection(toNode, projections),
    projections,
    types,
  )
  const result = projectedConnectionCompatibility(
    from,
    { channel: edge.channel, direction: 'output', portId: edge.from.portId },
    to,
    { channel: edge.channel, direction: 'input', portId: edge.to.portId },
    types,
  )
  if (!result.valid) throw new Error(result.message ?? 'edge is incompatible')
}

export function requireNode(graph: Graph, nodeId: string): Node {
  const node = graph.nodes.find((candidate) => candidate.id === nodeId)
  if (!node) throw new Error(`node ${nodeId} does not exist`)
  return node
}

export function requireProjection(
  node: Node,
  projections: Map<string, NodeProjection>,
): NodeProjection {
  const projection = projections.get(node.nodeRef.nodeTypeId)
  if (!projection || projection.nodeRef.semanticDigest !== node.nodeRef.semanticDigest) {
    throw new Error(`node ${node.id} has no exact authoring projection`)
  }
  return projection
}

export function resolveNodeInstanceProjection(
  source: YottaWorkflowSource | null,
  node: Node,
  base: NodeProjection,
  projections: ReadonlyMap<string, NodeProjection>,
  types: ReadonlyMap<string, TypeProjection>,
): NodeProjection {
  const effective = resolveConfigDependentProjection(base, node.config)
  const graph = source?.graphs.find((candidate) =>
    candidate.nodes.some((graphNode) => graphNode.id === node.id),
  )
  const bindings = graph
    ? (solveGraphTypeBindings(source, graph, projections, types).get(node.id) ?? new Map())
    : stateTypeBindings(source, node, effective)
  if (!bindings.size) return effective
  const projection = clone(effective)
  projection.dataInputs = projection.dataInputs.map((port) => ({
    ...port,
    type: specializeTypeUse(port.type, bindings, types),
  }))
  projection.dataOutputs = projection.dataOutputs.map((port) => ({
    ...port,
    type: specializeTypeUse(port.type, bindings, types),
  }))
  projection.stateAccesses = projection.stateAccesses.map((access) => ({
    ...access,
    type: specializeTypeUse(access.type, bindings, types),
  }))
  return projection
}

// Advances stale contracts when the current projection can represent all
// existing authoring data and topology. Missing required values remain normal
// diagnostics instead of keeping a stale digest that hides their real repair.
export function applyCompatibleNodeContractUpgrade(
  graph: Graph,
  node: Node,
  base: NodeProjection,
): boolean {
  if (base.nodeRef.nodeTypeId !== node.nodeRef.nodeTypeId) return false
  if (base.nodeRef.semanticDigest === node.nodeRef.semanticDigest) return true
  if (base.nodeRef.version !== node.nodeRef.version) return false

  const fields = new Map(base.configFields.map((field) => [field.id, field]))
  if (Object.keys(node.config).some((fieldId) => !fields.has(fieldId))) return false
  const config = clone(node.config)
  for (const field of base.configFields) {
    if (!(field.id in config) && field.hasDefault) config[field.id] = clone(field.default)
  }
  const projection = resolveConfigDependentProjection(base, config)
  const inputs = new Map(projection.dataInputs.map((port) => [port.id, port]))
  const bindings = clone(node.bindings)
  if (
    node.nodeRef.nodeTypeId === PLAY_INPUT_CLIP_NODE_TYPE_ID &&
    node.nodeRef.semanticDigest === PLAY_INPUT_CLIP_RETRACTED_SCALE_DIGEST
  ) {
    delete bindings['turn-scale']
  }
  if (Object.keys(bindings).some((portId) => !inputs.has(portId))) return false

  const incoming = new Set<string>()
  for (const edge of graph.edges) {
    if (
      edge.from.nodeId === node.id &&
      !projectedEndpointExists(projection, edge.from.portId, edge.channel, true)
    ) {
      return false
    }
    if (edge.to.nodeId === node.id) {
      if (!projectedEndpointExists(projection, edge.to.portId, edge.channel, false)) return false
      if (edge.channel === 'data') incoming.add(edge.to.portId)
    }
  }
  for (const port of graph.inputs) {
    if (port.nodeId !== node.id) continue
    if (!projectedEndpointExists(projection, port.portId, 'data', false)) return false
    incoming.add(port.portId)
  }
  for (const port of graph.outputs) {
    if (
      port.nodeId === node.id &&
      !projectedEndpointExists(projection, port.portId, 'data', true)
    ) {
      return false
    }
  }
  for (const entry of graph.entries ?? []) {
    if (
      entry.nodeId === node.id &&
      !projectedEndpointExists(projection, entry.portId, 'exec', false)
    ) {
      return false
    }
  }
  for (const exit of graph.exits ?? []) {
    if (
      exit.endpoint.nodeId === node.id &&
      !projectedEndpointExists(projection, exit.endpoint.portId, exit.channel, true)
    ) {
      return false
    }
  }

  for (const port of projection.dataInputs) {
    if (port.id in bindings || incoming.has(port.id)) continue
    if (port.hasDefault) bindings[port.id] = { kind: 'default' }
  }
  node.config = config
  node.bindings = bindings
  node.nodeRef = clone(projection.nodeRef)
  return true
}

function projectedEndpointExists(
  projection: NodeProjection,
  portId: string,
  channel: Edge['channel'],
  output: boolean,
): boolean {
  if (channel === 'data') {
    const ports = output ? projection.dataOutputs : projection.dataInputs
    return ports.some((port) => port.id === portId)
  }
  return projection.signals.some(
    (signal) =>
      signal.id === portId &&
      signal.channel === channel &&
      signal.direction === (output ? 'output' : 'input'),
  )
}

export function nodeKey(graphId: string, nodeId: string): string {
  return `${graphId}\u0000${nodeId}`
}

function solveGraphTypeBindings(
  source: YottaWorkflowSource | null,
  graph: Graph,
  projections: ReadonlyMap<string, NodeProjection>,
  types: ReadonlyMap<string, TypeProjection>,
): Map<string, Map<string, TypeExpression>> {
  const result = new Map<string, Map<string, TypeExpression>>()
  for (const node of graph.nodes) {
    const base = projections.get(node.nodeRef.nodeTypeId)
    const projection = base ? resolveConfigDependentProjection(base, node.config) : undefined
    result.set(node.id, projection ? stateTypeBindings(source, node, projection) : new Map())
  }
  const budget = Math.max(1, graph.nodes.length + graph.edges.length) * 2
  for (let iteration = 0; iteration < budget; iteration += 1) {
    let changed = false
    for (const edge of graph.edges) {
      if (edge.channel !== 'data') continue
      const fromNode = graph.nodes.find((candidate) => candidate.id === edge.from.nodeId)
      const toNode = graph.nodes.find((candidate) => candidate.id === edge.to.nodeId)
      const fromBase = fromNode ? projections.get(fromNode.nodeRef.nodeTypeId) : undefined
      const toBase = toNode ? projections.get(toNode.nodeRef.nodeTypeId) : undefined
      const from =
        fromNode && fromBase
          ? resolveConfigDependentProjection(fromBase, fromNode.config)
          : undefined
      const to =
        toNode && toBase ? resolveConfigDependentProjection(toBase, toNode.config) : undefined
      const output = from?.dataOutputs.find((port) => port.id === edge.from.portId)?.type.expression
      const input = to?.dataInputs.find((port) => port.id === edge.to.portId)?.type.expression
      if (!fromNode || !toNode || !output || !input) continue
      const fromBindings = result.get(fromNode.id)!
      const toBindings = result.get(toNode.id)!
      const resolvedOutput = specializeTypeExpression(output, fromBindings)
      const resolvedInput = specializeTypeExpression(input, toBindings)
      changed = bindTypeVariables(input, resolvedOutput, toBindings, types, true) || changed
      changed = bindTypeVariables(output, resolvedInput, fromBindings, types, false) || changed
    }
    if (!changed) break
  }
  return result
}

const SWITCH_INSTANCE_RESOLVER = 'https://schemas.yotta.dev/resolvers/control/switch/v1'

export function resolveConfigDependentProjection(
  base: NodeProjection,
  config: Record<string, unknown>,
): NodeProjection {
  if (!base.instanceResolver) return base
  if (base.instanceResolver.resolverId !== SWITCH_INSTANCE_RESOLVER)
    throw new Error(`unsupported instance resolver ${base.instanceResolver.resolverId}`)
  const configured = config.caseCount
  const count =
    configured === undefined
      ? 8
      : typeof configured === 'number' && Number.isInteger(configured)
        ? configured
        : 0
  if (count < 1 || count > 32) throw new Error('switch case count must be between 1 and 32')
  const prototype = base.dataInputs.find((port) => port.id === 'value')
  if (!prototype) throw new Error('switch value input prototype is missing')
  const projection = clone(base)
  for (let index = 1; index <= count; index += 1) {
    const id = `case-${index}`
    projection.dataInputs.push({
      ...clone(prototype),
      id,
      titleKey: `node.control.switch.input.case-${index}.title`,
      descriptionKey: `node.control.switch.input.case-${index}.description`,
      binding: 'optional',
      hasDefault: false,
      order: index + 1,
      importance: 'common',
    })
    projection.signals.push({ id, channel: 'exec', direction: 'output' })
  }
  return projection
}

function stateTypeBindings(
  source: YottaWorkflowSource | null,
  node: Node,
  projection: NodeProjection,
): Map<string, TypeExpression> {
  const bindings = new Map<string, TypeExpression>()
  for (const access of projection.stateAccesses) {
    if (access.type.expression.kind !== 'variable') continue
    const slotName = node.config[access.slotConfigKey]
    const slot =
      typeof slotName === 'string'
        ? source?.variables.find((candidate) => candidate.name === slotName)
        : undefined
    if (slot) bindings.set(access.type.expression.variable, slot.type)
  }
  return bindings
}

function bindTypeVariables(
  pattern: TypeExpression,
  concrete: TypeExpression,
  bindings: Map<string, TypeExpression>,
  types: ReadonlyMap<string, TypeProjection>,
  allowWiden: boolean,
): boolean {
  if (pattern.kind === 'variable') {
    if (!isConcreteTypeExpression(concrete)) return false
    const existing = bindings.get(pattern.variable)
    if (!existing) {
      bindings.set(pattern.variable, clone(concrete))
      return true
    }
    if (!allowWiden) return false
    const merged = mergeTypeEvidence(existing, concrete, types)
    if (JSON.stringify(existing) === JSON.stringify(merged)) return false
    bindings.set(pattern.variable, merged)
    return true
  }
  if (pattern.kind === 'list' && concrete.kind === 'list') {
    return bindTypeVariables(pattern.element, concrete.element, bindings, types, allowWiden)
  }
  return false
}

function isConcreteTypeExpression(expression: TypeExpression): boolean {
  if (expression.kind === 'variable') return false
  if (expression.kind === 'list') return isConcreteTypeExpression(expression.element)
  if (expression.kind === 'union') return expression.members.every(isConcreteTypeExpression)
  return true
}

function mergeTypeEvidence(
  existing: TypeExpression,
  incoming: TypeExpression,
  types: ReadonlyMap<string, TypeProjection>,
): TypeExpression {
  if (JSON.stringify(existing) === JSON.stringify(incoming)) return existing
  if (existing.kind !== 'ref' || incoming.kind !== 'ref') return existing
  const existingType = types.get(existing.ref.typeId)
  if (
    existingType?.assignableTo.some(
      (target) =>
        target.typeId === incoming.ref.typeId &&
        target.semanticDigest === incoming.ref.semanticDigest,
    )
  )
    return clone(incoming)
  return existing
}

function specializeTypeUse(
  use: TypeUse,
  bindings: ReadonlyMap<string, TypeExpression>,
  types: ReadonlyMap<string, TypeProjection>,
): TypeUse {
  const expression = specializeTypeExpression(use.expression, bindings)
  const result = { ...use, expression }
  if (expression.kind !== 'ref') return result
  const projection = types.get(expression.ref.typeId)
  if (!projection || projection.typeRef.semanticDigest !== expression.ref.semanticDigest)
    return result
  return {
    ...result,
    label: expression.ref.typeId,
    control: projection.control,
    color: projection.color,
    titleKey: projection.titleKey,
    descriptionKey: projection.descriptionKey,
    representations: clone(projection.representations),
    lifecycle: projection.lifecycle,
    constraints: clone(projection.constraints),
    examples: clone(projection.examples),
    editorAdapter: projection.editorAdapter,
    typeIds: [expression.ref.typeId],
  }
}

function specializeTypeExpression(
  expression: TypeExpression,
  bindings: ReadonlyMap<string, TypeExpression>,
): TypeExpression {
  if (expression.kind === 'variable') return clone(bindings.get(expression.variable) ?? expression)
  if (expression.kind === 'list') {
    return { kind: 'list', element: specializeTypeExpression(expression.element, bindings) }
  }
  if (expression.kind === 'union') {
    const [first, second, ...rest] = expression.members
    return {
      kind: 'union',
      members: [
        specializeTypeExpression(first, bindings),
        specializeTypeExpression(second, bindings),
        ...rest.map((member) => specializeTypeExpression(member, bindings)),
      ],
    }
  }
  return clone(expression)
}

export function requireDataInput(
  node: Node,
  portId: string,
  projections: Map<string, NodeProjection>,
): NodeProjection['dataInputs'][number] {
  const port = resolveConfigDependentProjection(
    requireProjection(node, projections),
    node.config,
  ).dataInputs.find((candidate) => candidate.id === portId)
  if (!port) throw new Error(`node ${node.id} has no data input ${portId}`)
  return port
}

export function pruneConfigDependentTopology(
  graph: Graph,
  node: Node,
  projections: Map<string, NodeProjection>,
): void {
  const projection = resolveConfigDependentProjection(
    requireProjection(node, projections),
    node.config,
  )
  if (!projection.instanceResolver) return
  const inputs = new Set(projection.dataInputs.map((port) => port.id))
  const outputs = new Set(
    projection.signals
      .filter((signal) => signal.direction === 'output')
      .map((signal) => `${signal.channel}\u0000${signal.id}`),
  )
  for (const portId of Object.keys(node.bindings)) {
    if (!inputs.has(portId)) delete node.bindings[portId]
  }
  graph.edges = graph.edges.filter((edge) => {
    if (edge.to.nodeId === node.id && edge.channel === 'data') return inputs.has(edge.to.portId)
    if (edge.from.nodeId === node.id) return outputs.has(`${edge.channel}\u0000${edge.from.portId}`)
    return true
  })
}

function clone<T>(value: T): T {
  return structuredClone(value)
}
