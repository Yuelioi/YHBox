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
const delay = node('https://schemas.yotta.dev/nodes/control/delay')
const retry = node('https://schemas.yotta.dev/nodes/control/retry')
const blobToStream = node('https://schemas.yotta.dev/nodes/conversion/blob-to-stream')

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

    const started = await session.debug()
    expect(started?.runId).toBe(run.runId)
    expect(session.debugging).toBe(true)
    expect(session.lastRunHash).toBe('sha256:program')
    expect(session.dirty).toBe(false)
    expect(transport.applyPatch).toHaveBeenCalledWith(
      source.workflow.id,
      0,
      expect.arrayContaining([expect.objectContaining({ kind: 'add-node' })]),
    )
    expect(transport.startRun).toHaveBeenCalledWith(source.workflow.id)
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
    ).toBe(true)
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
    expect(() => session.apply({ kind: 'remove-state-variable', name: 'message' })).toThrow(
      'still referenced',
    )
    session.apply({ kind: 'remove-node', nodeId: 'read' })
    session.apply({ kind: 'remove-state-variable', name: 'message' })
    expect(session.source?.variables).toEqual([])
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
    createSource: vi.fn(async () => saved),
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
    cancelRun: vi.fn(async () => runView('CANCELLED')),
    cancelAllRuns: vi.fn(async () => undefined),
    getRunTimeline: vi.fn(async () => run),
    getAuthoringProjection: vi.fn(async () => JSON.stringify(authoring)),
  }
}
