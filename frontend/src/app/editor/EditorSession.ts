import type {
  Edge,
  Graph,
  InputBinding,
  Node,
  BlobRef,
  YottaWorkflowSource31,
} from '../../../../contracts/workflow/3.1/workflow-source'
import type {
  NodeProjection,
  TypeExpression,
  YottaNodeAuthoringProjection31,
} from '../../../../contracts/node/3.1/authoring-projection'
import type {
  CompileView,
  RunView,
  SourceView,
  WorkflowJSONValue,
  WorkflowPatchCommand,
  WorkflowTransport,
} from '@/app/transport/workflow'

export type EditorPhase = 'empty' | 'loading' | 'ready' | 'saving' | 'running' | 'failed'

export type EditorCommand =
  | { kind: 'rename-workflow'; name: string }
  | { kind: 'add-state-variable'; name: string; type: TypeExpression; defaultValue: unknown }
  | { kind: 'remove-state-variable'; name: string }
  | { kind: 'add-node'; nodeTypeId: string; position: { x: number; y: number }; nodeId?: string }
  | { kind: 'remove-node'; nodeId: string }
  | { kind: 'move-node'; nodeId: string; position: { x: number; y: number } }
  | { kind: 'set-node-label'; nodeId: string; label: string }
  | { kind: 'set-config'; nodeId: string; fieldId: string; value: unknown }
  | { kind: 'clear-config'; nodeId: string; fieldId: string }
  | { kind: 'bind-value'; nodeId: string; portId: string; value: unknown }
  | { kind: 'bind-blob'; nodeId: string; portId: string; blob: BlobRef }
  | { kind: 'bind-default'; nodeId: string; portId: string }
  | { kind: 'clear-binding'; nodeId: string; portId: string }
  | { kind: 'connect'; edge: Edge }
  | { kind: 'disconnect'; edge: Edge }

interface PendingCommand {
  graphId: string
  command: EditorCommand
}

export class EditorSession {
  phase: EditorPhase = 'empty'
  source: YottaWorkflowSource31 | null = null
  authoring: YottaNodeAuthoringProjection31 | null = null
  baseRevision = -1
  sourceHash = ''
  compiledHash = ''
  lastRunHash = ''
  diagnostics: CompileView['diagnostics'] = []
  activeRun: RunView | null = null
  graphPath: string[] = []
  dirty = false
  saveConflict = ''
  failure = ''
  debugging = false

  private readonly history: YottaWorkflowSource31[] = []
  private readonly future: YottaWorkflowSource31[] = []
  private readonly pendingCommands: PendingCommand[] = []
  private readonly revertedCommands: PendingCommand[] = []
  private readonly projections = new Map<string, NodeProjection>()

  constructor(
    private readonly transport: WorkflowTransport,
    private readonly idFactory: () => string = defaultNodeId,
  ) {}

  get workflowId(): string {
    return this.source?.workflow.id ?? ''
  }

  get canUndo(): boolean {
    return this.history.length !== 0
  }

  get canRedo(): boolean {
    return this.future.length !== 0
  }

  get currentGraph(): Graph | null {
    if (!this.source) return null
    const graphId = this.graphPath.at(-1) ?? this.source.entryGraph
    return this.source.graphs.find((graph) => graph.id === graphId) ?? null
  }

  nodeProjection(nodeTypeId: string): NodeProjection | undefined {
    return this.projections.get(nodeTypeId)
  }

  async load(workflowId: string): Promise<void> {
    this.phase = 'loading'
    this.failure = ''
    try {
      const [view, authoringJson] = await Promise.all([
        this.transport.getSource(workflowId),
        this.transport.getAuthoringProjection(),
      ])
      this.loadAuthoring(authoringJson)
      this.acceptSource(view)
      this.phase = 'ready'
    } catch (error) {
      this.fail(error)
      throw error
    }
  }

  apply(command: EditorCommand): void {
    const source = this.requireSource()
    const next = clone(source)
    const graph = graphAt(next, this.graphPath)
    const resolved = resolveEditorCommand(command, graph, this.idFactory)
    applyCommand(next, graph, resolved, this.projections)
    next.revision = this.baseRevision + 1
    this.history.push(clone(source))
    if (this.history.length > 100) this.history.shift()
    this.future.length = 0
    this.pendingCommands.push({ graphId: graph.id, command: clone(resolved) })
    this.revertedCommands.length = 0
    this.source = next
    this.dirty = true
    this.saveConflict = ''
    this.resetCompileFacts()
  }

  undo(): void {
    if (!this.source || this.history.length === 0) return
    this.future.push(clone(this.source))
    this.source = this.history.pop() ?? this.source
    const reverted = this.pendingCommands.pop()
    if (reverted) this.revertedCommands.push(reverted)
    this.source.revision = this.baseRevision + 1
    this.dirty = true
    this.resetCompileFacts()
  }

  redo(): void {
    if (!this.source || this.future.length === 0) return
    this.history.push(clone(this.source))
    this.source = this.future.pop() ?? this.source
    const restored = this.revertedCommands.pop()
    if (restored) this.pendingCommands.push(restored)
    this.source.revision = this.baseRevision + 1
    this.dirty = true
    this.resetCompileFacts()
  }

  enterGraph(graphId: string): void {
    const source = this.requireSource()
    if (!source.graphs.some((graph) => graph.id === graphId)) {
      throw new Error(`graph ${graphId} does not exist`)
    }
    this.graphPath.push(graphId)
  }

  leaveGraph(): void {
    this.graphPath.pop()
  }

  async validate(): Promise<CompileView> {
    if (this.dirty) await this.save()
    const result = await this.transport.compileSource(this.workflowId)
    this.sourceHash = result.sourceHash ?? ''
    this.compiledHash = result.programHash ?? ''
    this.diagnostics = [...result.diagnostics]
    return result
  }

  async save(): Promise<SourceView> {
    if (!this.dirty) {
      const source = this.requireSource()
      return {
        workflowId: source.workflow.id,
        name: source.workflow.name,
        revision: this.baseRevision,
        sourceHash: this.sourceHash,
        sourceJson: this.serialize(),
      } as SourceView
    }
    this.phase = 'saving'
    this.saveConflict = ''
    try {
      const commands = toWorkflowPatch(this.pendingCommands)
      const patched = await this.transport.applyPatch(this.workflowId, this.baseRevision, commands)
      this.acceptSource(patched.source)
      this.phase = 'ready'
      return patched.source
    } catch (error) {
      this.saveConflict = errorText(error)
      this.phase = 'ready'
      throw error
    }
  }

  async run(): Promise<RunView | null> {
    return this.start(false)
  }

  // Debug is the same admitted Program interpreter with timeline inspection;
  // it never constructs a second stepping runtime.
  async debug(): Promise<RunView | null> {
    return this.start(true)
  }

  async refreshRun(): Promise<RunView | null> {
    if (!this.activeRun) return null
    this.activeRun = await this.transport.getRunTimeline(this.activeRun.runId)
    if (terminalStatus(this.activeRun.status)) this.phase = 'ready'
    return this.activeRun
  }

  async cancelRun(): Promise<RunView | null> {
    if (!this.activeRun) return null
    this.activeRun = await this.transport.cancelRun(this.activeRun.runId)
    this.phase = 'ready'
    return this.activeRun
  }

  serialize(): string {
    return JSON.stringify(this.requireSource())
  }

  private async start(debugging: boolean): Promise<RunView | null> {
    this.failure = ''
    try {
      const compile = await this.validate()
      if (compile.diagnostics.some((diagnostic) => diagnostic.severity === 'error')) return null
      if (!compile.programHash) throw new Error('compiler produced no Program hash')
      this.phase = 'running'
      const started = await this.transport.startRun(this.workflowId)
      this.diagnostics = [...started.diagnostics]
      this.sourceHash = started.sourceHash ?? this.sourceHash
      this.compiledHash = started.programHash ?? this.compiledHash
      this.lastRunHash = started.programHash ?? ''
      this.activeRun = started.run ?? null
      this.debugging = debugging
      if (!this.activeRun) this.phase = 'ready'
      return this.activeRun
    } catch (error) {
      this.fail(error)
      throw error
    }
  }

  private loadAuthoring(raw: string): void {
    const parsed: unknown = JSON.parse(raw)
    if (!isAuthoringProjection(parsed)) throw new Error('unsupported node authoring projection')
    this.authoring = parsed
    this.projections.clear()
    for (const projection of parsed.body.nodes) {
      this.projections.set(projection.nodeRef.nodeTypeId, projection)
    }
  }

  private acceptSource(view: SourceView): void {
    if (!view.sourceJson) throw new Error('Workflow Source response omitted sourceJson')
    const parsed: unknown = JSON.parse(view.sourceJson)
    if (!isWorkflowSource(parsed) || parsed.workflow.id !== view.workflowId) {
      throw new Error('Workflow Source response has invalid identity')
    }
    this.source = parsed
    this.baseRevision = view.revision
    this.sourceHash = view.sourceHash
    this.graphPath = [parsed.entryGraph]
    this.history.length = 0
    this.future.length = 0
    this.pendingCommands.length = 0
    this.revertedCommands.length = 0
    this.dirty = false
    this.saveConflict = ''
    this.resetCompileFacts()
  }

  private resetCompileFacts(): void {
    this.compiledHash = ''
    this.diagnostics = []
  }

  private requireSource(): YottaWorkflowSource31 {
    if (!this.source) throw new Error('EditorSession has no Workflow Source')
    return this.source
  }

  private fail(error: unknown): void {
    this.phase = 'failed'
    this.failure = errorText(error)
  }
}

function resolveEditorCommand(
  command: EditorCommand,
  graph: Graph,
  idFactory: () => string,
): EditorCommand {
  if (command.kind !== 'add-node' || command.nodeId) return command
  return { ...command, nodeId: uniqueNodeId(graph, idFactory) }
}

function toWorkflowPatch(pending: PendingCommand[]): WorkflowPatchCommand[] {
  const generated = new Set(
    pending.flatMap(({ command }) =>
      command.kind === 'add-node' && command.nodeId ? [command.nodeId] : [],
    ),
  )
  const nodeRef = (nodeId: string): string => (generated.has(nodeId) ? `$${nodeId}` : nodeId)
  return pending.map(({ graphId, command }): WorkflowPatchCommand => {
    switch (command.kind) {
      case 'rename-workflow':
        return { kind: command.kind, renameWorkflow: { name: command.name } }
      case 'add-state-variable':
        return {
          kind: command.kind,
          addStateVariable: {
            name: command.name,
            type: clone(command.type),
            default: jsonValue(command.defaultValue),
          },
        }
      case 'remove-state-variable':
        return { kind: command.kind, removeStateVariable: { name: command.name } }
      case 'add-node':
        if (!command.nodeId) throw new Error('pending add-node command omitted node ID')
        return {
          kind: command.kind,
          addNode: {
            graphId,
            nodeTypeId: command.nodeTypeId,
            handle: command.nodeId,
            position: clone(command.position),
          },
        }
      case 'remove-node':
        return { kind: command.kind, removeNode: { graphId, nodeId: nodeRef(command.nodeId) } }
      case 'move-node':
        return {
          kind: command.kind,
          moveNode: { graphId, nodeId: nodeRef(command.nodeId), position: clone(command.position) },
        }
      case 'set-node-label':
        return {
          kind: command.kind,
          setNodeLabel: { graphId, nodeId: nodeRef(command.nodeId), label: command.label },
        }
      case 'set-config':
        return {
          kind: command.kind,
          setConfig: {
            graphId,
            nodeId: nodeRef(command.nodeId),
            fieldId: command.fieldId,
            value: jsonValue(command.value),
          },
        }
      case 'clear-config':
        return {
          kind: command.kind,
          clearConfig: { graphId, nodeId: nodeRef(command.nodeId), fieldId: command.fieldId },
        }
      case 'bind-value':
        return {
          kind: command.kind,
          bindValue: {
            graphId,
            nodeId: nodeRef(command.nodeId),
            portId: command.portId,
            value: jsonValue(command.value),
          },
        }
      case 'bind-blob':
        return {
          kind: command.kind,
          bindBlob: {
            graphId,
            nodeId: nodeRef(command.nodeId),
            portId: command.portId,
            blob: clone(command.blob),
          },
        }
      case 'bind-default':
        return {
          kind: command.kind,
          bindDefault: {
            graphId,
            nodeId: nodeRef(command.nodeId),
            portId: command.portId,
          },
        }
      case 'clear-binding':
        return {
          kind: command.kind,
          clearBinding: {
            graphId,
            nodeId: nodeRef(command.nodeId),
            portId: command.portId,
          },
        }
      case 'connect':
        return {
          kind: command.kind,
          connect: {
            graphId,
            edge: {
              channel: command.edge.channel,
              from: { nodeId: nodeRef(command.edge.from.nodeId), portId: command.edge.from.portId },
              to: { nodeId: nodeRef(command.edge.to.nodeId), portId: command.edge.to.portId },
            },
          },
        }
      case 'disconnect':
        return {
          kind: command.kind,
          disconnect: {
            graphId,
            edge: {
              channel: command.edge.channel,
              from: { nodeId: nodeRef(command.edge.from.nodeId), portId: command.edge.from.portId },
              to: { nodeId: nodeRef(command.edge.to.nodeId), portId: command.edge.to.portId },
            },
          },
        }
    }
  })
}

function applyCommand(
  source: YottaWorkflowSource31,
  graph: Graph,
  command: EditorCommand,
  projections: Map<string, NodeProjection>,
): void {
  switch (command.kind) {
    case 'rename-workflow': {
      const name = command.name.trim()
      if (!name) throw new Error('workflow name is required')
      source.workflow.name = name
      return
    }
    case 'add-state-variable': {
      const name = command.name.trim()
      if (!/^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(name) || name.length > 128)
        throw new Error('state variable name is invalid')
      if (source.variables.some((variable) => variable.name === name))
        throw new Error(`duplicate state variable ${name}`)
      if (source.variables.length >= 4096) throw new Error('state variable budget exceeded')
      source.variables.push({
        name,
        type: clone(command.type),
        default: clone(command.defaultValue),
      })
      return
    }
    case 'remove-state-variable': {
      if (!source.variables.some((variable) => variable.name === command.name))
        throw new Error(`state variable ${command.name} does not exist`)
      const referenced = source.graphs.some((candidate) =>
        candidate.nodes.some(
          (node) =>
            node.nodeRef.nodeTypeId.includes('/nodes/state/') &&
            node.config.variable === command.name,
        ),
      )
      if (referenced) throw new Error(`state variable ${command.name} is still referenced`)
      source.variables = source.variables.filter((variable) => variable.name !== command.name)
      return
    }
    case 'add-node': {
      const projection = projections.get(command.nodeTypeId)
      if (!projection) throw new Error(`unknown Node Contract ${command.nodeTypeId}`)
      if (!command.nodeId) throw new Error('resolved add-node command omitted node ID')
      const id = command.nodeId
      if (graph.nodes.some((node) => node.id === id)) throw new Error(`duplicate node ${id}`)
      graph.nodes.push({
        id,
        nodeRef: clone(projection.nodeRef),
        position: clone(command.position),
        config: {},
        bindings: {},
      })
      return
    }
    case 'remove-node':
      requireNode(graph, command.nodeId)
      graph.nodes = graph.nodes.filter((node) => node.id !== command.nodeId)
      graph.edges = graph.edges.filter(
        (edge) => edge.from.nodeId !== command.nodeId && edge.to.nodeId !== command.nodeId,
      )
      return
    case 'move-node':
      requireNode(graph, command.nodeId).position = clone(command.position)
      return
    case 'set-node-label': {
      const node = requireNode(graph, command.nodeId)
      const label = command.label.trim()
      if (label) node.label = label
      else delete node.label
      return
    }
    case 'set-config':
      requireNode(graph, command.nodeId).config[command.fieldId] = clone(command.value)
      return
    case 'clear-config':
      delete requireNode(graph, command.nodeId).config[command.fieldId]
      return
    case 'bind-value': {
      const node = requireNode(graph, command.nodeId)
      requireDataInput(node, command.portId, projections)
      node.bindings[command.portId] = { kind: 'value', value: clone(command.value) }
      graph.edges = graph.edges.filter(
        (edge) =>
          !(
            edge.channel === 'data' &&
            edge.to.nodeId === command.nodeId &&
            edge.to.portId === command.portId
          ),
      )
      return
    }
    case 'bind-blob': {
      const node = requireNode(graph, command.nodeId)
      const port = requireDataInput(node, command.portId, projections)
      if (!port.type.representations.some((representation) => representation.kind === 'blob-ref'))
        throw new Error(`port ${command.portId} does not accept BlobRef`)
      if (!validBlob(command.blob)) throw new Error('BlobRef is invalid')
      node.bindings[command.portId] = { kind: 'blob', blob: clone(command.blob) }
      graph.edges = graph.edges.filter(
        (edge) =>
          !(
            edge.channel === 'data' &&
            edge.to.nodeId === command.nodeId &&
            edge.to.portId === command.portId
          ),
      )
      return
    }
    case 'bind-default': {
      const node = requireNode(graph, command.nodeId)
      const port = requireDataInput(node, command.portId, projections)
      if (!port.hasDefault) throw new Error(`port ${command.portId} has no declared default`)
      node.bindings[command.portId] = { kind: 'default' }
      return
    }
    case 'clear-binding':
      delete requireNode(graph, command.nodeId).bindings[command.portId]
      return
    case 'connect':
      validateEdge(graph, command.edge, projections)
      if (command.edge.channel === 'data') {
        const target = requireNode(graph, command.edge.to.nodeId)
        delete target.bindings[command.edge.to.portId]
        graph.edges = graph.edges.filter(
          (edge) =>
            !(
              edge.channel === 'data' &&
              edge.to.nodeId === command.edge.to.nodeId &&
              edge.to.portId === command.edge.to.portId
            ),
        )
      }
      if (!graph.edges.some((edge) => sameEdge(edge, command.edge))) {
        graph.edges.push(clone(command.edge))
      }
      return
    case 'disconnect':
      graph.edges = graph.edges.filter((edge) => !sameEdge(edge, command.edge))
  }
}

function validateEdge(graph: Graph, edge: Edge, projections: Map<string, NodeProjection>): void {
  const fromNode = requireNode(graph, edge.from.nodeId)
  const toNode = requireNode(graph, edge.to.nodeId)
  const from = requireProjection(fromNode, projections)
  const to = requireProjection(toNode, projections)
  if (edge.channel === 'data') {
    const output = from.dataOutputs.find((port) => port.id === edge.from.portId)
    const input = to.dataInputs.find((port) => port.id === edge.to.portId)
    if (!output || !input || !assignable(output.type.expression, input.type.expression)) {
      throw new Error('data edge is not assignable')
    }
    if (output.carrier !== input.carrier) throw new Error('data edge carrier class differs')
    return
  }
  const output = from.signals.find(
    (signal) =>
      signal.id === edge.from.portId &&
      signal.direction === 'output' &&
      signal.channel === edge.channel,
  )
  const input = to.signals.find(
    (signal) => signal.id === edge.to.portId && signal.direction === 'input',
  )
  if (!output || !input || !instructionAcceptsSignal(to, edge.channel, edge.to.portId)) {
    throw new Error(`${edge.channel} edge has invalid signal ports`)
  }
}

function instructionAcceptsSignal(
  projection: NodeProjection,
  channel: 'exec' | 'error',
  inputPort: string,
): boolean {
  const instruction = projection.instruction
  switch (instruction.kind) {
    case 'invoke':
      return channel === 'exec' || channel === 'error'
    case 'run-root':
      return false
    case 'counted-loop': {
      const value = instruction.countedLoop
      return Boolean(
        value &&
        channel === 'exec' &&
        [value.entryInput, value.breakInput, value.continueInput].includes(inputPort),
      )
    }
    case 'for-each': {
      const value = instruction.forEach
      return Boolean(
        value &&
        channel === 'exec' &&
        [value.entryInput, value.breakInput, value.continueInput].includes(inputPort),
      )
    }
    case 'retry': {
      const value = instruction.retry
      return Boolean(
        value &&
        ((channel === 'exec' && inputPort === value.entryInput) ||
          (channel === 'error' && inputPort === value.retryInput)),
      )
    }
  }
}

export function assignable(output: TypeExpression, input: TypeExpression): boolean {
  if (output.kind === 'variable' || input.kind === 'variable') return false
  if (output.kind === 'union') return output.members.every((member) => assignable(member, input))
  if (input.kind === 'union') return input.members.some((member) => assignable(output, member))
  if (output.kind !== input.kind) return false
  if (output.kind === 'ref' && input.kind === 'ref') {
    return (
      output.ref.typeId === input.ref.typeId &&
      output.ref.semanticDigest === input.ref.semanticDigest
    )
  }
  if (output.kind === 'list' && input.kind === 'list') {
    return assignable(output.element, input.element)
  }
  return false
}

function requireNode(graph: Graph, nodeId: string): Node {
  const node = graph.nodes.find((candidate) => candidate.id === nodeId)
  if (!node) throw new Error(`node ${nodeId} does not exist`)
  return node
}

function requireProjection(node: Node, projections: Map<string, NodeProjection>): NodeProjection {
  const projection = projections.get(node.nodeRef.nodeTypeId)
  if (!projection || projection.nodeRef.semanticDigest !== node.nodeRef.semanticDigest) {
    throw new Error(`node ${node.id} has no exact authoring projection`)
  }
  return projection
}

function requireDataInput(
  node: Node,
  portId: string,
  projections: Map<string, NodeProjection>,
): NodeProjection['dataInputs'][number] {
  const port = requireProjection(node, projections).dataInputs.find(
    (candidate) => candidate.id === portId,
  )
  if (!port) throw new Error(`node ${node.id} has no data input ${portId}`)
  return port
}

function graphAt(source: YottaWorkflowSource31, path: string[]): Graph {
  const graphId = path.at(-1) ?? source.entryGraph
  const graph = source.graphs.find((candidate) => candidate.id === graphId)
  if (!graph) throw new Error(`graph ${graphId} does not exist`)
  return graph
}

function uniqueNodeId(graph: Graph, idFactory: () => string): string {
  for (let attempt = 0; attempt < 32; attempt += 1) {
    const candidate = idFactory()
    if (
      /^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(candidate) &&
      !graph.nodes.some((node) => node.id === candidate)
    ) {
      return candidate
    }
  }
  throw new Error('could not allocate a unique node ID')
}

function defaultNodeId(): string {
  return `node_${crypto.randomUUID().replaceAll('-', '')}`
}

function sameEdge(left: Edge, right: Edge): boolean {
  return (
    left.channel === right.channel &&
    left.from.nodeId === right.from.nodeId &&
    left.from.portId === right.from.portId &&
    left.to.nodeId === right.to.nodeId &&
    left.to.portId === right.to.portId
  )
}

function terminalStatus(status: string): boolean {
  return ['SUCCEEDED', 'FAILED', 'CANCELLED', 'INTERRUPTED'].includes(status.toUpperCase())
}

function isWorkflowSource(value: unknown): value is YottaWorkflowSource31 {
  if (typeof value !== 'object' || value === null) return false
  const source = value as Record<string, unknown>
  return (
    source.format === 'yotta.workflow' &&
    source.version === '3.1' &&
    typeof source.revision === 'number' &&
    typeof source.entryGraph === 'string' &&
    Array.isArray(source.graphs) &&
    typeof source.workflow === 'object' &&
    source.workflow !== null
  )
}

function isAuthoringProjection(value: unknown): value is YottaNodeAuthoringProjection31 {
  if (typeof value !== 'object' || value === null) return false
  const document = value as Record<string, unknown>
  if (
    document.format !== 'yotta.node-authoring-projection' ||
    document.version !== '3.1' ||
    typeof document.projectionDigest !== 'string' ||
    typeof document.body !== 'object' ||
    document.body === null
  ) {
    return false
  }
  const body = document.body as Record<string, unknown>
  return Array.isArray(body.nodes) && Array.isArray(body.types)
}

function clone<T>(value: T): T {
  return structuredClone(value)
}

function jsonValue(value: unknown): WorkflowJSONValue {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return value
  if (typeof value === 'number') {
    if (!Number.isFinite(value)) throw new Error('authoring value must be a finite JSON number')
    return value
  }
  if (Array.isArray(value)) return value.map(jsonValue)
  if (typeof value === 'object') {
    const result: Record<string, WorkflowJSONValue> = {}
    for (const [key, member] of Object.entries(value)) result[key] = jsonValue(member)
    return result
  }
  throw new Error('authoring value must be JSON data')
}

function validBlob(blob: BlobRef): boolean {
  return (
    /^[a-z0-9][a-z0-9!#$&^_.+-]+\/[a-z0-9][a-z0-9!#$&^_.+-]+$/.test(blob.mediaType) &&
    /^sha256:[0-9a-f]{64}$/.test(blob.digest) &&
    Number.isSafeInteger(blob.size) &&
    blob.size >= 0
  )
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export type { Edge, Graph, InputBinding, Node, NodeProjection, YottaWorkflowSource31 }
