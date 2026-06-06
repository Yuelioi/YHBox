import { describe, it, expect, afterEach } from 'vitest'
import { ref } from 'vue'
import { useVarMutations } from './useVarMutations'
import type { Container } from '@/lib/backend'
import { NODE_FIELD_SCHEMAS } from '@/components/containers/nodeFieldSchemas'

function makeDraft(): Container {
  return {
    schemaVersion: 1, id: 'c1', name: 'c1',
    vars: [{ name: 'x', type: 'number', default: 1 }],
    graph: {
      id: 'g1', version: 1,
      nodes: [
        { id: 'gv1', kind: 'GetVar', x: 0, y: 0, config: { varName: 'x', scope: 'auto' }, createdAt: '2026-05-19T00:00:00Z' },
        { id: 'gv2', kind: 'GetVar', x: 0, y: 0, config: { varName: 'x', scope: 'local' }, createdAt: '2026-05-19T00:00:00Z' },
        { id: 'sv',  kind: 'SetVar', x: 0, y: 0, config: { varName: 'x', scope: 'global' }, createdAt: '2026-05-19T00:00:00Z' },
      ],
      edges: [],
    },
    subgraphs: [],
    tags: [], createdAt: '', updatedAt: '',
  } as unknown as Container
}

describe('useVarMutations', () => {
  it('renameVar: changes scope=auto/global/undefined nodes only, preserves scope=local', () => {
    const draft = ref<Container>(makeDraft())
    const m = useVarMutations(draft)
    m.renameVar('x', 'state')
    expect(draft.value.vars![0].name).toBe('state')
    const nodes = draft.value.graph.nodes
    expect(nodes.find(n => n.id === 'gv1')!.config!.varName).toBe('state')  // auto
    expect(nodes.find(n => n.id === 'gv2')!.config!.varName).toBe('x')      // local preserved
    expect(nodes.find(n => n.id === 'sv')!.config!.varName).toBe('state')   // global
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
        id: 'g2', version: 1,
        nodes: [
          { id: 'vlc1', kind: 'VarLastChange', x: 0, y: 0, config: { varName: 'hp', scope: 'auto' }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    expect(m.countUsage('hp')).toBeGreaterThanOrEqual(1)
  })

  it('VarLastChange: renameVar renames VarLastChange node config.varName', () => {
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c3', name: 'c3',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g3', version: 1,
        nodes: [
          { id: 'vlc1', kind: 'VarLastChange', x: 0, y: 0, config: { varName: 'hp', scope: 'auto' }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    m.renameVar('hp', 'x')
    expect(draft.value.graph.nodes[0].config!.varName).toBe('x')
  })

  // ── Task 12: 捕获框字段纳入 refs/rename/cascade-clear ──────────────────────
  afterEach(() => {
    delete NODE_FIELD_SCHEMAS['CheckTemplate']
  })

  it('capture field: countUsage includes capture field values', () => {
    NODE_FIELD_SCHEMAS['CheckTemplate'] = [
      { key: 'CaptureFound', label: '', type: 'text', semantic: 'capture' } as any,
    ]
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c4', name: 'c4',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g4', version: 1,
        nodes: [
          { id: 'ct1', kind: 'CheckTemplate', x: 0, y: 0, config: { literal: { CaptureFound: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    expect(m.countUsage('hp')).toBeGreaterThanOrEqual(1)
  })

  it('capture field: renameVar renames capture field value', () => {
    NODE_FIELD_SCHEMAS['CheckTemplate'] = [
      { key: 'CaptureFound', label: '', type: 'text', semantic: 'capture' } as any,
    ]
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c5', name: 'c5',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g5', version: 1,
        nodes: [
          { id: 'ct1', kind: 'CheckTemplate', x: 0, y: 0, config: { literal: { CaptureFound: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    m.renameVar('hp', 'x')
    const lit = draft.value.graph.nodes[0].config!.literal as Record<string, string>
    expect(lit['CaptureFound']).toBe('x')
  })

  it('capture field: deleteVar(cascade=true) clears capture field, preserves node', () => {
    NODE_FIELD_SCHEMAS['CheckTemplate'] = [
      { key: 'CaptureFound', label: '', type: 'text', semantic: 'capture' } as any,
    ]
    const draft = ref<Container>({
      schemaVersion: 1, id: 'c6', name: 'c6',
      vars: [{ name: 'hp', type: 'number', default: 0 }],
      graph: {
        id: 'g6', version: 1,
        nodes: [
          { id: 'ct1', kind: 'CheckTemplate', x: 0, y: 0, config: { literal: { CaptureFound: 'hp' } }, createdAt: '2026-05-19T00:00:00Z' },
        ],
        edges: [],
      },
      subgraphs: [], tags: [], createdAt: '', updatedAt: '',
    } as unknown as Container)
    const m = useVarMutations(draft)
    m.deleteVar('hp', { cascade: true })
    expect(draft.value.graph.nodes).toHaveLength(1)  // 节点保留
    expect(draft.value.graph.nodes[0].id).toBe('ct1')
    const lit = draft.value.graph.nodes[0].config!.literal as Record<string, string>
    expect(lit['CaptureFound']).toBe('')  // 字段清空
  })
})
