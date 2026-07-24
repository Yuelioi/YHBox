import { describe, expect, it } from 'vitest'
import type { Graph } from '../../../../contracts/workflow/current/workflow-source'
import { projectGraphDefinitions } from './subgraphManagement'

describe('subgraph management projection', () => {
  it('distinguishes definitions from calls and reports every reference location', () => {
    const source = {
      entryGraph: 'main',
      graphs: [
        graph('main', 'main', '主图', [
          call('call-a', 'child-a', '第一次调用'),
          call('call-b', 'child-b', ''),
        ]),
        graph('child-a', 'subgraph', '子图', [call('nested-b', 'child-b', '嵌套调用')]),
        graph('child-b', 'subgraph', '子图'),
      ],
    }

    const projected = projectGraphDefinitions(source)

    expect(projected.map((item) => item.id)).toEqual(['main', 'child-a', 'child-b'])
    expect(projected[1]).toMatchObject({
      name: '子图',
      duplicateName: true,
      callCount: 1,
      references: [
        {
          parentGraphId: 'main',
          parentGraphName: '主图',
          callId: 'call-a',
          callLabel: '第一次调用',
        },
      ],
    })
    expect(projected[2]?.callCount).toBe(2)
    expect(projected[2]?.references.map((reference) => reference.callId)).toEqual([
      'call-b',
      'nested-b',
    ])
  })

  it('projects interface health and searches names, IDs, and call locations', () => {
    const child = graph('child-technical-id', 'subgraph', '发送按键')
    child.entries = [{ nodeId: 'keys', portId: 'in' }]
    child.exits = [
      { id: 'completed', channel: 'exec', endpoint: { nodeId: 'keys', portId: 'completed' } },
      { id: 'failed', channel: 'error', endpoint: { nodeId: 'keys', portId: 'failed' } },
    ]
    const source = {
      entryGraph: 'main',
      graphs: [graph('main', 'main', '主图', [call('invoke-keys', child.id, '战斗按键')]), child],
    }

    expect(projectGraphDefinitions(source)[1]).toMatchObject({
      entryBound: true,
      dataInputCount: 0,
      dataOutputCount: 0,
      execExitCount: 1,
      errorExitCount: 1,
      interfaceHealthy: true,
    })
    expect(projectGraphDefinitions(source, '战斗按键').map((item) => item.id)).toEqual([
      'child-technical-id',
    ])
    expect(projectGraphDefinitions(source, 'technical').map((item) => item.id)).toEqual([
      'child-technical-id',
    ])
  })
})

function graph(
  id: string,
  kind: Graph['kind'],
  name: string,
  calls: NonNullable<Graph['calls']> = [],
): Graph {
  return {
    id,
    kind,
    name,
    nodes: [],
    calls,
    edges: [],
    inputs: [],
    outputs: [],
    entries: [],
    exits: [],
    annotations: [],
  }
}

function call(id: string, graphId: string, label: string): NonNullable<Graph['calls']>[number] {
  return { id, graphId, label, position: { x: 0, y: 0 }, bindings: {} }
}
