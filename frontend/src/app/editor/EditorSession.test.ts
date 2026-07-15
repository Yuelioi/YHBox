import { describe, expect, it, vi } from 'vitest'
import authoringDocument from '../../../../contracts/node/3.1/builtin-authoring.json'
import type {
  TypeExpression,
  YottaNodeAuthoringProjection31,
} from '../../../../contracts/node/3.1/authoring-projection'
import type { YottaWorkflowSource31 } from '../../../../contracts/workflow/3.1/workflow-source'
import type {
  CompileView,
  RunView,
  SourceView,
  StartRunView,
  WorkflowTransport,
} from '@/app/transport/workflow31'
import { assignable, EditorSession } from './EditorSession'

const authoring = authoringDocument as unknown as YottaNodeAuthoringProjection31
const concat = authoring.body.nodes.find((node) => node.nodeRef.nodeTypeId.includes('/concat/'))!

describe('EditorSession', () => {
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
    expect(transport.saveSource).toHaveBeenCalledWith(expect.any(String), 0)
    expect(transport.startRun).toHaveBeenCalledWith(source.workflow.id)
  })

  it('rejects an incompatible or wrong-carrier edge before compile', async () => {
    const source = emptySource()
    const transport = mockTransport(sourceView(source), runView('QUEUED'))
    const ids = ['blob_to_stream', 'concat']
    const session = new EditorSession(transport, () => ids.shift() ?? 'unused')
    await session.load(source.workflow.id)
    const blobToStream = authoring.body.nodes.find((node) =>
      node.nodeRef.nodeTypeId.includes('/blob-to-stream/'),
    )!
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

  it('uses nominal union and list assignability from the 3.1 contract', () => {
    const stringType = concat.dataInputs[0].type.expression
    const union: TypeExpression = {
      kind: 'union',
      members: [stringType, authoring.body.nodes[0].dataInputs[0].type.expression],
    }
    expect(assignable(stringType, union)).toBe(true)
    expect(assignable(union, stringType)).toBe(false)
    expect(
      assignable({ kind: 'list', element: stringType }, { kind: 'list', element: union }),
    ).toBe(true)
  })
})

function emptySource(): YottaWorkflowSource31 {
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

function sourceView(source: YottaWorkflowSource31): SourceView {
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
    saveSource: vi.fn(async (raw: string) => {
      const source = JSON.parse(raw) as YottaWorkflowSource31
      return sourceView(source)
    }),
    compileDraft: vi.fn(
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
