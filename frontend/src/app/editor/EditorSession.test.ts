import { describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import authoringDocument from '../../../../contracts/node/3.1/builtin-authoring.json'
import type {
  TypeExpression,
  YottaNodeAuthoringProjection,
} from '../../../../contracts/node/3.1/authoring-projection'
import type { YottaWorkflowSource } from '../../../../contracts/workflow/3.1/workflow-source'
import type {
  CompileView,
  RunView,
  SourceView,
  StartRunView,
  WorkflowTransport,
  DebugSnapshot,
} from '@/app/transport/workflow'
import { assignable, EditorSession } from './EditorSession'
import { createEditorSession } from './createEditorSession'

const authoring = authoringDocument as unknown as YottaNodeAuthoringProjection
const node = (nodeTypeId: string) => {
  const projection = authoring.body.nodes.find(
    (candidate) => candidate.nodeRef.nodeTypeId === nodeTypeId,
  )
  if (!projection) throw new Error(`missing builtin node projection: ${nodeTypeId}`)
  return projection
}
const concat = node('https://schemas.yotta.dev/nodes/text/concat')
const stateRead = node('https://schemas.yotta.dev/nodes/state/read')
const greater = node('https://schemas.yotta.dev/nodes/comparison/greater-than')
const toString = node('https://schemas.yotta.dev/nodes/conversion/to-string')
const select = node('https://schemas.yotta.dev/nodes/logic/select')
const delay = node('https://schemas.yotta.dev/nodes/control/delay')
const retry = node('https://schemas.yotta.dev/nodes/control/retry')
const blobToStream = node('https://schemas.yotta.dev/nodes/conversion/blob-to-stream')
const runStarted = node('https://schemas.yotta.dev/nodes/event/run-started')
const endBranch = node('https://schemas.yotta.dev/nodes/control/end-branch')

describe('EditorSession', () => {
  it('adds a catalog node when the session is consumed through Vue reactivity', async () => {
    const source = emptySource()
    const session = createEditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => 'node_concat',
    )

    await session.load(source.workflow.id)
    const before = session.currentGraph?.nodes.length ?? 0
    const projectedNodeCount = computed(() => session.currentGraph?.nodes.length ?? 0)

    session.apply({
      kind: 'add-node',
      nodeTypeId: concat.nodeRef.nodeTypeId,
      position: { x: 100, y: 100 },
    })

    expect(session.currentGraph?.nodes).toHaveLength(before + 1)
    expect(projectedNodeCount.value).toBe(before + 1)
  })

  it('owns revision, history, compile, save and Program Run facts', async () => {
    const source = emptySource()
    const saved = sourceView(source)
    const run = runView('QUEUED')
    const transport = mockTransport(saved, run)
    const session = new EditorSession(transport, () => 'node_concat')

    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: concat.nodeRef.nodeTypeId,
      position: { x: 10, y: 20 },
    })
    session.apply({ kind: 'bind-value', nodeId: 'node_concat', portId: 'a', value: 'hello' })
    session.apply({ kind: 'bind-value', nodeId: 'node_concat', portId: 'b', value: ' world' })

    expect(session.source?.revision).toBe(1)
    expect(session.dirty).toBe(true)
    expect(session.canUndo).toBe(true)
    session.undo()
    expect(session.currentGraph?.nodes[0].bindings.b).toBeUndefined()
    session.redo()
    expect(session.currentGraph?.nodes[0].bindings.b?.value).toBe(' world')

    const started = await session.run()
    expect(started?.runId).toBe(run.runId)
    expect(session.lastRunHash).toBe('sha256:program')
    expect(session.dirty).toBe(false)
    expect(transport.applyPatch).toHaveBeenCalledWith(
      source.workflow.id,
      0,
      expect.arrayContaining([expect.objectContaining({ kind: 'add-node' })]),
    )
    expect(transport.startRun).toHaveBeenCalledWith(source.workflow.id)
  })

  it('starts and controls a true debug Run through the admitted Run transport', async () => {
    const source = emptySource()
    const run = runView('RUNNING')
    const transport = mockTransport(sourceView(source), run)
    const paused = {
      status: 'paused',
      generation: 4,
      graphId: 'main',
      nodeId: 'run-started',
      queue: [],
      inputs: {},
      outputs: {},
      state: {},
    } as DebugSnapshot
    vi.mocked(transport.startDebugRun).mockResolvedValue({
      sourceHash: 'sha256:source-next',
      programHash: 'sha256:program',
      diagnostics: [],
      run,
      debug: paused,
    } as StartRunView)
    vi.mocked(transport.controlDebugRun).mockResolvedValue({
      ...paused,
      status: 'running',
      generation: 5,
    } as DebugSnapshot)
    const session = new EditorSession(transport)

    await session.load(source.workflow.id)
    await session.startDebug([{ graphId: 'main', nodeId: 'run-started' } as never])

    expect(transport.startDebugRun).toHaveBeenCalledWith(source.workflow.id, [
      { graphId: 'main', nodeId: 'run-started' },
    ])
    expect(session.debugSnapshot?.nodeId).toBe('run-started')
    expect(
      session.acceptDebugSnapshot(run.runId, { ...paused, generation: 3 } as DebugSnapshot),
    ).toBe(false)
    await session.controlDebug('continue')
    expect(session.debugSnapshot?.status).toBe('running')
  })

  it('does not let a stale debug control response overwrite a newer event', async () => {
    const source = emptySource()
    const run = runView('RUNNING')
    const transport = mockTransport(sourceView(source), run)
    const paused = {
      status: 'paused',
      generation: 4,
      graphId: 'main',
      nodeId: 'run-started',
      queue: [],
      inputs: {},
      outputs: {},
      state: {},
    } as DebugSnapshot
    vi.mocked(transport.startDebugRun).mockResolvedValue({
      sourceHash: 'sha256:source-next',
      programHash: 'sha256:program',
      diagnostics: [],
      run,
      debug: paused,
    } as StartRunView)
    let resolveControl!: (snapshot: DebugSnapshot) => void
    vi.mocked(transport.controlDebugRun).mockImplementation(
      () => new Promise((resolve) => (resolveControl = resolve)),
    )
    const session = new EditorSession(transport)
    await session.load(source.workflow.id)
    await session.startDebug([])

    const control = session.controlDebug('step')
    const completed = {
      ...paused,
      status: 'completed',
      generation: 6,
      nodeId: '',
    } as DebugSnapshot
    expect(session.acceptDebugSnapshot(run.runId, completed)).toBe(true)
    resolveControl({ ...paused, status: 'running', generation: 5 } as DebugSnapshot)

    await expect(control).resolves.toEqual(completed)
    expect(session.debugSnapshot).toEqual(completed)
  })

  it('rejects an incompatible or wrong-carrier edge before compile', async () => {
    const source = emptySource()
    const transport = mockTransport(sourceView(source), runView('QUEUED'))
    const ids = ['blob_to_stream', 'concat']
    const session = new EditorSession(transport, () => ids.shift() ?? 'unused')
    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: blobToStream.nodeRef.nodeTypeId,
      position: { x: 0, y: 0 },
    })
    session.apply({
      kind: 'add-node',
      nodeTypeId: concat.nodeRef.nodeTypeId,
      position: { x: 200, y: 0 },
    })

    expect(() =>
      session.apply({
        kind: 'connect',
        edge: {
          channel: 'data',
          from: { nodeId: 'blob_to_stream', portId: 'stream' },
          to: { nodeId: 'concat', portId: 'a' },
        },
      }),
    ).toThrow('not assignable')
  })

  it('persists BlobRef bindings through the authoring patch protocol', async () => {
    const source = emptySource()
    const transport = mockTransport(sourceView(source), runView('QUEUED'))
    const session = new EditorSession(transport, () => 'blob_to_stream')
    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: blobToStream.nodeRef.nodeTypeId,
      position: { x: 0, y: 0 },
    })
    const blob = {
      mediaType: 'application/octet-stream',
      digest: `sha256:${'a'.repeat(64)}`,
      size: 12,
    }
    session.apply({ kind: 'bind-blob', nodeId: 'blob_to_stream', portId: 'blob', blob })
    expect(session.currentGraph?.nodes[0].bindings.blob).toEqual({ kind: 'blob', blob })
    await session.save()
    expect(transport.applyPatch).toHaveBeenCalledWith(
      source.workflow.id,
      0,
      expect.arrayContaining([expect.objectContaining({ kind: 'bind-blob' })]),
    )
  })

  it('uses nominal union and list assignability from the 3.1 contract', () => {
    const stringType = concat.dataInputs[0].type.expression
    const number = authoring.body.types.find((type) => type.typeRef.typeId.endsWith('/number/v1'))!
    const numberType: TypeExpression = { kind: 'ref', ref: number.typeRef }
    const union: TypeExpression = {
      kind: 'union',
      members: [stringType, numberType],
    }
    expect(assignable(stringType, union)).toBe(true)
    expect(assignable(union, stringType)).toBe(false)
    expect(
      assignable({ kind: 'list', element: stringType }, { kind: 'list', element: union }),
    ).toBe(false)
  })

  it('uses instruction semantics for exec and error input hints', async () => {
    const source = emptySource()
    const ids = ['delay', 'retry']
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => ids.shift() ?? 'unused',
    )
    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: delay.nodeRef.nodeTypeId,
      position: { x: 0, y: 0 },
    })
    session.apply({
      kind: 'add-node',
      nodeTypeId: retry.nodeRef.nodeTypeId,
      position: { x: 200, y: 0 },
    })

    session.apply({
      kind: 'connect',
      edge: {
        channel: 'error',
        from: { nodeId: 'delay', portId: 'failed' },
        to: { nodeId: 'retry', portId: 'retry' },
      },
    })
    expect(session.currentGraph?.edges).toHaveLength(1)
    expect(() =>
      session.apply({
        kind: 'connect',
        edge: {
          channel: 'exec',
          from: { nodeId: 'delay', portId: 'done' },
          to: { nodeId: 'retry', portId: 'retry' },
        },
      }),
    ).toThrow('invalid signal ports')
  })

  it('inserts and connects a compatible node as one undoable edit', async () => {
    const source = emptySource()
    const ids = ['delay', 'retry']
    const transport = mockTransport(sourceView(source), runView('QUEUED'))
    const session = new EditorSession(transport, () => ids.shift() ?? 'unused')
    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: delay.nodeRef.nodeTypeId,
      position: { x: 0, y: 0 },
    })

    const inserted = session.insertConnectedNode(
      'delay',
      { channel: 'error', direction: 'output', portId: 'failed' },
      retry.nodeRef.nodeTypeId,
      { channel: 'error', direction: 'input', portId: 'retry' },
      { x: 220, y: 0 },
    )

    expect(inserted).toBe('retry')
    expect(session.currentGraph?.nodes.map((candidate) => candidate.id)).toEqual(['delay', 'retry'])
    expect(session.currentGraph?.edges).toEqual([
      {
        channel: 'error',
        from: { nodeId: 'delay', portId: 'failed' },
        to: { nodeId: 'retry', portId: 'retry' },
      },
    ])

    session.undo()
    expect(session.currentGraph?.nodes.map((candidate) => candidate.id)).toEqual(['delay'])
    expect(session.currentGraph?.edges).toEqual([])
    session.redo()
    expect(session.currentGraph?.nodes).toHaveLength(2)
    expect(session.currentGraph?.edges).toHaveLength(1)

    await session.save()
    expect(transport.applyPatch).toHaveBeenCalledWith(
      source.workflow.id,
      0,
      expect.arrayContaining([
        expect.objectContaining({ kind: 'add-node' }),
        expect.objectContaining({ kind: 'connect' }),
      ]),
    )
  })

  it('inserts a visible conversion bridge as one undoable edit', async () => {
    const source = emptySource()
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => 'string-to-number',
    )
    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: concat.nodeRef.nodeTypeId,
      nodeId: 'concat',
      position: { x: 0, y: 0 },
    })
    session.apply({
      kind: 'add-node',
      nodeTypeId: greater.nodeRef.nodeTypeId,
      nodeId: 'greater',
      position: { x: 440, y: 0 },
    })
    const edge = {
      channel: 'data' as const,
      from: { nodeId: 'concat', portId: 'result' },
      to: { nodeId: 'greater', portId: 'a' },
    }
    const plan = session.connectionCompatibility(edge)
    expect(plan).toMatchObject({ valid: false, disposition: 'conversion' })
    const conversion = plan.conversions?.[0]
    if (!conversion) throw new Error('missing conversion plan')

    const inserted = session.insertConversionBridge(edge, conversion, { x: 220, y: 0 })

    expect(inserted).toBe('string-to-number')
    expect(session.currentGraph?.nodes.map((candidate) => candidate.id)).toEqual([
      'concat',
      'greater',
      'string-to-number',
    ])
    expect(session.currentGraph?.edges).toEqual([
      {
        channel: 'data',
        from: { nodeId: 'concat', portId: 'result' },
        to: { nodeId: 'string-to-number', portId: 'text' },
      },
      {
        channel: 'data',
        from: { nodeId: 'string-to-number', portId: 'result' },
        to: { nodeId: 'greater', portId: 'a' },
      },
    ])

    session.undo()
    expect(session.currentGraph?.nodes.map((candidate) => candidate.id)).toEqual([
      'concat',
      'greater',
    ])
    expect(session.currentGraph?.edges).toEqual([])
    session.redo()
    expect(session.currentGraph?.nodes).toHaveLength(3)
    expect(session.currentGraph?.edges).toHaveLength(2)
  })

  it('promotes a durable output to typed state as one undoable edit', async () => {
    const source = emptySource()
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => 'state-write',
    )
    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: concat.nodeRef.nodeTypeId,
      nodeId: 'concat',
      position: { x: 0, y: 0 },
    })

    const stateNodeId = session.promoteOutputToState('concat', 'result', 'message', {
      x: 220,
      y: 0,
    })

    expect(stateNodeId).toBe('state-write')
    expect(session.source?.variables).toEqual([
      expect.objectContaining({ name: 'message', type: concat.dataOutputs[0]!.type.expression }),
    ])
    expect(session.currentGraph?.nodes.find((candidate) => candidate.id === 'state-write')).toEqual(
      expect.objectContaining({ config: { variable: 'message' } }),
    )
    expect(session.currentGraph?.edges).toContainEqual({
      channel: 'data',
      from: { nodeId: 'concat', portId: 'result' },
      to: { nodeId: 'state-write', portId: 'value' },
    })

    session.undo()
    expect(session.source?.variables).toEqual([])
    expect(session.currentGraph?.nodes.map((candidate) => candidate.id)).toEqual(['concat'])
    session.redo()
    expect(session.source?.variables[0]?.name).toBe('message')
    expect(session.currentGraph?.nodes).toHaveLength(2)
  })

  it('changes an unreferenced state type and default atomically', async () => {
    const source = emptySource()
    const transport = mockTransport(sourceView(source), runView('QUEUED'))
    const session = new EditorSession(transport, () => 'unused')
    await session.load(source.workflow.id)
    const stringType = authoring.body.types.find((type) =>
      type.typeRef.typeId.includes('/core/string/'),
    )!
    const integerType = authoring.body.types.find((type) =>
      type.typeRef.typeId.includes('/core/integer/'),
    )!
    session.apply({
      kind: 'add-state-variable',
      name: 'value',
      type: { kind: 'ref', ref: stringType.typeRef },
      defaultValue: '',
    })
    session.apply({
      kind: 'update-state-variable',
      name: 'value',
      type: { kind: 'ref', ref: integerType.typeRef },
      defaultValue: 0,
    })
    expect(session.source?.variables[0]).toEqual({
      name: 'value',
      type: { kind: 'ref', ref: integerType.typeRef },
      default: 0,
    })
    session.undo()
    expect(session.source?.variables[0]?.type).toEqual({ kind: 'ref', ref: stringType.typeRef })
    session.redo()
    await session.save()
    expect(transport.applyPatch).toHaveBeenCalledWith(
      source.workflow.id,
      0,
      expect.arrayContaining([
        expect.objectContaining({
          kind: 'update-state-variable',
          updateStateVariable: expect.objectContaining({ name: 'value', default: 0 }),
        }),
      ]),
    )
  })

  it('inserts Delay from the RunStarted exec output', async () => {
    const source = emptySource()
    const ids = ['run-started', 'delay']
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => ids.shift() ?? 'unused',
    )
    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: runStarted.nodeRef.nodeTypeId,
      position: { x: 0, y: 0 },
    })

    expect(
      session.insertConnectedNode(
        'run-started',
        { channel: 'exec', direction: 'output', portId: 'started' },
        delay.nodeRef.nodeTypeId,
        { channel: 'exec', direction: 'input', portId: 'in' },
        { x: 220, y: 0 },
      ),
    ).toBe('delay')
    expect(session.currentGraph?.edges).toEqual([
      {
        channel: 'exec',
        from: { nodeId: 'run-started', portId: 'started' },
        to: { nodeId: 'delay', portId: 'in' },
      },
    ])
  })

  it('keeps duplicate, batch move and batch delete atomic', async () => {
    const source = emptySource()
    const ids = ['concat_a', 'concat_b', 'copy_a', 'copy_b']
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => ids.shift() ?? 'unused',
    )
    await session.load(source.workflow.id)
    session.apply({
      kind: 'add-node',
      nodeTypeId: concat.nodeRef.nodeTypeId,
      position: { x: 10, y: 20 },
    })
    session.apply({
      kind: 'add-node',
      nodeTypeId: concat.nodeRef.nodeTypeId,
      position: { x: 250, y: 20 },
    })
    session.apply({ kind: 'set-node-label', nodeId: 'concat_a', label: 'Source' })
    session.apply({ kind: 'bind-value', nodeId: 'concat_a', portId: 'a', value: 'hello' })
    session.apply({ kind: 'bind-value', nodeId: 'concat_a', portId: 'b', value: ' world' })
    session.apply({
      kind: 'connect',
      edge: {
        channel: 'data',
        from: { nodeId: 'concat_a', portId: 'result' },
        to: { nodeId: 'concat_b', portId: 'a' },
      },
    })

    expect(session.duplicateNodes(['concat_a', 'concat_b'])).toEqual(['copy_a', 'copy_b'])
    expect(session.currentGraph?.nodes).toHaveLength(4)
    expect(session.currentGraph?.nodes.find((node) => node.id === 'copy_a')).toMatchObject({
      label: 'Source',
      bindings: { a: { kind: 'value', value: 'hello' } },
      position: { x: 42, y: 52 },
    })
    expect(session.currentGraph?.edges.at(-1)).toEqual({
      channel: 'data',
      from: { nodeId: 'copy_a', portId: 'result' },
      to: { nodeId: 'copy_b', portId: 'a' },
    })
    session.undo()
    expect(session.currentGraph?.nodes).toHaveLength(2)

    session.moveNodes([
      { nodeId: 'concat_a', position: { x: 100, y: 120 } },
      { nodeId: 'concat_b', position: { x: 340, y: 120 } },
    ])
    expect(session.currentGraph?.nodes.map((node) => node.position)).toEqual([
      { x: 100, y: 120 },
      { x: 340, y: 120 },
    ])
    session.undo()
    expect(session.currentGraph?.nodes.map((node) => node.position)).toEqual([
      { x: 10, y: 20 },
      { x: 250, y: 20 },
    ])

    session.removeNodes(['concat_a', 'concat_b'])
    expect(session.currentGraph?.nodes).toEqual([])
    session.undo()
    expect(session.currentGraph?.nodes).toHaveLength(2)
    expect(session.currentGraph?.edges).toHaveLength(1)
  })

  it('inserts a linear recording draft as one undoable edit', async () => {
    const source = emptySource()
    const ids = ['recorded_a', 'recorded_b']
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => ids.shift() ?? 'unused',
    )
    await session.load(source.workflow.id)

    const inserted = session.insertLinearDraft(
      [
        {
          nodeTypeID: delay.nodeRef.nodeTypeId,
          config: {},
          values: { 'duration-milliseconds': 100 },
          blobs: {},
          execInput: 'in',
          execOutput: 'done',
        },
        {
          nodeTypeID: delay.nodeRef.nodeTypeId,
          config: {},
          values: { 'duration-milliseconds': 250 },
          blobs: {},
          execInput: 'in',
          execOutput: 'done',
        },
      ],
      { x: 320, y: 180 },
    )

    expect(inserted).toEqual(['recorded_a', 'recorded_b'])
    expect(session.currentGraph?.nodes).toHaveLength(2)
    expect(session.currentGraph?.edges).toEqual([
      {
        channel: 'exec',
        from: { nodeId: 'recorded_a', portId: 'done' },
        to: { nodeId: 'recorded_b', portId: 'in' },
      },
    ])
    session.undo()
    expect(session.currentGraph?.nodes).toEqual([])
    expect(session.currentGraph?.edges).toEqual([])
  })

  it('collapses a selection into a navigable graph call as one undoable edit', async () => {
    const source = emptySource()
    const ids = [
      'start',
      'delay',
      'end',
      'child',
      'call_child',
      'comment',
      'call_copy',
      'comment_copy',
    ]
    const transport = mockTransport(sourceView(source), runView('QUEUED'))
    const session = new EditorSession(transport, () => ids.shift() ?? 'unused')
    await session.load(source.workflow.id)
    for (const [projection, x] of [
      [runStarted, 0],
      [delay, 240],
      [endBranch, 480],
    ] as const) {
      session.apply({
        kind: 'add-node',
        nodeTypeId: projection.nodeRef.nodeTypeId,
        position: { x, y: 0 },
      })
    }
    session.apply({
      kind: 'connect',
      edge: {
        channel: 'exec',
        from: { nodeId: 'start', portId: 'started' },
        to: { nodeId: 'delay', portId: 'in' },
      },
    })
    session.apply({
      kind: 'connect',
      edge: {
        channel: 'exec',
        from: { nodeId: 'delay', portId: 'done' },
        to: { nodeId: 'end', portId: 'in' },
      },
    })

    expect(session.collapseSelection(['delay'], 'Reusable delay')).toBe('call_child')
    expect(session.currentGraph?.calls).toEqual([
      expect.objectContaining({ id: 'call_child', graphId: 'child', label: 'Reusable delay' }),
    ])
    expect(session.currentGraph?.edges).toEqual([
      expect.objectContaining({ to: { nodeId: 'call_child', portId: 'in' } }),
      expect.objectContaining({ from: { nodeId: 'call_child', portId: 'exit_done_1' } }),
    ])
    expect(session.source?.graphs.find((graph) => graph.id === 'child')).toMatchObject({
      kind: 'subgraph',
      entries: [{ nodeId: 'delay', portId: 'in' }],
      exits: [
        { id: 'exit_done_1', channel: 'exec', endpoint: { nodeId: 'delay', portId: 'done' } },
      ],
    })

    session.undo()
    expect(session.source?.graphs).toHaveLength(1)
    expect(session.currentGraph?.nodes.map((candidate) => candidate.id)).toEqual([
      'start',
      'delay',
      'end',
    ])
    session.redo()
    session.openGraphPath(['main', 'call_child', 'child'])
    expect(session.currentGraph?.id).toBe('child')
    expect(session.graphPath).toEqual(['main', 'child'])

    session.openGraphPath(['main'])
    expect(session.addAnnotation({ x: 80, y: 160 })).toBe('comment')
    const copied = session.insertNodeSelection(
      session.selectionSnapshot(['call_child', 'comment']),
      { x: 32, y: 32 },
    )
    expect(copied).toEqual(['call_copy', 'comment_copy'])
    expect(session.currentGraph?.calls).toHaveLength(2)
    expect(session.currentGraph?.annotations).toHaveLength(2)

    await session.save()
    expect(transport.applyPatch).toHaveBeenCalledWith(
      source.workflow.id,
      0,
      expect.arrayContaining([
        expect.objectContaining({ kind: 'collapse-selection' }),
        expect.objectContaining({ kind: 'add-graph-call' }),
        expect.objectContaining({ kind: 'add-annotation' }),
      ]),
    )
  })

  it('inserts an existing subgraph call and reuses the callee input projection', async () => {
    const source = emptySource()
    const duration = delay.dataInputs.find((port) => port.id === 'duration-milliseconds')!
    source.graphs.push({
      id: 'child',
      name: 'Child',
      kind: 'subgraph',
      nodes: [
        {
          id: 'wait',
          nodeRef: delay.nodeRef,
          position: { x: 0, y: 0 },
          config: {},
          bindings: {},
        },
      ],
      edges: [],
      inputs: [],
      outputs: [],
      entries: [],
      exits: [],
    })
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => 'call_child',
    )
    await session.load(source.workflow.id)
    session.enterGraph('child')
    session.inferCurrentGraphInterface()
    expect(session.currentGraph).toMatchObject({
      entries: [{ nodeId: 'wait', portId: 'in' }],
      inputs: [
        {
          id: 'input_duration-milliseconds_1',
          nodeId: 'wait',
          portId: 'duration-milliseconds',
        },
      ],
      exits: expect.arrayContaining([
        expect.objectContaining({
          channel: 'exec',
          endpoint: { nodeId: 'wait', portId: 'done' },
        }),
      ]),
    })
    session.openGraphPath(['main'])

    expect(session.graphInputProjection('child', 'input_duration-milliseconds_1')).toMatchObject({
      id: 'input_duration-milliseconds_1',
      type: duration.type,
    })
    expect(session.insertGraphCall('child', { x: 120, y: 80 })).toBe('call_child')
    session.apply({
      kind: 'update-graph-call',
      call: {
        ...session.currentGraph!.calls![0],
        bindings: { 'input_duration-milliseconds_1': { kind: 'value', value: 250 } },
      },
    })
    expect(session.currentGraph?.calls?.[0]).toMatchObject({
      graphId: 'child',
      bindings: { 'input_duration-milliseconds_1': { kind: 'value', value: 250 } },
    })
  })

  it('owns typed state declarations and prevents deleting referenced slots', async () => {
    const source = emptySource()
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => 'read',
    )
    await session.load(source.workflow.id)
    const stringType = authoring.body.types.find((type) =>
      type.typeRef.typeId.includes('/core/string/'),
    )!

    session.apply({
      kind: 'add-state-variable',
      name: 'message',
      type: { kind: 'ref', ref: stringType.typeRef },
      defaultValue: 'initial',
    })
    expect(session.source?.variables).toEqual([
      { name: 'message', type: { kind: 'ref', ref: stringType.typeRef }, default: 'initial' },
    ])
    expect(() =>
      session.apply({
        kind: 'add-state-variable',
        name: 'message',
        type: { kind: 'ref', ref: stringType.typeRef },
        defaultValue: '',
      }),
    ).toThrow('duplicate state variable')

    session.apply({
      kind: 'add-node',
      nodeTypeId: stateRead.nodeRef.nodeTypeId,
      position: { x: 0, y: 0 },
    })
    session.apply({ kind: 'set-config', nodeId: 'read', fieldId: 'variable', value: 'message' })
    expect(
      session.nodeInstanceProjection(session.currentGraph!.nodes[0]!)?.dataOutputs[0]?.type
        .expression,
    ).toEqual({
      kind: 'ref',
      ref: stringType.typeRef,
    })
    expect(() => session.apply({ kind: 'remove-state-variable', name: 'message' })).toThrow(
      'still referenced',
    )
    session.apply({ kind: 'remove-node', nodeId: 'read' })
    session.apply({ kind: 'remove-state-variable', name: 'message' })
    expect(session.source?.variables).toEqual([])
  })

  it('creates a typed state read atomically and connects Integer to Number consumers', async () => {
    const source = emptySource()
    const ids = ['read', 'greater', 'to-string', 'last-change', 'increment']
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => ids.shift() ?? 'unused',
    )
    await session.load(source.workflow.id)
    const integer = authoring.body.types.find((type) =>
      type.typeRef.typeId.includes('/core/integer/'),
    )!
    session.apply({
      kind: 'add-state-variable',
      name: 'index',
      type: { kind: 'ref', ref: integer.typeRef },
      defaultValue: 0,
    })
    expect(session.insertStateReference('index', 'read', { x: 0, y: 0 })).toBe('read')
    session.apply({
      kind: 'add-node',
      nodeTypeId: greater.nodeRef.nodeTypeId,
      position: { x: 200, y: 0 },
    })
    expect(() =>
      session.apply({
        kind: 'connect',
        edge: {
          channel: 'data',
          from: { nodeId: 'read', portId: 'result' },
          to: { nodeId: 'greater', portId: 'a' },
        },
      }),
    ).not.toThrow()
    session.apply({
      kind: 'add-node',
      nodeTypeId: toString.nodeRef.nodeTypeId,
      position: { x: 200, y: 100 },
    })

    expect(session.insertStateReference('index', 'last-change', { x: 0, y: 200 })).toBe(
      'last-change',
    )
    expect(session.insertStateReference('index', 'increment', { x: 0, y: 300 })).toBe('increment')
    expect(
      session.currentGraph?.nodes.find((candidate) => candidate.id === 'increment')?.config,
    ).toEqual({ variable: 'index' })
    session.apply({
      kind: 'connect',
      edge: {
        channel: 'data',
        from: { nodeId: 'read', portId: 'result' },
        to: { nodeId: 'to-string', portId: 'value' },
      },
    })
    expect(
      session.nodeInstanceProjection(session.currentGraph!.nodes[2]!)?.dataInputs[0]?.type
        .expression,
    ).toEqual({ kind: 'ref', ref: integer.typeRef })
    const number = authoring.body.types.find((type) =>
      type.typeRef.typeId.includes('/core/number/'),
    )!
    expect(
      session.stateTypeChangeImpact('index', { kind: 'ref', ref: number.typeRef }),
    ).toMatchObject({
      references: [
        { nodeId: 'read', mode: 'read' },
        { nodeId: 'last-change', mode: 'read' },
        { nodeId: 'increment', mode: 'write' },
      ],
      issues: [],
    })
    expect(() =>
      session.apply({
        kind: 'update-state-variable',
        name: 'index',
        type: { kind: 'ref', ref: number.typeRef },
        defaultValue: 0,
      }),
    ).not.toThrow()
  })

  it('previews every referenced state edge and blocks implicit conversion migrations', async () => {
    const source = emptySource()
    const ids = ['read', 'concat']
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => ids.shift() ?? 'unused',
    )
    await session.load(source.workflow.id)
    const string = authoring.body.types.find((type) =>
      type.typeRef.typeId.includes('/core/string/'),
    )!
    const integer = authoring.body.types.find((type) =>
      type.typeRef.typeId.includes('/core/integer/'),
    )!
    session.apply({
      kind: 'add-state-variable',
      name: 'message',
      type: { kind: 'ref', ref: string.typeRef },
      defaultValue: '',
    })
    session.insertStateReference('message', 'read', { x: 0, y: 0 })
    session.apply({
      kind: 'add-node',
      nodeTypeId: concat.nodeRef.nodeTypeId,
      position: { x: 200, y: 0 },
    })
    session.apply({ kind: 'bind-value', nodeId: 'concat', portId: 'b', value: '' })
    session.apply({
      kind: 'connect',
      edge: {
        channel: 'data',
        from: { nodeId: 'read', portId: 'result' },
        to: { nodeId: 'concat', portId: 'a' },
      },
    })

    const impact = session.stateTypeChangeImpact('message', {
      kind: 'ref',
      ref: integer.typeRef,
    })
    expect(impact.references).toEqual([{ graphId: 'main', nodeId: 'read', mode: 'read' }])
    expect(impact.issues).toEqual([
      expect.objectContaining({
        graphId: 'main',
        disposition: 'conversion',
        conversions: expect.arrayContaining([
          expect.objectContaining({ nodeTypeId: toString.nodeRef.nodeTypeId }),
        ]),
      }),
    ])
    expect(
      session.stateTypeChangeImpact('message', { kind: 'ref', ref: string.typeRef }).issues,
    ).toEqual([])
  })

  it('propagates instance types through a chain of generic nodes', async () => {
    const source = emptySource()
    const ids = ['read', 'select', 'to-string']
    const session = new EditorSession(
      mockTransport(sourceView(source), runView('QUEUED')),
      () => ids.shift() ?? 'unused',
    )
    await session.load(source.workflow.id)
    const integer = authoring.body.types.find((type) =>
      type.typeRef.typeId.includes('/core/integer/'),
    )!
    session.apply({
      kind: 'add-state-variable',
      name: 'index',
      type: { kind: 'ref', ref: integer.typeRef },
      defaultValue: 0,
    })
    session.insertStateReference('index', 'read', { x: 0, y: 0 })
    session.apply({
      kind: 'add-node',
      nodeTypeId: select.nodeRef.nodeTypeId,
      position: { x: 200, y: 0 },
    })
    session.apply({
      kind: 'add-node',
      nodeTypeId: toString.nodeRef.nodeTypeId,
      position: { x: 400, y: 0 },
    })
    session.apply({
      kind: 'connect',
      edge: {
        channel: 'data',
        from: { nodeId: 'read', portId: 'result' },
        to: { nodeId: 'select', portId: 'when_true' },
      },
    })
    session.apply({
      kind: 'connect',
      edge: {
        channel: 'data',
        from: { nodeId: 'select', portId: 'result' },
        to: { nodeId: 'to-string', portId: 'value' },
      },
    })
    expect(
      session.nodeInstanceProjection(session.currentGraph!.nodes[2]!)?.dataInputs[0]?.type
        .expression,
    ).toEqual({ kind: 'ref', ref: integer.typeRef })
  })
})

function emptySource(): YottaWorkflowSource {
  return {
    format: 'yotta.workflow',
    version: '3.1',
    workflow: { id: 'workflow_test', name: 'Test workflow' },
    revision: 0,
    entryGraph: 'main',
    graphs: [{ id: 'main', kind: 'main', nodes: [], edges: [], inputs: [], outputs: [] }],
    variables: [],
    secretRefs: [],
  }
}

function sourceView(source: YottaWorkflowSource): SourceView {
  return {
    workflowId: source.workflow.id,
    name: source.workflow.name,
    revision: source.revision,
    sourceHash: 'sha256:source',
    sourceJson: JSON.stringify(source),
  } as SourceView
}

function runView(status: string): RunView {
  return {
    runId: '01980a13-0000-7000-8000-000000000001',
    status,
    generation: 0,
    recordDigest: 'sha256:record',
    programHash: 'sha256:program',
    queuedAt: '2026-07-15T00:00:00Z',
    timeline: [],
  } as RunView
}

function mockTransport(saved: SourceView, run: RunView): WorkflowTransport {
  return {
    listSources: vi.fn(async () => [saved]),
    querySources: vi.fn(async () => ({ items: [saved], total: 1, page: 1, pageSize: 20 })),
    listSourceRecoveries: vi.fn(async () => []),
    repairSourceRecovery: vi.fn(async () => saved),
    deleteSourceRecovery: vi.fn(async () => undefined),
    previewDeleteSources: vi.fn(async () => []),
    deleteSources: vi.fn(async () => []),
    createSource: vi.fn(async () => saved),
    chooseSourceBundle: vi.fn(async () => ''),
    chooseSourceBundleDestination: vi.fn(async () => ''),
    chooseSourceBundleDirectory: vi.fn(async () => ''),
    inspectSourceBundle: vi.fn(async () => ({
      workflowId: saved.workflowId,
      name: saved.name,
      revision: saved.revision,
      sourceHash: saved.sourceHash,
      blobCount: 0,
      blobBytes: 0,
    })),
    importSourceBundle: vi.fn(async () => saved),
    replaceSourceFromBundle: vi.fn(async () => saved),
    exportSourceBundle: vi.fn(async () => ({
      workflowId: saved.workflowId,
      exported: true,
    })),
    exportSourceBundles: vi.fn(async () => []),
    getSource: vi.fn(async () => saved),
    applyPatch: vi.fn(async (_workflowId: string, baseRevision: number) => {
      if (!saved.sourceJson) throw new Error('mock source omitted sourceJson')
      const source = JSON.parse(saved.sourceJson) as YottaWorkflowSource
      source.revision = baseRevision + 1
      return { source: sourceView(source), generatedNodes: [] }
    }),
    compileSource: vi.fn(
      async () =>
        ({
          sourceHash: 'sha256:source-next',
          programHash: 'sha256:program',
          diagnostics: [],
        }) as CompileView,
    ),
    startRun: vi.fn(
      async () =>
        ({
          sourceHash: 'sha256:source-next',
          programHash: 'sha256:program',
          diagnostics: [],
          run,
        }) as StartRunView,
    ),
    startDebugRun: vi.fn(
      async () =>
        ({
          sourceHash: 'sha256:source-next',
          programHash: 'sha256:program',
          diagnostics: [],
          run,
          debug: null,
        }) as StartRunView,
    ),
    getDebugSnapshot: vi.fn(async () => ({ status: 'paused', generation: 1 }) as never),
    controlDebugRun: vi.fn(async () => ({ status: 'paused', generation: 2 }) as never),
    setDebugBreakpoints: vi.fn(async () => ({ status: 'paused', generation: 2 }) as never),
    cancelRun: vi.fn(async () => runView('CANCELLED')),
    cancelAllRuns: vi.fn(async () => undefined),
    getRunTimeline: vi.fn(async () => run),
    getAuthoringProjection: vi.fn(async () => JSON.stringify(authoring)),
  }
}
