import { describe, it, expect, beforeEach } from 'vitest'
import { ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useVarMutations } from './useVarMutations'
import type { Container } from '@/lib/backend'
import { register, __resetForTests } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry'

// 最小 NodeKindSpec — capture 用例注册假节点 (bindableFields 走 registry).
function makeSpec(partial: Partial<NodeKindSpec> & { kind: string }): NodeKindSpec {
  return {
    group: 'detect', labelZh: '', description: '', example: '',
    visual: { icon: '', bg: '', border: '' },
    execIn: ['In'], execOut: [], dataIn: {}, dataOut: {}, fields: [], defaults: {},
    ...partial,
  }
}

function makeDraft(): Container {
  return {
    schemaVersion: 1, id: 'c1', name: 'c1',
    vars: [{ name: 'x', type: 'number', default: 1 }],
    graph: {
      id: 'g1', schemaVersion: 1,
      nodes: [
        { id: 'gv1', kind: 'GetVar', x: 0, y: 0, config: { literal: { VarName: 'x', Scope: 'auto' } }, createdAt: '2026-05-19T00:00:00Z' },
        { id: 'gv2', kind: 'GetVar', x: 0, y: 0, config: { literal: { VarName: 'x', Scope: 'local' } }, createdAt: '2026-05-19T00:00:00Z' },
        { id: 'sv',  kind: 'SetVar', x: 0, y: 0, config: { literal: { VarName: 'x', Scope: 'global' } }, createdAt: '2026-05-19T00:00:00Z' },
      ],
      edges: [],
    },
    subgraphs: [],
    tags: [], createdAt: '', updatedAt: '',
  } as unknown as Container
}

describe('useVarMutations', () => {
  // walkAllGraphs 全局化后经 containerEditor store 解析引用闭包 — 需要活的 pinia。
  // CheckTemplate 注册假 spec (dataOut={Found}) → bindableFields('CheckTemplate')=['Found'], 供捕获用例。
  beforeEach(() => {
    setActivePinia(createPinia())
    __resetForTests()
    register(makeSpec({ kind: 'CheckTemplate', execOut: ['Found', 'NotFound'], dataOut: { Found: 'bool' } }))
  })

  it('renameVar: changes scope=auto/global/undefined nodes only, preserves scope=local', () => {
    const draft = ref<Container>(makeDraft())
    const m = useVarMutations(draft)
    m.renameVar('x', 'state')
    expect(draft.value.vars![0].name).toBe('state')
    const nodes = draft.value.graph.nodes
    const litOf = (id: string) => (nodes.find(n => n.id === id)!.config!.literal as Record<string, unknown>).VarName
    expect(litOf('gv1')).toBe('state')  // auto
    expect(litOf('gv2')).toBe('x')      // local preserved
    expect(litOf('sv')).toBe('state')   // global
  })

  it('renameVar: nonexistent var is no-op', () => {
    const draft = ref<Container>(makeDraft())
    const m = useVarMutations(draft)
    const before = JSON.stringify(draft.value)
    m.renameVar('nonexistent', 'whatever')
    expect(JSON.stringify(draft.value)).toBe(before)
  })

  it('renameVar: duplicate target name rejected', () => {
    const draft = ref<Container>(makeDraft())
    draft.value.vars!.push({ name: 'y', type: 'number', default: 0 })
    const m = useVarMutations(draft)
    expect(() => m.renameVar('x', 'y')).toThrow(/already exists/i)
  })

  it('countUsage: returns count of non-local references', () => {
    const draft = ref<Container>(makeDraft())
    const m = useVarMutations(draft)
    expect(m.countUsage('x')).toBe(2)  // gv1 (auto) + sv (global), NOT gv2 (local)
  })

  it('addVar: adds new VarDecl to draft.vars', () => {
    const draft = ref<Container>(makeDraft())
    const m = useVarMutations(draft)
    m.addVar({ name: 'y', type: 'number', default: 0 })
    expect(draft.value.vars).toHaveLength(2)
    expect(draft.value.vars![1].name).toBe('y')
  })

  it('addVar: duplicate name rejected', () => {
    const draft = ref<Container>(makeDraft())
    const m = useVarMutations(draft)
    expect(() => m.addVar({ name: 'x', type: 'string', default: '' })).toThrow(/already exists/i)
  })

  it('deleteVar(cascade=false): removes decl only, references untouched', () => {
    const draft = ref<Container>(makeDraft())
    const m = useVarMutations(draft)
    m.deleteVar('x', { cascade: false })
    expect(draft.value.vars).toHaveLength(0)
    expect(draft.value.graph.nodes).toHaveLength(3)  // nodes preserved
  })

  it('deleteVar(cascade=true): removes decl + non-local referencing nodes + their edges', () => {
    const draft = ref<Container>(makeDraft())
    draft.value.graph.edges = [
      { from: 'gv1.value', to: 'sv.value' },
    ]
    const m = useVarMutations(draft)
    m.deleteVar('x', { cascade: true })
    expect(draft.value.vars).toHaveLength(0)
    expect(draft.value.graph.nodes.map(n => n.id)).toEqual(['gv2'])  // only local preserved
    expect(draft.value.graph.edges).toHaveLength(0)  // edge of removed nodes gone
  })

  it('reorderVars: moves var from idx to idx', () => {
    const draft = ref<Container>(makeDraft())
    draft.value.vars!.push({ name: 'y', type: 'number', default: 2 })
    const m = useVarMutations(draft)
    m.reorderVars(0, 1)
    expect(draft.value.vars!.map(v => v.name)).toEqual(['y', 'x'])
  })

  // ── Task 11: VarLastChange 纳入 VAR_NODE_KINDS ──────────────────────────────
  it('VarLastChange: countUsage includes VarLastChange nodes', () => {
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c2', name: 'c2',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g2', schemaVersion: 1,
        nodes: [
          { id: 'vlc1', kind: 'VarLastChange', x: 0, y: 0, config: { literal: { VarName: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    expect(m.countUsage('hp')).toBe(1)
  })

  it('VarLastChange: renameVar renames VarLastChange node config.literal.VarName', () => {
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c3', name: 'c3',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g3', schemaVersion: 1,
        nodes: [
          { id: 'vlc1', kind: 'VarLastChange', x: 0, y: 0, config: { literal: { VarName: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    m.renameVar('hp', 'x')
    expect((draft.value.graph.nodes[0].config!.literal as Record<string, unknown>).VarName).toBe('x')
  })

  // ── 输出捕获 (config.capture) 纳入 refs/rename/cascade-delete (Spec C) ──────────────
  // CheckTemplate 在 beforeEach 注册 (dataOut={Found}) → bindableFields('CheckTemplate')=['Found']。

  it('capture: countUsage includes config.capture bindings', () => {
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c4', name: 'c4',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g4', schemaVersion: 1,
        nodes: [
          { id: 'ct1', kind: 'CheckTemplate', x: 0, y: 0, config: { capture: { Found: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    expect(m.countUsage('hp')).toBe(1)
  })

  it('capture: renameVar renames config.capture binding value', () => {
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c5', name: 'c5',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g5', schemaVersion: 1,
        nodes: [
          { id: 'ct1', kind: 'CheckTemplate', x: 0, y: 0, config: { capture: { Found: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    m.renameVar('hp', 'x')
    const cap = draft.value.graph.nodes[0].config!.capture as Record<string, string>
    expect(cap['Found']).toBe('x')
  })

  it('listUsageRefs: read for GetVar, write for config.capture binding', () => {
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c13', name: 'c13',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g13', schemaVersion: 1,
        nodes: [
          { id: 'gv1', kind: 'GetVar', x: 0, y: 0, config: { literal: { VarName: 'hp', Scope: 'auto' } }, createdAt: '2026-05-19T00:00:00Z' },
          { id: 'ct1', kind: 'CheckTemplate', x: 0, y: 0, config: { capture: { Found: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    const refs = m.listUsageRefs('hp')
    expect(refs).toHaveLength(2)
    const readRef = refs.find(r => r.nodeID === 'gv1')
    const writeRef = refs.find(r => r.nodeID === 'ct1')
    expect(readRef?.access).toBe('read')
    expect(readRef?.kind).toBe('GetVar')
    expect(writeRef?.access).toBe('write')
    expect(writeRef?.kind).toBe('CheckTemplate')
  })

  it('capture: deleteVar(cascade=true) deletes config.capture key, preserves node', () => {
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c6', name: 'c6',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g6', schemaVersion: 1,
        nodes: [
          { id: 'ct1', kind: 'CheckTemplate', x: 0, y: 0, config: { capture: { Found: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    m.deleteVar('hp', { cascade: true })
    expect(draft.value.graph.nodes).toHaveLength(1)  // 节点保留
    expect(draft.value.graph.nodes[0].id).toBe('ct1')
    const cap = draft.value.graph.nodes[0].config!.capture as Record<string, string>
    expect('Found' in cap).toBe(false)  // 键被删 (非置空串, 落地精度#2)
  })
})
