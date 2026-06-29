// useInlineMenu — pin 拖到空白弹「添加节点」菜单状态机单测。
// 复发根因: 成功连线时 vue-flow 同步先 emit connect(→markConnectSuccess 清空 connectionStart),
// 再 emit connect-end; onVfConnectEnd 第一行 `if (!connectionStart) return` 早退, 那个负责把
// connectMadeThisGesture 重置回 false 的 setTimeout 永不排上 → flag 残留 true → 紧接着「拖出口到
// 空白」被误判成功而不开菜单 (用户报"拖拽有概率拖不出来", 实为每次成功连线后必失效一次)。
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { computed } from 'vue'
import type { Graph, GraphNode } from '@/lib/backend'
import { __resetForTests, register } from '@/components/containers/nodeRegistry/registry'

vi.mock('@vue-flow/core', () => ({
  useVueFlow: () => ({
    screenToFlowCoordinate: (p: { x: number; y: number }) => p,
  }),
}))

import { useInlineMenu } from './useInlineMenu'

function node(id: string, kind: string): GraphNode {
  return { id, kind, x: 0, y: 0, config: {} } as GraphNode
}

function setup() {
  const graph: Graph = { id: 'g', version: 1, nodes: [node('a', 'Log')], edges: [] } as Graph
  const activeGraph = computed<Graph | null>(() => graph)
  const m = useInlineMenu({
    activeGraph,
    applyDraftMutation: (mutator) => mutator({} as any),
    syncFlowFromDraft: () => {},
  })
  return { m, graph }
}

// 一次「从 pin a.out 拖到空白」(无 @connect) — 期望开菜单。
function dragPinToEmpty(m: ReturnType<typeof setup>['m']) {
  m.onVfConnectStart({ nodeId: 'a', handleId: 'out', handleType: 'source' })
  m.onVfConnectEnd({ clientX: 100, clientY: 100 } as MouseEvent)
  vi.runAllTimers()
}

// 一次「成功连线」(vue-flow 先 connect 再 connect-end)。
function dragPinToConnect(m: ReturnType<typeof setup>['m']) {
  m.onVfConnectStart({ nodeId: 'a', handleId: 'out', handleType: 'source' })
  m.markConnectSuccess() // @connect 先同步 fire
  m.onVfConnectEnd({ clientX: 0, clientY: 0 } as MouseEvent) // @connect-end 紧随
  vi.runAllTimers()
}

describe('useInlineMenu pin 拖到空白', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    __resetForTests()
  })
  afterEach(() => vi.useRealTimers())

  it('拖 pin 到空白 → 开「添加节点」菜单', () => {
    const { m } = setup()
    dragPinToEmpty(m)
    expect(m.inlineMenu.value.open).toBe(true)
  })

  it('成功连线 → 不开菜单', () => {
    const { m } = setup()
    dragPinToConnect(m)
    expect(m.inlineMenu.value.open).toBe(false)
  })

  it('成功连线后紧接着拖 pin 到空白 → 仍开菜单 (不被残留 flag 吞掉)', () => {
    const { m } = setup()
    dragPinToConnect(m)
    expect(m.inlineMenu.value.open).toBe(false)
    dragPinToEmpty(m)
    expect(m.inlineMenu.value.open).toBe(true)
  })

  it('VueFlow 连接缺 sourceHandle/targetHandle 时不放行', () => {
    const { m } = setup()
    expect(
      m.isValidVueFlowConnection({
        source: 'a',
        target: 'b',
        sourceHandle: null,
        targetHandle: 'In',
      }),
    ).toBe(false)
    expect(
      m.isValidVueFlowConnection({
        source: 'a',
        target: 'b',
        sourceHandle: 'Done',
        targetHandle: undefined,
      }),
    ).toBe(false)
  })

  it('新增多个带嵌套默认 config 的节点时不共享 literal 对象', () => {
    register({
      kind: 'Win32WindowTarget',
      group: 'target',
      labelZh: 'node.Win32WindowTarget.label',
      description: '',
      example: '',
      visual: { icon: '', bg: '', border: '' },
      execIn: ['In'],
      execOut: ['Done'],
      dataIn: {},
      dataOut: {},
      fields: [],
      defaults: { literal: { Title: '', Class: '', ProcessName: '', TitleMatch: 'exact' } },
    })
    const { m, graph } = setup()

    m.openInlineMenuAt(10, 20)
    m.onInlineMenuPick('Win32WindowTarget')
    m.openInlineMenuAt(30, 40)
    m.onInlineMenuPick('Win32WindowTarget')

    const targets = graph.nodes.filter((n) => n.kind === 'Win32WindowTarget')
    ;((targets[0].config as any).literal as Record<string, unknown>).Title = 'AE'

    expect((targets[1].config as any).literal.Title).toBe('')
    expect((targets[0].config as any).literal).not.toBe((targets[1].config as any).literal)
  })
})
