import { describe, expect, it } from 'vitest'
import type { Graph } from '../../../../contracts/workflow/current/workflow-source'
import { graphHandle } from './graphHandles'
import {
  analyzeCollapseBoundary,
  graphBoundaryBindingFromConnection,
  projectGraphBoundaries,
} from './workflowGraphBoundary'

describe('workflow graph boundary authoring', () => {
  it('projects one entry, named exits, data boundaries, and presentation-only edges', () => {
    const graph = subgraph()
    const projection = projectGraphBoundaries(graph)

    expect(projection.nodes.map((node) => node.data!.role)).toEqual([
      'entry',
      'exit',
      'exit',
      'output',
    ])
    expect(projection.edges).toHaveLength(5)
    expect(projection.edges.every((edge) => edge.data?.boundaryKey)).toBe(true)
    expect(graph.nodes).toHaveLength(2)
    expect(graph.edges).toHaveLength(2)
  })

  it('keeps an entry marker visible in a new empty subgraph', () => {
    const graph = subgraph()
    graph.nodes = []
    graph.edges = []
    graph.inputs = []
    graph.outputs = []
    graph.entries = []
    graph.exits = []

    const projection = projectGraphBoundaries(graph)

    expect(projection.nodes).toHaveLength(1)
    expect(projection.nodes[0]?.data?.role).toBe('entry')
  })

  it('turns a boundary gesture into a Source-native entry binding', () => {
    const graph = subgraph()
    const entry = projectGraphBoundaries(graph).nodes.find((node) => node.data?.role === 'entry')!

    expect(
      graphBoundaryBindingFromConnection(
        {
          source: entry.id,
          sourceHandle: graphHandle('exec', 'output', 'in'),
          target: 'work',
          targetHandle: graphHandle('exec', 'input', 'in'),
        },
        graph,
      ),
    ).toEqual({ kind: 'entry', endpoint: { nodeId: 'work', portId: 'in' } })
  })

  it('identifies the conflicting incoming edge before a multi-entry collapse', () => {
    const graph = subgraph()
    graph.edges.push({
      channel: 'exec',
      from: { nodeId: 'outside', portId: 'done' },
      to: { nodeId: 'finish', portId: 'in' },
    })

    const issue = analyzeCollapseBoundary(graph, new Set(['work', 'finish']))

    expect(issue?.kind).toBe('multiple-entry')
    expect(issue?.edges).toHaveLength(2)
  })
})

function subgraph(): Graph {
  return {
    id: 'child',
    name: 'Child',
    kind: 'subgraph',
    nodes: [
      {
        id: 'work',
        nodeRef: { nodeTypeId: 'work', version: '1', semanticDigest: 'sha256:work' },
        position: { x: 160, y: 80 },
        config: {},
        bindings: {},
      },
      {
        id: 'finish',
        nodeRef: { nodeTypeId: 'finish', version: '1', semanticDigest: 'sha256:finish' },
        position: { x: 460, y: 80 },
        config: {},
        bindings: {},
      },
    ],
    edges: [
      {
        channel: 'exec',
        from: { nodeId: 'outside', portId: 'done' },
        to: { nodeId: 'work', portId: 'in' },
      },
      {
        channel: 'exec',
        from: { nodeId: 'work', portId: 'done' },
        to: { nodeId: 'finish', portId: 'in' },
      },
    ],
    entries: [{ nodeId: 'work', portId: 'in' }],
    exits: [
      { id: 'completed', channel: 'exec', endpoint: { nodeId: 'finish', portId: 'done' } },
      { id: 'failed', channel: 'error', endpoint: { nodeId: 'work', portId: 'failed' } },
    ],
    inputs: [
      {
        id: 'duration',
        nodeId: 'work',
        portId: 'duration',
        type: { kind: 'ref', ref: { typeId: 'duration', semanticDigest: 'sha256:duration' } },
      },
    ],
    outputs: [
      {
        id: 'result',
        nodeId: 'finish',
        portId: 'result',
        type: { kind: 'ref', ref: { typeId: 'result', semanticDigest: 'sha256:result' } },
      },
    ],
  }
}
