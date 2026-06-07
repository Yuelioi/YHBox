import type { Ref } from 'vue'
import type { Container, VarDecl, Graph, GraphNode } from '@/lib/backend'
import { NODE_FIELD_SCHEMAS } from '@/components/containers/nodeFieldSchemas'

export interface DeleteVarOptions {
  cascade: boolean
}

export interface UsageRef {
  nodeID: string
  access: 'read' | 'write'
  kind: string
}

const VAR_NODE_KINDS = new Set(['GetVar', 'SetVar', 'IncVar', 'VarLastChange'])
// VAR_READ_KINDS 是读引用 kind；VAR_NODE_KINDS 里其余成员视为写。新增变量节点 kind 时须同步这两个 Set。
const VAR_READ_KINDS = new Set(['GetVar', 'VarLastChange'])

// 某 kind 的捕获框字段名 (semantic==='capture'); 来自 NODE_FIELD_SCHEMAS 派生.
function captureFieldsOf(kind: string): string[] {
  return (NODE_FIELD_SCHEMAS[kind] ?? []).filter(f => f.semantic === 'capture').map(f => f.key)
}

// 读 node.config.literal[field] 的 string 值, 不存在返回 ''.
function captureLiteralOf(node: GraphNode, field: string): string {
  const lit = node.config?.literal as Record<string, unknown> | undefined
  return typeof lit?.[field] === 'string' ? (lit[field] as string) : ''
}

// VAR_NODE_KINDS (GetVar/SetVar/IncVar/VarLastChange) 的变量名 = VarName pin 字面量 = config.literal.VarName.
// (跟后端 PinString(n,"VarName") / 真实存盘 shape 对齐; 历史小写 config.varName 是错的, 后端读不到。)
function varNodeName(node: GraphNode): string {
  return captureLiteralOf(node, 'VarName')
}

/**
 * useVarMutations — Container.Vars 增删改查 + scope-aware rename + cascade delete.
 * 所有 mutation 直接改 draft.value (caller 用 applyDraftMutation 包裹保证 dirty+sync).
 */
export function useVarMutations(draft: Ref<Container | null>) {
  function isContainerScope(n: GraphNode): boolean {
    // scope=auto / global / undefined → resolves to container var (applies rename/delete)
    // scope=local → frame-private, preserved (allowed to shadow container var)
    // Scope pin 字面量 = config.literal.Scope (VarLastChange 无 Scope pin → undefined → auto)。
    const scope = captureLiteralOf(n, 'Scope') || 'auto'
    return scope !== 'local'
  }

  function walkAllGraphs(d: Container, fn: (g: Graph) => void) {
    fn(d.graph)
    for (const sg of d.subgraphs ?? []) {
      if (sg.graph) fn(sg.graph)
    }
  }

  function addVar(v: VarDecl): void {
    if (!draft.value) throw new Error('draft is null')
    const existing = (draft.value.vars ?? []).some(x => x.name === v.name)
    if (existing) throw new Error(`Variable '${v.name}' already exists`)
    if (!draft.value.vars) draft.value.vars = []
    draft.value.vars.push(v)
  }

  function renameVar(oldName: string, newName: string): void {
    if (!draft.value) return
    if (oldName === newName) return
    const decl = (draft.value.vars ?? []).find(v => v.name === oldName)
    if (!decl) return  // nonexistent → no-op
    const dup = (draft.value.vars ?? []).some(v => v.name === newName)
    if (dup) throw new Error(`Variable '${newName}' already exists`)
    decl.name = newName
    walkAllGraphs(draft.value, g => {
      for (const node of g.nodes) {
        // VAR_NODE_KINDS: rename config.literal.VarName
        if (VAR_NODE_KINDS.has(node.kind)) {
          if (varNodeName(node) === oldName && isContainerScope(node)) {
            (node.config!.literal as Record<string, unknown>).VarName = newName
          }
        }
        // capture fields: rename config.literal[field]
        const captureFields = captureFieldsOf(node.kind)
        if (captureFields.length > 0) {
          const lit = node.config?.literal as Record<string, unknown> | undefined
          if (lit) {
            for (const f of captureFields) {
              if (lit[f] === oldName) lit[f] = newName
            }
          }
        }
      }
    })
  }

  function countUsage(name: string): number {
    if (!draft.value) return 0
    let count = 0
    walkAllGraphs(draft.value, g => {
      for (const node of g.nodes) {
        // VAR_NODE_KINDS: count config.literal.VarName refs
        if (VAR_NODE_KINDS.has(node.kind)) {
          if (varNodeName(node) === name && isContainerScope(node)) {
            count++
          }
        }
        // capture fields: count config.literal[field] refs
        for (const f of captureFieldsOf(node.kind)) {
          if (captureLiteralOf(node, f) === name) count++
        }
      }
    })
    return count
  }

  function listUsageNodeIDs(name: string): string[] {
    if (!draft.value) return []
    const ids: string[] = []
    walkAllGraphs(draft.value, g => {
      for (const node of g.nodes) {
        let matched = false
        // VAR_NODE_KINDS
        if (VAR_NODE_KINDS.has(node.kind) && varNodeName(node) === name && isContainerScope(node)) {
          matched = true
        }
        // capture fields
        if (!matched) {
          for (const f of captureFieldsOf(node.kind)) {
            if (captureLiteralOf(node, f) === name) { matched = true; break }
          }
        }
        if (matched) ids.push(node.id)
      }
    })
    return ids
  }

  function deleteVar(name: string, opts: DeleteVarOptions): void {
    if (!draft.value) return
    draft.value.vars = (draft.value.vars ?? []).filter(v => v.name !== name)
    if (!opts.cascade) return
    walkAllGraphs(draft.value, g => {
      const removedIDs = new Set<string>()
      g.nodes = g.nodes.filter(node => {
        if (!VAR_NODE_KINDS.has(node.kind)) return true
        if (varNodeName(node) !== name) return true
        if (!isContainerScope(node)) return true
        removedIDs.add(node.id)
        return false
      })
      g.edges = g.edges.filter(e => {
        const fromID = e.from.split('.')[0]
        const toID = e.to.split('.')[0]
        return !removedIDs.has(fromID) && !removedIDs.has(toID)
      })
      // cascade: clear capture fields on surviving nodes (保留节点, 清空字段)
      for (const node of g.nodes) {
        const captureFields = captureFieldsOf(node.kind)
        if (captureFields.length === 0) continue
        const lit = node.config?.literal as Record<string, unknown> | undefined
        if (!lit) continue
        for (const f of captureFields) {
          if (lit[f] === name) lit[f] = ''
        }
      }
    })
  }

  function listUsageRefs(name: string): UsageRef[] {
    if (!draft.value) return []
    const refs: UsageRef[] = []
    walkAllGraphs(draft.value, g => {
      for (const node of g.nodes) {
        // VAR_NODE_KINDS: read vs write based on kind
        if (VAR_NODE_KINDS.has(node.kind) && varNodeName(node) === name && isContainerScope(node)) {
          const access: 'read' | 'write' = VAR_READ_KINDS.has(node.kind) ? 'read' : 'write'
          refs.push({ nodeID: node.id, access, kind: node.kind })
        }
        // capture fields → write
        for (const f of captureFieldsOf(node.kind)) {
          if (captureLiteralOf(node, f) === name) {
            refs.push({ nodeID: node.id, access: 'write', kind: node.kind })
            break  // 同一节点多个 capture 字段命中同名变量, 只计一条
          }
        }
      }
    })
    return refs
  }

  function reorderVars(fromIdx: number, toIdx: number): void {
    if (!draft.value || !draft.value.vars) return
    if (fromIdx < 0 || fromIdx >= draft.value.vars.length) return
    if (toIdx < 0 || toIdx >= draft.value.vars.length) return
    const [moved] = draft.value.vars.splice(fromIdx, 1)
    draft.value.vars.splice(toIdx, 0, moved)
  }

  return { addVar, renameVar, deleteVar, reorderVars, countUsage, listUsageNodeIDs, listUsageRefs }
}
