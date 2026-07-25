import type {
  Edge,
  Graph,
  GraphCall,
  Annotation,
  InputBinding,
  Node,
  BlobRef,
  ResourceBinding,
  WorkflowResource,
  YottaWorkflowSource,
} from '../../../../contracts/workflow/current/workflow-source'
import type {
  NodeProjection,
  PortProjection,
  TypeExpression,
  TypeProjection,
  TypeUse,
  YottaNodeAuthoringProjection,
} from '../../../../contracts/node/current/authoring-projection'
import type {
  CompileView,
  RunView,
  SourceView,
  WorkflowJSONValue,
  WorkflowPatchCommand,
  WorkflowTransport,
  DebugBreakpoint,
  DebugSnapshot,
} from '@/app/transport/workflow'
import {
  assignable,
  projectedConnectionCompatibility,
  type ConversionCandidatePlan,
  type ConnectionCompatibility,
} from './connectionCompatibility'
import type { ParsedHandle } from './graphHandles'
import type { GraphBoundaryBinding, GraphBoundaryKey } from './workflowGraphBoundary'
import {
  addGraphInterfaceCandidate,
  graphInterfaceReferences,
  inferGraphInterface,
  moveGraphInterfaceItem,
  projectGraphInterfaceCandidates,
  removeGraphInterfaceItem,
  renameGraphInterfaceItem,
  type GraphInterfaceCandidate,
  type GraphInterfaceDraft,
  type GraphInterfaceElement,
  type GraphInterfaceInferencePreview,
  type GraphInterfaceItemKind,
  type GraphInterfaceReference,
} from './subgraphInterface'
import {
  duplicateGraphCall,
  duplicateGraphDefinition,
  expandGraphCall,
  graphCallSites,
  type ExpandedGraphCall,
  type GraphCallSite,
} from './subgraphLifecycle'

export { assignable } from './connectionCompatibility'

const PLAY_INPUT_CLIP_NODE_TYPE_ID = 'https://schemas.yotta.dev/nodes/automation/play-input-clip'
const PLAY_INPUT_CLIP_STABLE_DIGEST =
  'sha256:5c353fb0725ca6a841a7ef5e9adcca12bb10e2d6362fed4d7d38449a58608e02'
const PLAY_INPUT_CLIP_RETRACTED_SCALE_DIGEST =
  'sha256:ff7ea9d0b2ca91cb2062cff30dd5ca8575555ec5363b4c76e746925ee6ae027b'

export type EditorPhase = 'empty' | 'loading' | 'ready' | 'saving' | 'running' | 'failed'

export interface LinearWorkflowDraftNode {
  nodeTypeID: string
  config: Record<string, unknown>
  values: Record<string, unknown>
  blobs: Record<string, BlobRef>
  resources?: Record<string, ResourceBinding>
  execInput: string
  execOutput: string
}

export interface StateReferenceLocation {
  graphId: string
  nodeId: string
  mode: 'read' | 'write'
}

export type StateReferenceMode = 'read' | 'write' | 'last-change' | 'increment'

export interface StateTypeChangeIssue {
  graphId: string
  edge: Edge
  disposition: 'conversion' | 'incompatible'
  conversions: ConversionCandidatePlan[]
}

export interface StateTypeChangeImpact {
  references: StateReferenceLocation[]
  issues: StateTypeChangeIssue[]
}

export type EditorCommand =
  | { kind: 'rename-workflow'; name: string }
  | { kind: 'set-target-default'; target: string; slot: string }
  | { kind: 'clear-target-default'; target: string }
  | { kind: 'add-state-variable'; name: string; type: TypeExpression; defaultValue: unknown }
  | { kind: 'update-state-variable'; name: string; type: TypeExpression; defaultValue: unknown }
  | { kind: 'remove-state-variable'; name: string }
  | { kind: 'add-node'; nodeTypeId: string; position: { x: number; y: number }; nodeId?: string }
  | { kind: 'upgrade-node-contract'; nodeId: string }
  | { kind: 'remove-node'; nodeId: string }
  | { kind: 'move-node'; nodeId: string; position: { x: number; y: number } }
  | {
      kind: 'move-nodes'
      positions: Array<{ nodeId: string; position: { x: number; y: number } }>
    }
  | { kind: 'set-node-label'; nodeId: string; label: string }
  | { kind: 'set-node-disabled'; nodeId: string; disabled: boolean }
  | { kind: 'set-config'; nodeId: string; fieldId: string; value: unknown }
  | { kind: 'clear-config'; nodeId: string; fieldId: string }
  | { kind: 'bind-value'; nodeId: string; portId: string; value: unknown }
  | { kind: 'bind-blob'; nodeId: string; portId: string; blob: BlobRef }
  | { kind: 'bind-resource'; nodeId: string; portId: string; resource: ResourceBinding }
  | { kind: 'add-resource'; resource: WorkflowResource }
  | { kind: 'replace-resource'; resourceId: string; resource: WorkflowResource }
  | {
      kind: 'update-resource-metadata'
      resourceId: string
      name: string
      description: string
      category: string
      tags: string[]
    }
  | { kind: 'remove-resource'; resourceId: string }
  | { kind: 'bind-default'; nodeId: string; portId: string }
  | { kind: 'clear-binding'; nodeId: string; portId: string }
  | { kind: 'connect'; edge: Edge }
  | { kind: 'disconnect'; edge: Edge }
  | { kind: 'add-graph'; graph: Graph }
  | { kind: 'rename-graph'; graphId: string; name: string }
  | { kind: 'remove-graph'; graphId: string }
  | { kind: 'remove-graph-cascade'; graphId: string; calls: GraphCallSite[] }
  | {
      kind: 'update-graph-interface'
      inputs: Graph['inputs']
      outputs: Graph['outputs']
      entries: NonNullable<Graph['entries']>
      exits: NonNullable<Graph['exits']>
    }
  | { kind: 'add-graph-call'; call: GraphCall }
  | { kind: 'update-graph-call'; call: GraphCall }
  | { kind: 'remove-graph-call'; callId: string }
  | { kind: 'fork-graph-call'; graph: Graph; call: GraphCall }
  | ({ kind: 'expand-graph-call' } & ExpandedGraphCall)
  | { kind: 'add-annotation'; annotation: Annotation }
  | { kind: 'update-annotation'; annotation: Annotation }
  | { kind: 'remove-annotation'; annotationId: string }
  | { kind: 'set-edge-reroutes'; edge: Edge; reroutes: Array<{ x: number; y: number }> }
  | {
      kind: 'collapse-selection'
      subgraphId: string
      callId: string
      name: string
      nodeIds: string[]
      position: { x: number; y: number }
    }
  | { kind: 'remove-nodes'; nodeIds: string[] }
  | {
      kind: 'insert-node-selection'
      nodes: Node[]
      calls: GraphCall[]
      annotations: Annotation[]
      edges: Edge[]
    }
  | {
      kind: 'insert-connected-node'
      nodeTypeId: string
      nodeId: string
      position: { x: number; y: number }
      edge: Edge
    }
  | {
      kind: 'promote-output-to-state'
      name: string
      type: TypeExpression
      defaultValue: unknown
      nodeTypeId: string
      nodeId: string
      stateConfigKey: string
      position: { x: number; y: number }
      edge: Edge
    }
  | { kind: 'batch'; commands: EditorCommand[] }

interface PendingCommand {
  graphId: string
  command: EditorCommand
}

export class EditorSession {
  phase: EditorPhase = 'empty'
  source: YottaWorkflowSource | null = null
  authoring: YottaNodeAuthoringProjection | null = null
  baseRevision = -1
  sourceHash = ''
  compiledHash = ''
  lastRunHash = ''
  diagnostics: CompileView['diagnostics'] = []
  activeRun: RunView | null = null
  debugSnapshot: DebugSnapshot | null = null
  graphPath: string[] = []
  dirty = false
  saveConflict = ''
  failure = ''

  private readonly history: YottaWorkflowSource[] = []
  private readonly future: YottaWorkflowSource[] = []
  private readonly pendingCommands: PendingCommand[] = []
  private readonly revertedCommands: PendingCommand[] = []
  private readonly projections = new Map<string, NodeProjection>()
  private readonly typeProjections = new Map<string, TypeProjection>()
  private readonly pendingDebugSnapshots = new Map<string, DebugSnapshot>()
  private debugStartPending = false

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

  createSubgraph(name = 'Subgraph'): string {
    const source = this.requireSource()
    const graphId = uniqueGraphId(source, this.idFactory)
    this.apply({
      kind: 'add-graph',
      graph: {
        id: graphId,
        name,
        kind: 'subgraph',
        nodes: [],
        calls: [],
        edges: [],
        inputs: [],
        outputs: [],
        entries: [],
        exits: [],
        annotations: [],
      },
    })
    this.enterGraph(graphId)
    return graphId
  }

  renameGraph(graphId: string, name: string): void {
    this.apply({ kind: 'rename-graph', graphId, name })
  }

  setTargetDefault(target: string, slot: string): void {
    if (slot) this.apply({ kind: 'set-target-default', target, slot })
    else this.apply({ kind: 'clear-target-default', target })
  }

  removeGraph(graphId: string): void {
    this.apply({ kind: 'remove-graph', graphId })
    this.graphPath = [this.requireSource().entryGraph]
  }

  removeGraphCascade(graphId: string): void {
    const source = this.requireSource()
    if (graphId === source.entryGraph) throw new Error('entry graph cannot be removed')
    if (!source.graphs.some((graph) => graph.id === graphId))
      throw new Error(`graph ${graphId} does not exist`)
    this.apply({ kind: 'remove-graph-cascade', graphId, calls: graphCallSites(source, graphId) })
    this.graphPath = [this.requireSource().entryGraph]
  }

  duplicateGraphDefinition(graphId: string): string {
    const source = this.requireSource()
    const original = source.graphs.find((graph) => graph.id === graphId)
    const newGraphId = uniqueGraphId(source, this.idFactory)
    const graph = duplicateGraphDefinition(
      source,
      graphId,
      newGraphId,
      `${original?.name?.trim() || graphId} Copy`,
    )
    this.apply({ kind: 'add-graph', graph })
    return newGraphId
  }

  duplicateCurrentGraphCall(callId: string, offset = { x: 32, y: 32 }): string {
    const graph = this.currentGraph
    if (!graph) throw new Error('workflow graph is unavailable')
    const newCallId = uniqueElementId(graph, this.idFactory)
    this.apply({
      kind: 'add-graph-call',
      call: duplicateGraphCall(graph, callId, newCallId, offset),
    })
    return newCallId
  }

  forkCurrentGraphCall(callId: string): string {
    const source = this.requireSource()
    const graph = this.currentGraph
    const call = graph?.calls?.find((candidate) => candidate.id === callId)
    if (!graph || !call) throw new Error(`graph call ${callId} does not exist`)
    const callee = source.graphs.find((candidate) => candidate.id === call.graphId)
    const newGraphId = uniqueGraphId(source, this.idFactory)
    const copy = duplicateGraphDefinition(
      source,
      call.graphId,
      newGraphId,
      `${callee?.name?.trim() || call.graphId} Copy`,
    )
    this.apply({
      kind: 'fork-graph-call',
      graph: copy,
      call: { ...clone(call), graphId: newGraphId },
    })
    return newGraphId
  }

  expandCurrentGraphCall(callId: string): string[] {
    const graph = this.currentGraph
    if (!graph) throw new Error('workflow graph is unavailable')
    const expansion = expandGraphCall(this.requireSource(), graph.id, callId, this.idFactory)
    this.apply({ kind: 'expand-graph-call', ...expansion })
    return [
      ...expansion.nodes.map((node) => node.id),
      ...expansion.calls.map((call) => call.id),
      ...expansion.annotations.map((annotation) => annotation.id),
    ]
  }

  addAnnotation(position: { x: number; y: number }): string {
    const graph = this.currentGraph
    if (!graph) throw new Error('workflow graph is unavailable')
    const id = uniqueElementId(graph, this.idFactory)
    this.apply({
      kind: 'add-annotation',
      annotation: { id, text: '', position, size: { width: 260, height: 140 } },
    })
    return id
  }

  collapseSelection(nodeIds: string[], name = 'Subgraph'): string {
    const graph = this.currentGraph
    if (!graph || nodeIds.length === 0) throw new Error('selection is empty')
    const subgraphId = uniqueGraphId(this.requireSource(), this.idFactory)
    const callId = uniqueElementId(graph, this.idFactory)
    const selected = [
      ...graph.nodes.filter((node) => nodeIds.includes(node.id)),
      ...graph.calls!.filter((call) => nodeIds.includes(call.id)),
    ]
    if (selected.length !== nodeIds.length)
      throw new Error('only executable nodes and graph calls can be collapsed')
    const position = {
      x: selected.reduce((sum, node) => sum + node.position.x, 0) / selected.length,
      y: selected.reduce((sum, node) => sum + node.position.y, 0) / selected.length,
    }
    this.apply({ kind: 'collapse-selection', subgraphId, callId, name, nodeIds, position })
    return callId
  }

  nodeProjection(nodeTypeId: string): NodeProjection | undefined {
    return this.projections.get(nodeTypeId)
  }

  nodeInstanceProjection(node: Node): NodeProjection | undefined {
    const base = this.projections.get(node.nodeRef.nodeTypeId)
    if (!base || base.nodeRef.semanticDigest !== node.nodeRef.semanticDigest) return undefined
    return resolveNodeInstanceProjection(
      this.source,
      node,
      base,
      this.projections,
      this.typeProjections,
    )
  }

  stateTypeChangeImpact(name: string, type: TypeExpression): StateTypeChangeImpact {
    const source = this.requireSource()
    const variable = source.variables.find((candidate) => candidate.name === name)
    if (!variable) throw new Error(`state variable ${name} does not exist`)
    const references = collectStateReferences(source, name)
    const candidate = clone(source)
    const candidateVariable = candidate.variables.find((item) => item.name === name)!
    candidateVariable.type = clone(type)
    const issues: StateTypeChangeIssue[] = []
    for (const graph of source.graphs) {
      const candidateGraph = candidate.graphs.find((item) => item.id === graph.id)!
      for (const edge of graph.edges) {
        if (edge.channel !== 'data') continue
        const before = connectionCompatibilityInGraph(
          source,
          graph,
          edge,
          this.projections,
          this.typeProjections,
          this.authoring?.body.nodes ?? [],
        )
        const after = connectionCompatibilityInGraph(
          candidate,
          candidateGraph,
          edge,
          this.projections,
          this.typeProjections,
          this.authoring?.body.nodes ?? [],
        )
        if (!before.valid || after.valid) continue
        issues.push({
          graphId: graph.id,
          edge: clone(edge),
          disposition: after.disposition === 'conversion' ? 'conversion' : 'incompatible',
          conversions: clone(after.conversions ?? []),
        })
      }
    }
    return { references, issues }
  }

  calleeGraph(call: GraphCall): Graph | undefined {
    return this.source?.graphs.find((graph) => graph.id === call.graphId)
  }

  insertGraphCall(graphId: string, position: { x: number; y: number }): string {
    const graph = this.currentGraph
    const callee = this.source?.graphs.find(
      (candidate) => candidate.id === graphId && candidate.kind === 'subgraph',
    )
    if (!graph || !callee) throw new Error(`subgraph ${graphId} does not exist`)
    const id = uniqueElementId(graph, this.idFactory)
    this.apply({
      kind: 'add-graph-call',
      call: {
        id,
        graphId,
        label: callee.name || callee.id,
        position,
        bindings: {},
      },
    })
    return id
  }

  graphInputProjection(graphId: string, portId: string): PortProjection | undefined {
    const source = this.source
    const graph = source?.graphs.find((candidate) => candidate.id === graphId)
    const port = graph?.inputs.find((candidate) => candidate.id === portId)
    if (!source || !graph || !port) return undefined
    const projection = resolveGraphInputProjection(
      source,
      graph,
      { nodeId: port.nodeId, portId: port.portId },
      this.projections,
      new Set(),
    )
    return projection ? { ...clone(projection), id: port.id } : undefined
  }

  currentGraphInterfaceReadiness(): {
    valid: boolean
    reason?: 'not-subgraph' | 'multiple-entry' | 'missing-entry-or-exit'
  } {
    const graph = this.currentGraph
    if (!graph || graph.kind !== 'subgraph') return { valid: false, reason: 'not-subgraph' }
    const candidates = this.currentGraphInterfaceCandidates()
    const entries = candidates.filter((candidate) => candidate.kind === 'entry').length
    const exits = candidates.filter((candidate) => candidate.kind === 'exit').length
    if (entries > 1) return { valid: false, reason: 'multiple-entry' }
    if (!entries || !exits) return { valid: false, reason: 'missing-entry-or-exit' }
    return { valid: true }
  }

  currentGraphInterfaceCandidates(): GraphInterfaceCandidate[] {
    const graph = this.currentGraph
    if (!graph || graph.kind !== 'subgraph') throw new Error('open a subgraph first')
    return projectGraphInterfaceCandidates(graph, this.graphInterfaceElements(graph))
  }

  addCurrentGraphInterfaceCandidate(candidateKey: string): void {
    const graph = this.requireCurrentSubgraph()
    const selected = this.currentGraphInterfaceCandidates().find(
      (candidate) => candidate.key === candidateKey,
    )
    if (!selected) throw new Error('subgraph interface candidate is no longer available')
    this.updateCurrentGraphInterface(addGraphInterfaceCandidate(graph, selected))
  }

  renameCurrentGraphInterfaceItem(kind: GraphInterfaceItemKind, id: string, name: string): void {
    const graph = this.requireCurrentSubgraph()
    this.updateCurrentGraphInterface(renameGraphInterfaceItem(graph, kind, id, name))
  }

  moveCurrentGraphInterfaceItem(kind: GraphInterfaceItemKind, id: string, direction: -1 | 1): void {
    const graph = this.requireCurrentSubgraph()
    this.updateCurrentGraphInterface(moveGraphInterfaceItem(graph, kind, id, direction))
  }

  removeCurrentGraphInterfaceItem(kind: 'entry' | GraphInterfaceItemKind, id = ''): void {
    const graph = this.requireCurrentSubgraph()
    this.updateCurrentGraphInterface(removeGraphInterfaceItem(graph, kind, id))
  }

  currentGraphInterfaceReferences(
    kind: GraphInterfaceItemKind,
    id: string,
  ): GraphInterfaceReference[] {
    const graph = this.requireCurrentSubgraph()
    return graphInterfaceReferences(this.requireSource(), graph.id, kind, id)
  }

  previewCurrentGraphInterfaceInference(): GraphInterfaceInferencePreview {
    const graph = this.requireCurrentSubgraph()
    return inferGraphInterface(graph, this.currentGraphInterfaceCandidates())
  }

  applyCurrentGraphInterfaceInference(preview?: GraphInterfaceInferencePreview): void {
    this.updateCurrentGraphInterface(
      preview?.draft ?? this.previewCurrentGraphInterfaceInference().draft,
    )
  }

  inferCurrentGraphInterface(): void {
    this.applyCurrentGraphInterfaceInference()
  }

  graphBoundaryCompatibility(binding: GraphBoundaryBinding): ConnectionCompatibility {
    const graph = this.currentGraph
    if (!graph || graph.kind !== 'subgraph')
      return { valid: false, issue: 'port', message: 'open a subgraph first' }
    if (binding.kind === 'entry') {
      return graphSignalEndpointValid(
        this.requireSource(),
        graph,
        binding.endpoint,
        false,
        'exec',
        this.projections,
      )
        ? { valid: true, disposition: 'direct' }
        : { valid: false, issue: 'port', message: 'subgraph entry requires an execution input' }
    }
    if (binding.kind === 'exit') {
      const exit = graph.exits?.find((candidate) => candidate.id === binding.boundaryId)
      return exit &&
        graphSignalEndpointValid(
          this.requireSource(),
          graph,
          binding.endpoint,
          true,
          exit.channel,
          this.projections,
        )
        ? { valid: true, disposition: 'direct' }
        : {
            valid: false,
            issue: 'port',
            message: 'subgraph exit requires a matching signal output',
          }
    }
    const port = (binding.kind === 'input' ? graph.inputs : graph.outputs).find(
      (candidate) => candidate.id === binding.boundaryId,
    )
    if (!port) return { valid: false, issue: 'port', message: 'subgraph interface port is missing' }
    const endpointType = graphEndpointType(
      this.requireSource(),
      graph,
      binding.endpoint,
      binding.kind === 'output',
      this.projections,
    )
    const compatible =
      endpointType &&
      (binding.kind === 'input'
        ? assignable(port.type, endpointType, this.typeProjections)
        : assignable(endpointType, port.type, this.typeProjections))
    return compatible
      ? { valid: true, disposition: 'direct' }
      : {
          valid: false,
          issue: 'type',
          message: 'subgraph interface and endpoint types are incompatible',
          disposition: 'incompatible',
        }
  }

  bindGraphBoundary(binding: GraphBoundaryBinding): void {
    const compatibility = this.graphBoundaryCompatibility(binding)
    if (!compatibility.valid) throw new Error(compatibility.message ?? 'invalid subgraph binding')
    const graph = this.currentGraph!
    const inputs = clone(graph.inputs)
    const outputs = clone(graph.outputs)
    const entries: NonNullable<Graph['entries']> = graph.entries ? clone(graph.entries) : []
    const exits = clone(graph.exits ?? [])
    if (binding.kind === 'entry') entries.splice(0, entries.length, clone(binding.endpoint))
    if (binding.kind === 'input') {
      const port = inputs.find((candidate) => candidate.id === binding.boundaryId)!
      Object.assign(port, clone(binding.endpoint))
    }
    if (binding.kind === 'output') {
      const port = outputs.find((candidate) => candidate.id === binding.boundaryId)!
      Object.assign(port, clone(binding.endpoint))
    }
    if (binding.kind === 'exit') {
      const exit = exits.find((candidate) => candidate.id === binding.boundaryId)!
      exit.endpoint = clone(binding.endpoint)
    }
    this.apply({ kind: 'update-graph-interface', inputs, outputs, entries, exits })
  }

  unbindGraphBoundary(key: GraphBoundaryKey): void {
    const graph = this.currentGraph
    if (!graph || graph.kind !== 'subgraph') throw new Error('open a subgraph first')
    this.apply({
      kind: 'update-graph-interface',
      inputs:
        key.kind === 'input'
          ? graph.inputs.filter((port) => port.id !== key.boundaryId)
          : clone(graph.inputs),
      outputs:
        key.kind === 'output'
          ? graph.outputs.filter((port) => port.id !== key.boundaryId)
          : clone(graph.outputs),
      entries: key.kind === 'entry' ? [] : clone(graph.entries ?? []),
      exits:
        key.kind === 'exit'
          ? (graph.exits ?? []).filter((exit) => exit.id !== key.boundaryId)
          : clone(graph.exits ?? []),
    })
  }

  connectionCompatibility(edge: Edge): ConnectionCompatibility {
    const graph = this.currentGraph
    if (!graph) return { valid: false, issue: 'port', message: 'workflow graph is unavailable' }
    return connectionCompatibilityInGraph(
      this.requireSource(),
      graph,
      edge,
      this.projections,
      this.typeProjections,
      this.authoring?.body.nodes ?? [],
    )
  }

  insertConversionBridge(
    edge: Edge,
    conversion: ConversionCandidatePlan,
    position: { x: number; y: number },
  ): string {
    if (edge.channel !== 'data') throw new Error('conversion bridge requires a data edge')
    const plan = this.connectionCompatibility(edge)
    const allowed = plan.conversions?.find(
      (candidate) =>
        candidate.nodeTypeId === conversion.nodeTypeId &&
        candidate.inputPort === conversion.inputPort &&
        candidate.outputPort === conversion.outputPort,
    )
    if (!allowed) throw new Error('conversion is not valid for this connection')
    const graph = this.currentGraph
    const projection = this.projections.get(allowed.nodeTypeId)
    if (!graph || !projection) throw new Error('conversion projection is unavailable')
    const nodeId = uniqueNodeId(graph, this.idFactory)
    this.apply({
      kind: 'insert-node-selection',
      nodes: [
        {
          id: nodeId,
          nodeRef: clone(projection.nodeRef),
          position: clone(position),
          config: {},
          bindings: {},
        },
      ],
      calls: [],
      annotations: [],
      edges: [
        {
          channel: 'data',
          from: clone(edge.from),
          to: { nodeId, portId: allowed.inputPort },
        },
        {
          channel: 'data',
          from: { nodeId, portId: allowed.outputPort },
          to: clone(edge.to),
        },
      ],
    })
    return nodeId
  }

  promoteOutputToState(
    nodeId: string,
    portId: string,
    name: string,
    position: { x: number; y: number },
  ): string {
    const graph = this.currentGraph
    const sourceNode = graph?.nodes.find((node) => node.id === nodeId)
    const sourceProjection = sourceNode ? this.nodeInstanceProjection(sourceNode) : undefined
    const output = sourceProjection?.dataOutputs.find((port) => port.id === portId)
    if (
      !graph ||
      !output ||
      output.carrier !== 'durable' ||
      output.type.expression.kind !== 'ref'
    ) {
      throw new Error('output cannot be promoted to durable state')
    }
    const type = this.typeProjections.get(output.type.expression.ref.typeId)
    if (!type || !type.traits.includes('durable') || !stateTypeHasDefault(type)) {
      throw new Error('output type cannot be initialized as durable state')
    }
    const stateWrite = [...this.projections.values()].find(
      (projection) =>
        projection.nodeRef.nodeTypeId.endsWith('/state/write') &&
        projection.stateAccesses.some((access) => access.mode === 'write'),
    )
    if (!stateWrite) throw new Error('state write node is unavailable')
    const inputPort = stateWrite.stateAccesses.find((access) => access.mode === 'write')
    const valuePort = stateWrite.dataInputs.find((port) => port.type.expression.kind === 'variable')
    if (!inputPort || !valuePort) throw new Error('state write contract is incomplete')
    const stateNodeId = uniqueNodeId(graph, this.idFactory)
    this.apply({
      kind: 'promote-output-to-state',
      name,
      type: clone(output.type.expression),
      defaultValue: defaultStateValue(type),
      nodeTypeId: stateWrite.nodeRef.nodeTypeId,
      nodeId: stateNodeId,
      stateConfigKey: inputPort.slotConfigKey,
      position: clone(position),
      edge: {
        channel: 'data',
        from: { nodeId, portId },
        to: { nodeId: stateNodeId, portId: valuePort.id },
      },
    })
    return stateNodeId
  }

  insertConnectedNode(
    anchorNodeId: string,
    anchor: ParsedHandle,
    nodeTypeId: string,
    candidate: ParsedHandle,
    position: { x: number; y: number },
  ): string {
    if (anchor.direction === candidate.direction || anchor.channel !== candidate.channel) {
      throw new Error('connected node ports are incompatible')
    }
    const graph = this.currentGraph
    if (!graph) throw new Error('workflow graph is unavailable')
    const nodeId = uniqueNodeId(graph, this.idFactory)
    const edge: Edge =
      anchor.direction === 'output'
        ? {
            channel: anchor.channel,
            from: { nodeId: anchorNodeId, portId: anchor.portId },
            to: { nodeId, portId: candidate.portId },
          }
        : {
            channel: anchor.channel,
            from: { nodeId, portId: candidate.portId },
            to: { nodeId: anchorNodeId, portId: anchor.portId },
          }
    this.apply({ kind: 'insert-connected-node', nodeTypeId, nodeId, position, edge })
    return nodeId
  }

  moveNodes(positions: Array<{ nodeId: string; position: { x: number; y: number } }>): void {
    if (positions.length) this.apply({ kind: 'move-nodes', positions })
  }

  removeNodes(nodeIds: string[]): void {
    const unique = [...new Set(nodeIds)]
    if (unique.length) this.apply({ kind: 'remove-nodes', nodeIds: unique })
  }

  selectionSnapshot(nodeIds: string[]): {
    nodes: Node[]
    calls: GraphCall[]
    annotations: Annotation[]
    edges: Edge[]
  } {
    const graph = this.currentGraph
    if (!graph) return { nodes: [], calls: [], annotations: [], edges: [] }
    const selected = new Set(nodeIds)
    return {
      nodes: clone(graph.nodes.filter((node) => selected.has(node.id))),
      calls: clone(graph.calls!.filter((call) => selected.has(call.id))),
      annotations: clone(graph.annotations!.filter((annotation) => selected.has(annotation.id))),
      edges: clone(
        graph.edges.filter(
          (edge) => selected.has(edge.from.nodeId) && selected.has(edge.to.nodeId),
        ),
      ),
    }
  }

  insertNodeSelection(
    selection: {
      nodes: Node[]
      calls?: GraphCall[]
      annotations?: Annotation[]
      edges: Edge[]
    },
    offset: { x: number; y: number },
  ): string[] {
    const graph = this.currentGraph
    if (
      !graph ||
      selection.nodes.length +
        (selection.calls?.length ?? 0) +
        (selection.annotations?.length ?? 0) ===
        0
    )
      return []
    const shadow = clone(graph)
    const ids = new Map<string, string>()
    const nodes = selection.nodes.map((node) => {
      const id = uniqueNodeId(shadow, this.idFactory)
      ids.set(node.id, id)
      const inserted = {
        ...clone(node),
        id,
        position: { x: node.position.x + offset.x, y: node.position.y + offset.y },
      }
      shadow.nodes.push(inserted)
      return inserted
    })
    const calls = (selection.calls ?? []).map((call) => {
      const id = uniqueElementId(shadow, this.idFactory)
      ids.set(call.id, id)
      const inserted = {
        ...clone(call),
        id,
        position: { x: call.position.x + offset.x, y: call.position.y + offset.y },
      }
      shadow.calls!.push(inserted)
      return inserted
    })
    const annotations = (selection.annotations ?? []).map((annotation) => {
      const id = uniqueElementId(shadow, this.idFactory)
      const inserted = {
        ...clone(annotation),
        id,
        position: {
          x: annotation.position.x + offset.x,
          y: annotation.position.y + offset.y,
        },
      }
      shadow.annotations!.push(inserted)
      return inserted
    })
    const edges = selection.edges.flatMap((edge) => {
      const from = ids.get(edge.from.nodeId)
      const to = ids.get(edge.to.nodeId)
      return from && to
        ? [
            {
              ...clone(edge),
              from: { ...edge.from, nodeId: from },
              to: { ...edge.to, nodeId: to },
            },
          ]
        : []
    })
    this.apply({ kind: 'insert-node-selection', nodes, calls, annotations, edges })
    return [
      ...nodes.map((node) => node.id),
      ...calls.map((call) => call.id),
      ...annotations.map((annotation) => annotation.id),
    ]
  }

  insertStateReference(
    variable: string,
    mode: StateReferenceMode,
    position: { x: number; y: number },
  ): string {
    const accessMode = mode === 'read' || mode === 'last-change' ? 'read' : 'write'
    const nodeTypeId = [...this.projections.values()].find(
      (projection) =>
        projection.nodeRef.nodeTypeId.endsWith(`/state/${mode}`) &&
        projection.stateAccesses.some((access) => access.mode === accessMode),
    )?.nodeRef.nodeTypeId
    if (!nodeTypeId) throw new Error(`state ${mode} node is unavailable`)
    const projection = this.projections.get(nodeTypeId)!
    const [nodeId] = this.insertNodeSelection(
      {
        nodes: [
          {
            id: 'state-reference',
            nodeRef: clone(projection.nodeRef),
            position: { x: 0, y: 0 },
            config: { variable },
            bindings: {},
          },
        ],
        edges: [],
      },
      position,
    )
    return nodeId
  }

  insertLinearDraft(
    draftNodes: LinearWorkflowDraftNode[],
    origin: { x: number; y: number },
    resources: WorkflowResource[] = [],
  ): string[] {
    const graph = this.currentGraph
    if (!graph || draftNodes.length === 0) return []
    const shadow = clone(graph)
    const defaultTargetSlot = this.source?.targetDefaults?.find(
      (item) => item.target === 'target',
    )?.slot
    const nodes = draftNodes.map((draft, index): Node => {
      const projection = this.projections.get(draft.nodeTypeID)
      if (!projection) throw new Error(`draft node type ${draft.nodeTypeID} is unavailable`)
      const id = uniqueNodeId(shadow, this.idFactory)
      const bindings: Node['bindings'] = {}
      for (const [portId, value] of Object.entries(draft.values)) {
        bindings[portId] = { kind: 'value', value: clone(value) }
      }
      for (const [portId, blob] of Object.entries(draft.blobs)) {
        bindings[portId] = { kind: 'blob', blob: clone(blob) }
      }
      for (const [portId, resource] of Object.entries(draft.resources ?? {})) {
        bindings[portId] = { kind: 'resource', resource: clone(resource) }
      }
      const config = clone(draft.config)
      if (defaultTargetSlot && config.slot === defaultTargetSlot) delete config.slot
      const node: Node = {
        id,
        nodeRef: clone(projection.nodeRef),
        position: { x: origin.x + index * 280, y: origin.y },
        config,
        bindings,
      }
      shadow.nodes.push(node)
      return node
    })
    const edges = nodes.slice(1).map(
      (node, index): Edge => ({
        channel: 'exec',
        from: { nodeId: nodes[index].id, portId: draftNodes[index].execOutput },
        to: { nodeId: node.id, portId: draftNodes[index + 1].execInput },
      }),
    )
    this.applyBatch([
      ...resources.map((resource): EditorCommand => ({ kind: 'add-resource', resource })),
      { kind: 'insert-node-selection', nodes, calls: [], annotations: [], edges },
    ])
    return nodes.map((node) => node.id)
  }

  duplicateNodes(nodeIds: string[], offset = { x: 32, y: 32 }): string[] {
    return this.insertNodeSelection(this.selectionSnapshot(nodeIds), offset)
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
      this.upgradeCompatibleNodeContracts()
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
    applyCommand(next, graph, resolved, this.projections, this.typeProjections)
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

  applyBatch(commands: EditorCommand[]): void {
    if (!commands.length) return
    this.apply({ kind: 'batch', commands: clone(commands) })
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

  openGraphPath(graphPath: readonly string[]): void {
    const source = this.requireSource()
    // Runtime provenance interleaves call IDs so two calls to one subgraph
    // remain distinguishable. The editor navigation model only contains graphs.
    const graphIds = new Set(source.graphs.map((graph) => graph.id))
    const next = graphPath.filter((id) => graphIds.has(id))
    if (!next.length) next.push(source.entryGraph)
    this.graphPath = next
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
    return this.start(false, [])
  }

  async startDebug(breakpoints: DebugBreakpoint[]): Promise<RunView | null> {
    return this.start(true, breakpoints)
  }

  async controlDebug(action: 'continue' | 'pause' | 'step'): Promise<DebugSnapshot> {
    if (!this.activeRun || !this.debugSnapshot) throw new Error('debug Run is unavailable')
    const runId = this.activeRun.runId
    const snapshot = await this.transport.controlDebugRun(runId, action)
    this.acceptDebugSnapshot(runId, snapshot)
    return this.debugSnapshot ?? snapshot
  }

  async setDebugBreakpoints(breakpoints: DebugBreakpoint[]): Promise<DebugSnapshot> {
    if (!this.activeRun || !this.debugSnapshot) throw new Error('debug Run is unavailable')
    const runId = this.activeRun.runId
    const snapshot = await this.transport.setDebugBreakpoints(runId, breakpoints)
    this.acceptDebugSnapshot(runId, snapshot)
    return this.debugSnapshot ?? snapshot
  }

  acceptDebugSnapshot(runId: string, snapshot: DebugSnapshot): boolean {
    if (runId !== this.activeRun?.runId || !this.debugSnapshot) {
      if (this.debugStartPending) {
        const pending = this.pendingDebugSnapshots.get(runId)
        if (!pending || snapshot.generation >= pending.generation) {
          this.pendingDebugSnapshots.set(runId, snapshot)
        }
      }
      return false
    }
    if (snapshot.generation < this.debugSnapshot.generation) return false
    this.debugSnapshot = snapshot
    return true
  }

  async refreshRun(): Promise<RunView | null> {
    if (!this.activeRun) return null
    this.activeRun = await this.transport.getRunTimeline(this.activeRun.runId)
    if (terminalStatus(this.activeRun.status)) this.phase = 'ready'
    return this.activeRun
  }

  async loadTimelinePage(page: number): Promise<RunView | null> {
    if (!this.activeRun) return null
    this.activeRun = await this.transport.getRunTimelinePage(this.activeRun.runId, page, 200)
    return this.activeRun
  }

  async cancelRun(): Promise<RunView | null> {
    if (!this.activeRun) return null
    this.activeRun = await this.transport.cancelRun(this.activeRun.runId)
    this.phase = 'ready'
    return this.activeRun
  }

  clearRunTrace(): boolean {
    if (!this.activeRun || !terminalStatus(this.activeRun.status)) return false
    this.activeRun = null
    this.debugSnapshot = null
    this.pendingDebugSnapshots.clear()
    return true
  }

  serialize(): string {
    return JSON.stringify(this.requireSource())
  }

  private async start(debug: boolean, breakpoints: DebugBreakpoint[]): Promise<RunView | null> {
    this.failure = ''
    try {
      const compile = await this.validate()
      if (compile.diagnostics.some((diagnostic) => diagnostic.severity === 'error')) return null
      if (!compile.programHash) throw new Error('compiler produced no Program hash')
      this.phase = 'running'
      if (debug) {
        this.debugStartPending = true
        this.pendingDebugSnapshots.clear()
      }
      const started = debug
        ? await this.transport.startDebugRun(this.workflowId, breakpoints)
        : await this.transport.startRun(this.workflowId)
      this.diagnostics = [...started.diagnostics]
      this.sourceHash = started.sourceHash ?? this.sourceHash
      this.compiledHash = started.programHash ?? this.compiledHash
      this.lastRunHash = started.programHash ?? ''
      this.activeRun = started.run ?? null
      this.debugSnapshot = started.debug ?? null
      if (debug && this.activeRun) {
        const pending = this.pendingDebugSnapshots.get(this.activeRun.runId)
        if (
          pending &&
          (!this.debugSnapshot || pending.generation >= this.debugSnapshot.generation)
        ) {
          this.debugSnapshot = pending
        }
      }
      this.debugStartPending = false
      this.pendingDebugSnapshots.clear()
      if (!this.activeRun) this.phase = 'ready'
      return this.activeRun
    } catch (error) {
      this.debugStartPending = false
      this.pendingDebugSnapshots.clear()
      this.fail(error)
      throw error
    }
  }

  private loadAuthoring(raw: string): void {
    const parsed: unknown = JSON.parse(raw)
    if (!isAuthoringProjection(parsed)) throw new Error('unsupported node authoring projection')
    this.authoring = parsed
    this.projections.clear()
    this.typeProjections.clear()
    for (const projection of parsed.body.nodes) {
      this.projections.set(projection.nodeRef.nodeTypeId, projection)
    }
    for (const projection of parsed.body.types) {
      this.typeProjections.set(projection.typeRef.typeId, projection)
    }
  }

  private acceptSource(view: SourceView): void {
    if (!view.sourceJson) throw new Error('Workflow Source response omitted sourceJson')
    const parsed: unknown = JSON.parse(view.sourceJson)
    if (!isWorkflowSource(parsed) || parsed.workflow.id !== view.workflowId) {
      throw new Error('Workflow Source response has invalid identity')
    }
    for (const graph of parsed.graphs) normalizeGraph(graph)
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

  private upgradeCompatibleNodeContracts(): void {
    const source = this.requireSource()
    let upgraded = false
    for (const graph of source.graphs) {
      for (const node of graph.nodes) {
        const projection = this.projections.get(node.nodeRef.nodeTypeId)
        if (
          !projection ||
          projection.nodeRef.semanticDigest === node.nodeRef.semanticDigest ||
          !applyCompatibleNodeContractUpgrade(graph, node, projection)
        ) {
          continue
        }
        this.pendingCommands.push({
          graphId: graph.id,
          command: { kind: 'upgrade-node-contract', nodeId: node.id },
        })
        upgraded = true
      }
    }
    if (!upgraded) return
    source.revision = this.baseRevision + 1
    this.dirty = true
    this.resetCompileFacts()
  }

  private requireCurrentSubgraph(): Graph {
    const graph = this.currentGraph
    if (!graph || graph.kind !== 'subgraph') throw new Error('open a subgraph first')
    return graph
  }

  private updateCurrentGraphInterface(draft: GraphInterfaceDraft): void {
    this.apply({ kind: 'update-graph-interface', ...draft })
  }

  private graphInterfaceElements(graph: Graph): GraphInterfaceElement[] {
    const elements: GraphInterfaceElement[] = []
    for (const node of graph.nodes) {
      const projection = this.nodeInstanceProjection(node)
      if (!projection) continue
      elements.push({
        id: node.id,
        label: node.label?.trim() || node.id,
        dataInputs: projection.dataInputs.map((port) => ({
          id: port.id,
          name: port.id,
          type: clone(port.type.expression),
        })),
        dataOutputs: projection.dataOutputs.map((port) => ({
          id: port.id,
          name: port.id,
          type: clone(port.type.expression),
        })),
        signals: projection.signals.map((signal) => ({ ...signal })),
        bindings: node.bindings,
      })
    }
    for (const call of graph.calls ?? []) {
      const callee = this.calleeGraph(call)
      if (!callee) continue
      elements.push({
        id: call.id,
        label: call.label?.trim() || callee.name?.trim() || call.id,
        dataInputs: callee.inputs.map((port) => ({
          id: port.id,
          name: port.name?.trim() || port.id,
          type: clone(port.type),
        })),
        dataOutputs: callee.outputs.map((port) => ({
          id: port.id,
          name: port.name?.trim() || port.id,
          type: clone(port.type),
        })),
        signals: [
          { id: 'in', name: 'in', channel: 'exec', direction: 'input' },
          ...(callee.exits ?? []).map((exit) => ({
            id: exit.id,
            name: exit.name?.trim() || exit.id,
            channel: exit.channel,
            direction: 'output' as const,
          })),
        ],
        bindings: call.bindings,
      })
    }
    return elements
  }

  private resetCompileFacts(): void {
    this.compiledHash = ''
    this.diagnostics = []
  }

  private requireSource(): YottaWorkflowSource {
    if (!this.source) throw new Error('EditorSession has no Workflow Source')
    return this.source
  }

  private fail(error: unknown): void {
    this.phase = 'failed'
    this.failure = errorText(error)
  }
}

function collectStateReferences(
  source: YottaWorkflowSource,
  name: string,
): StateReferenceLocation[] {
  const references: StateReferenceLocation[] = []
  for (const graph of source.graphs) {
    for (const node of graph.nodes) {
      if (!node.nodeRef.nodeTypeId.includes('/nodes/state/') || node.config.variable !== name)
        continue
      references.push({
        graphId: graph.id,
        nodeId: node.id,
        mode:
          node.nodeRef.nodeTypeId.endsWith('/write') ||
          node.nodeRef.nodeTypeId.endsWith('/increment')
            ? 'write'
            : 'read',
      })
    }
  }
  return references
}

function connectionCompatibilityInGraph(
  source: YottaWorkflowSource,
  graph: Graph,
  edge: Edge,
  projections: ReadonlyMap<string, NodeProjection>,
  types: ReadonlyMap<string, TypeProjection>,
  nodes: readonly NodeProjection[],
): ConnectionCompatibility {
  const fromNode = graph.nodes.find((node) => node.id === edge.from.nodeId)
  const toNode = graph.nodes.find((node) => node.id === edge.to.nodeId)
  const fromCall = graph.calls!.find((call) => call.id === edge.from.nodeId)
  const toCall = graph.calls!.find((call) => call.id === edge.to.nodeId)
  if ((!fromNode && !fromCall) || (!toNode && !toCall))
    return { valid: false, issue: 'port', message: 'connection node is missing' }
  if (fromCall || toCall) {
    if (edge.channel === 'data') {
      const fromType = fromCall
        ? source.graphs
            .find((candidate) => candidate.id === fromCall.graphId)
            ?.outputs.find((port) => port.id === edge.from.portId)?.type
        : projections
            .get(fromNode!.nodeRef.nodeTypeId)
            ?.dataOutputs.find((port) => port.id === edge.from.portId)?.type.expression
      const toType = toCall
        ? source.graphs
            .find((candidate) => candidate.id === toCall.graphId)
            ?.inputs.find((port) => port.id === edge.to.portId)?.type
        : projections
            .get(toNode!.nodeRef.nodeTypeId)
            ?.dataInputs.find((port) => port.id === edge.to.portId)?.type.expression
      return fromType && toType && assignable(fromType, toType, types)
        ? { valid: true, disposition: 'direct' }
        : {
            valid: false,
            issue: 'type',
            message: 'connection types are incompatible',
            disposition: 'incompatible',
          }
    }
    const fromValid = fromCall
      ? source.graphs
          .find((candidate) => candidate.id === fromCall.graphId)
          ?.exits?.some((exit) => exit.id === edge.from.portId && exit.channel === edge.channel)
      : projections
          .get(fromNode!.nodeRef.nodeTypeId)
          ?.signals.some(
            (signal) =>
              signal.id === edge.from.portId &&
              signal.direction === 'output' &&
              signal.channel === edge.channel,
          )
    const toValid = toCall
      ? edge.channel === 'exec' && edge.to.portId === 'in'
      : projections
          .get(toNode!.nodeRef.nodeTypeId)
          ?.signals.some((signal) => signal.id === edge.to.portId && signal.direction === 'input')
    return fromValid && toValid
      ? { valid: true }
      : { valid: false, issue: 'port', message: 'connection ports are incompatible' }
  }
  const fromBase = projections.get(fromNode!.nodeRef.nodeTypeId)
  const toBase = projections.get(toNode!.nodeRef.nodeTypeId)
  if (!fromBase || !toBase)
    return { valid: false, issue: 'port', message: 'connection projection is missing' }
  const from = resolveNodeInstanceProjection(source, fromNode!, fromBase, projections, types)
  const to = resolveNodeInstanceProjection(source, toNode!, toBase, projections, types)
  return projectedConnectionCompatibility(
    from,
    { channel: edge.channel, direction: 'output', portId: edge.from.portId },
    to,
    { channel: edge.channel, direction: 'input', portId: edge.to.portId },
    types,
    nodes,
  )
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
  const expanded = pending.flatMap(({ graphId, command }) => {
    if (command.kind === 'remove-graph-cascade') {
      return [
        ...command.calls.map((call) => ({
          graphId: call.parentGraphId,
          command: {
            kind: 'remove-graph-call' as const,
            callId: call.callId,
          } satisfies EditorCommand,
        })),
        {
          graphId,
          command: {
            kind: 'remove-graph' as const,
            graphId: command.graphId,
          } satisfies EditorCommand,
        },
      ]
    }
    return expandEditorCommand(command).map((expandedCommand) => ({
      graphId,
      command: expandedCommand,
    }))
  })
  const generated = new Set(
    expanded.flatMap(({ command }) =>
      command.kind === 'add-node' && command.nodeId ? [command.nodeId] : [],
    ),
  )
  const nodeRef = (nodeId: string): string => (generated.has(nodeId) ? `$${nodeId}` : nodeId)
  return expanded.map(({ graphId, command }): WorkflowPatchCommand => {
    switch (command.kind) {
      case 'rename-workflow':
        return { kind: command.kind, renameWorkflow: { name: command.name } }
      case 'set-target-default':
        return {
          kind: command.kind,
          setTargetDefault: { target: command.target, slot: command.slot },
        }
      case 'clear-target-default':
        return { kind: command.kind, clearTargetDefault: { target: command.target } }
      case 'add-state-variable':
        return {
          kind: command.kind,
          addStateVariable: {
            name: command.name,
            type: clone(command.type),
            default: jsonValue(command.defaultValue),
          },
        }
      case 'update-state-variable':
        return {
          kind: command.kind,
          updateStateVariable: {
            name: command.name,
            type: clone(command.type),
            default: clone(command.defaultValue) as WorkflowJSONValue,
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
      case 'upgrade-node-contract':
        return {
          kind: command.kind,
          upgradeNodeContract: { graphId, nodeId: nodeRef(command.nodeId) },
        }
      case 'remove-node':
        return { kind: command.kind, removeNode: { graphId, nodeId: nodeRef(command.nodeId) } }
      case 'move-node':
        return {
          kind: command.kind,
          moveNode: { graphId, nodeId: nodeRef(command.nodeId), position: clone(command.position) },
        }
      case 'move-nodes':
        throw new Error('move-nodes must be expanded before persistence')
      case 'set-node-label':
        return {
          kind: command.kind,
          setNodeLabel: { graphId, nodeId: nodeRef(command.nodeId), label: command.label },
        }
      case 'set-node-disabled':
        return {
          kind: command.kind,
          setNodeDisabled: {
            graphId,
            nodeId: nodeRef(command.nodeId),
            disabled: command.disabled,
          },
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
      case 'bind-resource':
        return {
          kind: command.kind,
          bindResource: {
            graphId,
            nodeId: nodeRef(command.nodeId),
            portId: command.portId,
            resource: clone(command.resource),
          },
        }
      case 'add-resource':
        return {
          kind: command.kind,
          addResource: { resource: clone(command.resource) },
        }
      case 'replace-resource':
        return {
          kind: command.kind,
          replaceResource: {
            resourceId: command.resourceId,
            resource: clone(command.resource),
          },
        }
      case 'update-resource-metadata':
        return {
          kind: command.kind,
          updateResourceMetadata: {
            resourceId: command.resourceId,
            name: command.name,
            description: command.description,
            category: command.category,
            tags: [...command.tags],
          },
        }
      case 'remove-resource':
        return {
          kind: command.kind,
          removeResource: { resourceId: command.resourceId },
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
      case 'add-graph':
        return { kind: command.kind, addGraph: { graph: clone(command.graph) } }
      case 'rename-graph':
        return {
          kind: command.kind,
          renameGraph: { graphId: command.graphId, name: command.name },
        }
      case 'remove-graph':
        return { kind: command.kind, removeGraph: { graphId: command.graphId } }
      case 'remove-graph-cascade':
        throw new Error('remove-graph-cascade must be expanded before persistence')
      case 'update-graph-interface':
        return {
          kind: command.kind,
          updateGraphInterface: {
            graphId,
            inputs: command.inputs.map((port) => ({
              ...clone(port),
              nodeId: nodeRef(port.nodeId),
            })),
            outputs: command.outputs.map((port) => ({
              ...clone(port),
              nodeId: nodeRef(port.nodeId),
            })),
            entries: command.entries.map((entry) => ({
              ...clone(entry),
              nodeId: nodeRef(entry.nodeId),
            })),
            exits: command.exits.map((exit) => ({
              ...clone(exit),
              endpoint: { ...clone(exit.endpoint), nodeId: nodeRef(exit.endpoint.nodeId) },
            })),
          },
        }
      case 'add-graph-call':
        return { kind: command.kind, addGraphCall: { graphId, call: clone(command.call) } }
      case 'update-graph-call':
        return { kind: command.kind, updateGraphCall: { graphId, call: clone(command.call) } }
      case 'remove-graph-call':
        return { kind: command.kind, removeGraphCall: { graphId, callId: command.callId } }
      case 'fork-graph-call':
        throw new Error('fork-graph-call must be expanded before persistence')
      case 'expand-graph-call':
        throw new Error('expand-graph-call must be expanded before persistence')
      case 'add-annotation':
        return {
          kind: command.kind,
          addAnnotation: { graphId, annotation: clone(command.annotation) },
        }
      case 'update-annotation':
        return {
          kind: command.kind,
          updateAnnotation: { graphId, annotation: clone(command.annotation) },
        }
      case 'remove-annotation':
        return {
          kind: command.kind,
          removeAnnotation: { graphId, annotationId: command.annotationId },
        }
      case 'set-edge-reroutes':
        return {
          kind: command.kind,
          setEdgeReroutes: {
            graphId,
            edge: clone(command.edge),
            reroutes: clone(command.reroutes),
          },
        }
      case 'collapse-selection':
        return {
          kind: command.kind,
          collapseSelection: {
            graphId,
            subgraphId: command.subgraphId,
            callId: command.callId,
            name: command.name,
            nodeIds: [...command.nodeIds],
            position: clone(command.position),
          },
        }
      case 'insert-connected-node':
      case 'promote-output-to-state':
        throw new Error(`${command.kind} must be expanded before persistence`)
      case 'remove-nodes':
        throw new Error('remove-nodes must be expanded before persistence')
      case 'insert-node-selection':
        throw new Error('insert-node-selection must be expanded before persistence')
      case 'batch':
        throw new Error('batch must be expanded before persistence')
    }
  })
}

function expandEditorCommand(command: EditorCommand): EditorCommand[] {
  switch (command.kind) {
    case 'batch':
      return command.commands.flatMap(expandEditorCommand)
    case 'insert-connected-node':
      return [
        {
          kind: 'add-node',
          nodeTypeId: command.nodeTypeId,
          nodeId: command.nodeId,
          position: command.position,
        },
        { kind: 'connect', edge: command.edge },
      ]
    case 'promote-output-to-state':
      return [
        {
          kind: 'add-state-variable',
          name: command.name,
          type: clone(command.type),
          defaultValue: clone(command.defaultValue),
        },
        {
          kind: 'add-node',
          nodeTypeId: command.nodeTypeId,
          nodeId: command.nodeId,
          position: clone(command.position),
        },
        {
          kind: 'set-config',
          nodeId: command.nodeId,
          fieldId: command.stateConfigKey,
          value: command.name,
        },
        { kind: 'connect', edge: clone(command.edge) },
      ]
    case 'move-nodes':
      return command.positions.map(({ nodeId, position }) => ({
        kind: 'move-node',
        nodeId,
        position,
      }))
    case 'remove-nodes':
      return command.nodeIds.map((nodeId) => ({ kind: 'remove-node', nodeId }))
    case 'insert-node-selection':
      return [
        ...command.nodes.flatMap(nodeCommands),
        ...command.calls.map((call): EditorCommand => ({ kind: 'add-graph-call', call })),
        ...command.annotations.map(
          (annotation): EditorCommand => ({ kind: 'add-annotation', annotation }),
        ),
        ...command.edges.map((edge): EditorCommand => ({ kind: 'connect', edge })),
      ]
    case 'fork-graph-call':
      return [
        { kind: 'add-graph', graph: command.graph },
        { kind: 'update-graph-call', call: command.call },
      ]
    case 'expand-graph-call':
      return [
        { kind: 'remove-graph-call', callId: command.callId },
        ...command.nodes.flatMap(nodeCommands),
        ...command.calls.map((call): EditorCommand => ({ kind: 'add-graph-call', call })),
        ...command.annotations.map(
          (annotation): EditorCommand => ({ kind: 'add-annotation', annotation }),
        ),
        ...command.edges.map((edge): EditorCommand => ({ kind: 'connect', edge })),
      ]
    default:
      return [command]
  }
}

function nodeCommands(node: Node): EditorCommand[] {
  const commands: EditorCommand[] = [
    {
      kind: 'add-node',
      nodeTypeId: node.nodeRef.nodeTypeId,
      nodeId: node.id,
      position: node.position,
    },
  ]
  if (node.label) commands.push({ kind: 'set-node-label', nodeId: node.id, label: node.label })
  if (node.disabled) {
    commands.push({ kind: 'set-node-disabled', nodeId: node.id, disabled: true })
  }
  for (const [fieldId, value] of Object.entries(node.config)) {
    commands.push({ kind: 'set-config', nodeId: node.id, fieldId, value })
  }
  for (const [portId, binding] of Object.entries(node.bindings)) {
    if (binding.kind === 'value') {
      commands.push({ kind: 'bind-value', nodeId: node.id, portId, value: binding.value })
    } else if (binding.kind === 'blob' && binding.blob) {
      commands.push({ kind: 'bind-blob', nodeId: node.id, portId, blob: binding.blob })
    } else if (binding.kind === 'resource' && binding.resource) {
      commands.push({
        kind: 'bind-resource',
        nodeId: node.id,
        portId,
        resource: binding.resource,
      })
    } else if (binding.kind === 'default') {
      commands.push({ kind: 'bind-default', nodeId: node.id, portId })
    }
  }
  return commands
}

function applyCommand(
  source: YottaWorkflowSource,
  graph: Graph,
  command: EditorCommand,
  projections: Map<string, NodeProjection>,
  types: Map<string, TypeProjection>,
): void {
  const expanded = expandEditorCommand(command)
  if (expanded.length !== 1 || expanded[0] !== command) {
    for (const primitive of expanded) applyCommand(source, graph, primitive, projections, types)
    return
  }
  switch (command.kind) {
    case 'rename-workflow': {
      const name = command.name.trim()
      if (!name) throw new Error('workflow name is required')
      source.workflow.name = name
      return
    }
    case 'set-target-default': {
      const target = command.target.trim()
      const slot = command.slot.trim()
      if (!/^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$/.test(target))
        throw new Error('target default name is invalid')
      if (!/^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$/.test(slot))
        throw new Error('target default slot is invalid')
      const defaults = (source.targetDefaults ??= [])
      const existing = defaults.find((candidate) => candidate.target === target)
      if (existing) existing.slot = slot
      else defaults.push({ target, slot })
      defaults.sort((left, right) => left.target.localeCompare(right.target))
      return
    }
    case 'clear-target-default':
      source.targetDefaults = source.targetDefaults?.filter(
        (candidate) => candidate.target !== command.target,
      )
      if (!source.targetDefaults?.length) delete source.targetDefaults
      return
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
    case 'update-state-variable': {
      const variable = source.variables.find((candidate) => candidate.name === command.name)
      if (!variable) throw new Error(`state variable ${command.name} does not exist`)
      variable.type = clone(command.type)
      variable.default = clone(command.defaultValue)
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
      const config = Object.fromEntries(
        projection.configFields
          .filter((field) => field.hasDefault)
          .map((field) => [field.id, clone(field.default)]),
      )
      const bindings = Object.fromEntries(
        projection.dataInputs
          .filter((port) => port.hasDefault)
          .map((port) => [port.id, { kind: 'default' as const }]),
      )
      graph.nodes.push({
        id,
        nodeRef: clone(projection.nodeRef),
        position: clone(command.position),
        config,
        bindings,
      })
      return
    }
    case 'upgrade-node-contract': {
      const node = requireNode(graph, command.nodeId)
      const projection = projections.get(node.nodeRef.nodeTypeId)
      if (!projection || !applyCompatibleNodeContractUpgrade(graph, node, projection)) {
        throw new Error(`node ${node.id} cannot be upgraded without losing authoring data`)
      }
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
    case 'move-nodes':
      throw new Error('move-nodes expansion failed')
    case 'set-node-label': {
      const node = requireNode(graph, command.nodeId)
      const label = command.label.trim()
      if (label) node.label = label
      else delete node.label
      return
    }
    case 'set-node-disabled':
      requireNode(graph, command.nodeId).disabled = command.disabled || undefined
      return
    case 'set-config': {
      const node = requireNode(graph, command.nodeId)
      node.config[command.fieldId] = clone(command.value)
      pruneConfigDependentTopology(graph, node, projections)
      return
    }
    case 'clear-config': {
      const node = requireNode(graph, command.nodeId)
      const field = requireProjection(node, projections).configFields.find(
        (candidate) => candidate.id === command.fieldId,
      )
      if (field?.hasDefault) node.config[command.fieldId] = clone(field.default)
      else delete node.config[command.fieldId]
      pruneConfigDependentTopology(graph, node, projections)
      return
    }
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
    case 'bind-resource': {
      const node = requireNode(graph, command.nodeId)
      requireDataInput(node, command.portId, projections)
      requireWorkflowResourceBinding(source, command.resource)
      node.bindings[command.portId] = {
        kind: 'resource',
        resource: clone(command.resource),
      }
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
    case 'add-resource': {
      if (source.resources.some((resource) => resource.id === command.resource.id))
        throw new Error(`workflow resource ${command.resource.id} already exists`)
      source.resources.push(normalizeWorkflowResource(command.resource))
      source.resources.sort((left, right) => left.id.localeCompare(right.id))
      return
    }
    case 'replace-resource': {
      const index = source.resources.findIndex((candidate) => candidate.id === command.resourceId)
      if (index < 0) throw new Error(`workflow resource ${command.resourceId} does not exist`)
      if (command.resource.id !== command.resourceId)
        throw new Error('workflow resource replacement must preserve identity')
      if (command.resource.kind !== source.resources[index]!.kind)
        throw new Error('workflow resource replacement must preserve kind')
      source.resources[index] = normalizeWorkflowResource(command.resource)
      return
    }
    case 'update-resource-metadata': {
      const resource = source.resources.find((candidate) => candidate.id === command.resourceId)
      if (!resource) throw new Error(`workflow resource ${command.resourceId} does not exist`)
      const name = command.name.trim()
      if (!name) throw new Error('workflow resource name is required')
      resource.name = name
      resource.description = command.description.trim() || undefined
      resource.category = command.category.trim() || undefined
      const tags = normalizeTextSet(command.tags)
      resource.tags = tags.length ? tags : undefined
      return
    }
    case 'remove-resource': {
      if (!source.resources.some((resource) => resource.id === command.resourceId))
        throw new Error(`workflow resource ${command.resourceId} does not exist`)
      if (workflowResourceReferenceCount(source, command.resourceId) !== 0)
        throw new Error('workflow resource is still in use')
      source.resources = source.resources.filter((resource) => resource.id !== command.resourceId)
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
      validateEdge(source, graph, command.edge, projections, types)
      if (command.edge.channel === 'data') {
        const target = graph.nodes.find((node) => node.id === command.edge.to.nodeId)
        const targetCall = graph.calls!.find((call) => call.id === command.edge.to.nodeId)
        if (target) delete target.bindings[command.edge.to.portId]
        else if (targetCall) delete targetCall.bindings[command.edge.to.portId]
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
    case 'add-graph':
      if (source.graphs.some((candidate) => candidate.id === command.graph.id))
        throw new Error(`duplicate graph ${command.graph.id}`)
      source.graphs.push(normalizeGraph(clone(command.graph)))
      return
    case 'rename-graph': {
      const target = source.graphs.find((candidate) => candidate.id === command.graphId)
      if (!target) throw new Error(`graph ${command.graphId} does not exist`)
      target.name = command.name.trim() || undefined
      return
    }
    case 'remove-graph':
      if (command.graphId === source.entryGraph) throw new Error('entry graph cannot be removed')
      if (!source.graphs.some((candidate) => candidate.id === command.graphId))
        throw new Error(`graph ${command.graphId} does not exist`)
      if (
        source.graphs.some((candidate) =>
          candidate.calls?.some((call) => call.graphId === command.graphId),
        )
      )
        throw new Error(`graph ${command.graphId} is still referenced`)
      source.graphs.splice(
        source.graphs.findIndex((candidate) => candidate.id === command.graphId),
        1,
      )
      return
    case 'remove-graph-cascade': {
      if (command.graphId === source.entryGraph) throw new Error('entry graph cannot be removed')
      if (!source.graphs.some((candidate) => candidate.id === command.graphId))
        throw new Error(`graph ${command.graphId} does not exist`)
      const actual = graphCallSites(source, command.graphId)
      const expected = new Set(command.calls.map((call) => `${call.parentGraphId}\0${call.callId}`))
      if (
        actual.length !== expected.size ||
        actual.some((call) => !expected.has(`${call.parentGraphId}\0${call.callId}`))
      )
        throw new Error('subgraph call sites changed before cascade deletion')
      for (const call of actual) {
        const parent = source.graphs.find((candidate) => candidate.id === call.parentGraphId)!
        parent.calls = (parent.calls ?? []).filter((candidate) => candidate.id !== call.callId)
        parent.edges = parent.edges.filter(
          (edge) => edge.from.nodeId !== call.callId && edge.to.nodeId !== call.callId,
        )
      }
      source.graphs.splice(
        source.graphs.findIndex((candidate) => candidate.id === command.graphId),
        1,
      )
      return
    }
    case 'update-graph-interface':
      if (graph.kind !== 'subgraph') throw new Error('only subgraphs have a callable interface')
      {
        const inputs = new Set(command.inputs.map((port) => port.id))
        const outputs = new Set(command.outputs.map((port) => port.id))
        const exits = new Set(command.exits.map((exit) => exit.id))
        for (const caller of source.graphs) {
          for (const call of caller.calls ?? []) {
            if (call.graphId !== graph.id) continue
            if (Object.keys(call.bindings).some((portId) => !inputs.has(portId)))
              throw new Error('removed graph input is still bound by a call')
            if (
              caller.edges.some(
                (edge) =>
                  (edge.to.nodeId === call.id &&
                    edge.channel === 'data' &&
                    !inputs.has(edge.to.portId)) ||
                  (edge.from.nodeId === call.id &&
                    edge.channel === 'data' &&
                    !outputs.has(edge.from.portId)) ||
                  (edge.from.nodeId === call.id &&
                    edge.channel !== 'data' &&
                    !exits.has(edge.from.portId)),
              )
            )
              throw new Error('removed graph port is still connected by a call')
          }
        }
      }
      graph.inputs = clone(command.inputs)
      graph.outputs = clone(command.outputs)
      graph.entries = clone(command.entries)
      graph.exits = clone(command.exits)
      return
    case 'add-graph-call':
      if (graphElementExists(graph, command.call.id))
        throw new Error(`duplicate graph element ${command.call.id}`)
      graph.calls!.push(clone(command.call))
      return
    case 'update-graph-call': {
      const index = graph.calls!.findIndex((call) => call.id === command.call.id)
      if (index < 0) throw new Error(`call ${command.call.id} does not exist`)
      graph.calls![index] = clone(command.call)
      return
    }
    case 'remove-graph-call':
      if (!graph.calls!.some((call) => call.id === command.callId))
        throw new Error(`call ${command.callId} does not exist`)
      graph.calls = graph.calls!.filter((call) => call.id !== command.callId)
      graph.edges = graph.edges.filter(
        (edge) => edge.from.nodeId !== command.callId && edge.to.nodeId !== command.callId,
      )
      return
    case 'fork-graph-call':
    case 'expand-graph-call':
      throw new Error(`${command.kind} must be expanded before application`)
    case 'add-annotation':
      if (graph.annotations!.some((annotation) => annotation.id === command.annotation.id))
        throw new Error(`annotation ${command.annotation.id} already exists`)
      graph.annotations!.push(clone(command.annotation))
      return
    case 'update-annotation': {
      const index = graph.annotations!.findIndex(
        (annotation) => annotation.id === command.annotation.id,
      )
      if (index < 0) throw new Error(`annotation ${command.annotation.id} does not exist`)
      graph.annotations![index] = clone(command.annotation)
      return
    }
    case 'remove-annotation':
      graph.annotations = graph.annotations!.filter(
        (annotation) => annotation.id !== command.annotationId,
      )
      return
    case 'set-edge-reroutes': {
      const edge = graph.edges.find((candidate) => sameEdge(candidate, command.edge))
      if (!edge) throw new Error('edge does not exist')
      edge.presentation = command.reroutes.length
        ? { reroutes: clone(command.reroutes) }
        : undefined
      return
    }
    case 'collapse-selection':
      collapseGraphSelection(source, graph, command, projections)
      return
    case 'insert-connected-node':
    case 'promote-output-to-state':
    case 'remove-nodes':
    case 'insert-node-selection':
    case 'batch':
      throw new Error(`${command.kind} expansion failed`)
    case 'disconnect':
      graph.edges = graph.edges.filter((edge) => !sameEdge(edge, command.edge))
  }
}

function stateTypeHasDefault(type: TypeProjection): boolean {
  return type.examples.length > 0 || type.control !== 'object'
}

function defaultStateValue(type: TypeProjection): unknown {
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

function validateEdge(
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

function resolveNodeInstanceProjection(
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

// Advances only additive-compatible stale contracts. Existing authoring data
// and topology must all remain representable; otherwise the node stays stale
// for an explicit, user-directed repair.
function applyCompatibleNodeContractUpgrade(
  graph: Graph,
  node: Node,
  base: NodeProjection,
): boolean {
  if (base.nodeRef.nodeTypeId !== node.nodeRef.nodeTypeId) return false
  if (base.nodeRef.semanticDigest === node.nodeRef.semanticDigest) return true
  if (
    node.nodeRef.nodeTypeId !== PLAY_INPUT_CLIP_NODE_TYPE_ID ||
    node.nodeRef.semanticDigest !== PLAY_INPUT_CLIP_RETRACTED_SCALE_DIGEST ||
    base.nodeRef.semanticDigest !== PLAY_INPUT_CLIP_STABLE_DIGEST
  ) {
    return false
  }

  const fields = new Map(base.configFields.map((field) => [field.id, field]))
  if (Object.keys(node.config).some((fieldId) => !fields.has(fieldId))) return false
  const config = clone(node.config)
  for (const field of base.configFields) {
    if (!(field.id in config) && field.hasDefault) config[field.id] = clone(field.default)
  }
  const projection = resolveConfigDependentProjection(base, config)
  const inputs = new Map(projection.dataInputs.map((port) => [port.id, port]))
  const bindings = clone(node.bindings)
  delete bindings['turn-scale']
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
    else if (port.binding === 'required') return false
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

function resolveConfigDependentProjection(
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

function requireDataInput(
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

function pruneConfigDependentTopology(
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

function graphAt(source: YottaWorkflowSource, path: string[]): Graph {
  const graphId = path.at(-1) ?? source.entryGraph
  const graph = source.graphs.find((candidate) => candidate.id === graphId)
  if (!graph) throw new Error(`graph ${graphId} does not exist`)
  return graph
}

function uniqueNodeId(graph: Graph, idFactory: () => string): string {
  for (let attempt = 0; attempt < 32; attempt += 1) {
    const candidate = idFactory()
    if (/^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(candidate) && !graphElementExists(graph, candidate)) {
      return candidate
    }
  }
  throw new Error('could not allocate a unique node ID')
}

function uniqueElementId(graph: Graph, idFactory: () => string): string {
  return uniqueNodeId(graph, idFactory)
}

function uniqueGraphId(source: YottaWorkflowSource, idFactory: () => string): string {
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

function normalizeGraph(graph: Graph): Graph {
  graph.calls ??= []
  graph.entries ??= []
  graph.exits ??= []
  graph.annotations ??= []
  return graph
}

function graphElementExists(graph: Graph, id: string): boolean {
  return (
    graph.nodes.some((node) => node.id === id) ||
    graph.calls!.some((call) => call.id === id) ||
    graph.annotations!.some((annotation) => annotation.id === id)
  )
}

function collapseGraphSelection(
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

function graphEndpointType(
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

function graphSignalEndpointValid(
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

function resolveGraphInputProjection(
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

function isWorkflowSource(value: unknown): value is YottaWorkflowSource {
  if (typeof value !== 'object' || value === null) return false
  const source = value as Record<string, unknown>
  return (
    source.format === 'yotta.workflow' &&
    source.version === '1' &&
    typeof source.revision === 'number' &&
    typeof source.entryGraph === 'string' &&
    Array.isArray(source.graphs) &&
    Array.isArray(source.resources) &&
    Array.isArray(source.targetProfileDefinitions) &&
    Array.isArray(source.credentialRequirements) &&
    Array.isArray(source.dependencies) &&
    Array.isArray(source.variables) &&
    typeof source.workflow === 'object' &&
    source.workflow !== null
  )
}

function isAuthoringProjection(value: unknown): value is YottaNodeAuthoringProjection {
  if (typeof value !== 'object' || value === null) return false
  const document = value as Record<string, unknown>
  if (
    document.format !== 'yotta.node-authoring-projection' ||
    document.version !== '1' ||
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

function normalizeTextSet(values: readonly string[]): string[] {
  const byKey = new Map<string, string>()
  for (const raw of values) {
    const value = raw.trim()
    if (value && !byKey.has(value.toLocaleLowerCase())) {
      byKey.set(value.toLocaleLowerCase(), value)
    }
  }
  return [...byKey.values()].sort()
}

function normalizeWorkflowResource(resource: WorkflowResource): WorkflowResource {
  const next = clone(resource)
  next.id = next.id.trim()
  next.name = next.name.trim()
  if (!next.id) throw new Error('workflow resource ID is required')
  if (!next.name) throw new Error('workflow resource name is required')
  next.description = next.description?.trim() || undefined
  next.category = next.category?.trim() || undefined
  const tags = normalizeTextSet(next.tags ?? [])
  next.tags = tags.length ? tags : undefined
  return next
}

function requireWorkflowResourceBinding(
  source: YottaWorkflowSource,
  binding: ResourceBinding,
): WorkflowResource {
  const resource = source.resources.find((candidate) => candidate.id === binding.resourceId)
  if (!resource) throw new Error(`workflow resource ${binding.resourceId} does not exist`)
  if (resource.kind === 'image') {
    if (
      !binding.variantId ||
      !resource.image?.variants.some((variant) => variant.id === binding.variantId)
    ) {
      throw new Error(`workflow image resource ${binding.resourceId} variant does not exist`)
    }
  } else if (binding.variantId) {
    throw new Error(`workflow resource ${binding.resourceId} does not accept a variant`)
  }
  return resource
}

function workflowResourceReferenceCount(source: YottaWorkflowSource, resourceId: string): number {
  let count = 0
  for (const graph of source.graphs) {
    for (const owner of [...graph.nodes, ...(graph.calls ?? [])]) {
      for (const binding of Object.values(owner.bindings)) {
        if (binding.kind === 'resource' && binding.resource?.resourceId === resourceId) count++
      }
    }
  }
  return count
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export type {
  Edge,
  Graph,
  InputBinding,
  Node,
  NodeProjection,
  WorkflowResource,
  YottaWorkflowSource,
}
