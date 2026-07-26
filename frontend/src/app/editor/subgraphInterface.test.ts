import { describe, expect, it } from 'vitest'
import type { Graph, TypeExpression } from '../../../../contracts/workflow/current/workflow-source'
import {
  addGraphInterfaceCandidate,
  graphInterfaceReferences,
  inferGraphInterface,
  moveGraphInterfaceItem,
  projectGraphInterfaceCandidates,
  renameGraphInterfaceItem,
  type GraphInterfaceElement,
} from './subgraphInterface'

const stringType: TypeExpression = {
  kind: 'ref',
  ref: {
    typeId: 'https://schemas.yotta.dev/types/core/string/v1',
    semanticDigest: `sha256:${'1'.repeat(64)}`,
  },
}

describe('subgraph interface model', () => {
  it('projects only open internal endpoints and marks already published endpoints', () => {
    const graph = childGraph()
    graph.edges.push({
      channel: 'data',
      from: { nodeId: 'first', portId: 'result' },
      to: { nodeId: 'second', portId: 'value' },
    })
    graph.inputs.push({
      id: 'input_text_1',
      name: '文本',
      type: stringType,
      nodeId: 'first',
      portId: 'value',
    })
    const candidates = projectGraphInterfaceCandidates(graph, elements())

    expect(
      candidates.map((candidate) => [
        candidate.kind,
        candidate.endpoint.nodeId,
        candidate.endpoint.portId,
        candidate.published,
      ]),
    ).toEqual([
      ['entry', 'first', 'in', false],
      ['exit', 'first', 'done', false],
      ['input', 'first', 'value', true],
      ['entry', 'second', 'in', false],
      ['exit', 'second', 'done', false],
      ['output', 'second', 'result', false],
    ])
  })

  it('adds, renames, and reorders ports without changing their stable IDs', () => {
    const graph = childGraph()
    const candidates = projectGraphInterfaceCandidates(graph, elements())
    const first = candidates.find(
      (candidate) =>
        candidate.kind === 'input' &&
        candidate.endpoint.nodeId === 'first' &&
        candidate.endpoint.portId === 'value',
    )!
    const second = candidates.find(
      (candidate) =>
        candidate.kind === 'input' &&
        candidate.endpoint.nodeId === 'second' &&
        candidate.endpoint.portId === 'value',
    )!

    Object.assign(graph, addGraphInterfaceCandidate(graph, first))
    Object.assign(graph, addGraphInterfaceCandidate(graph, second))
    const stableID = graph.inputs[0]!.id
    expect(graph.inputs.map((port) => port.name)).toEqual(['文本', '文本 2'])
    expect(() => renameGraphInterfaceItem(graph, 'input', stableID, '文本 2')).toThrow(
      'already in use',
    )
    Object.assign(graph, renameGraphInterfaceItem(graph, 'input', stableID, '目标文本'))
    Object.assign(graph, moveGraphInterfaceItem(graph, 'input', stableID, 1))

    expect(graph.inputs[1]).toMatchObject({ id: stableID, name: '目标文本' })
  })

  it('infers an interface while preserving IDs and user-facing names at matching endpoints', () => {
    const graph = childGraph()
    graph.entries = [{ nodeId: 'first', portId: 'in' }]
    graph.inputs = [
      {
        id: 'stable-text',
        name: '目标文本',
        type: stringType,
        nodeId: 'first',
        portId: 'value',
      },
    ]
    graph.exits = [
      {
        id: 'stable-done',
        name: '完成',
        channel: 'exec',
        endpoint: { nodeId: 'first', portId: 'done' },
      },
    ]
    graph.edges.push({
      channel: 'exec',
      from: { nodeId: 'first', portId: 'done' },
      to: { nodeId: 'second', portId: 'in' },
    })
    const preview = inferGraphInterface(graph, projectGraphInterfaceCandidates(graph, elements()))

    expect(preview.draft.inputs[0]).toMatchObject({ id: 'stable-text', name: '目标文本' })
    expect(preview.draft.exits.some((exit) => exit.id === 'stable-done')).toBe(false)
    expect(preview.removed).toContainEqual({ kind: 'exit', id: 'stable-done' })
  })

  it('reports caller bindings and edges before an interface item is removed', () => {
    const child = childGraph()
    child.inputs = [
      {
        id: 'stable-text',
        name: '目标文本',
        type: stringType,
        nodeId: 'first',
        portId: 'value',
      },
    ]
    const main = mainGraph()
    main.calls = [
      {
        id: 'call-child',
        graphId: child.id,
        label: '发送文本',
        position: { x: 0, y: 0 },
        bindings: { 'stable-text': { kind: 'value', value: 'hello' } },
      },
    ]
    main.edges = [
      {
        channel: 'data',
        from: { nodeId: 'source', portId: 'result' },
        to: { nodeId: 'call-child', portId: 'stable-text' },
      },
    ]
    const source = { graphs: [main, child] }

    expect(graphInterfaceReferences(source, child.id, 'input', 'stable-text')).toEqual([
      expect.objectContaining({ callId: 'call-child', usage: 'binding' }),
      expect.objectContaining({ callId: 'call-child', usage: 'edge' }),
    ])
  })
})

function elements(): GraphInterfaceElement[] {
  return ['first', 'second'].map((id) => ({
    id,
    label: id,
    dataInputs: [{ id: 'value', name: '文本', type: stringType }],
    dataOutputs: [{ id: 'result', name: '结果', type: stringType }],
    signals: [
      { id: 'in', channel: 'exec', direction: 'input' },
      { id: 'done', name: '完成', channel: 'exec', direction: 'output' },
    ],
    bindings: {},
  }))
}

function childGraph(): Graph {
  return {
    id: 'child',
    kind: 'subgraph',
    name: '子图',
    nodes: [],
    calls: [],
    edges: [],
    inputs: [],
    outputs: [],
    entries: [],
    exits: [],
    annotations: [],
  }
}

function mainGraph(): Graph {
  return { ...childGraph(), id: 'main', kind: 'main', name: '主图' }
}
