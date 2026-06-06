import type { Ref } from 'vue'
import type { Container, VarDecl, Graph, GraphNode } from '@/lib/backend'

export interface DeleteVarOptions {
  cascade: boolean
}

const VAR_NODE_KINDS = new Set(['GetVar', 'SetVar', 'IncVar', 'VarLastChange'])

/**
 * useVarMutations — Container.Vars 增删改查 + scope-aware rename + cascade delete.
 * 所有 mutation 直接改 draft.value (caller 用 applyDraftMutation 包裹保证 dirty+sync).
 */
export function useVarMutations(draft: Ref<Container | null>) {
  function isContainerScope(n: GraphNode): boolean {
    // scope=auto / global / undefined → resolves to container var (applies rename/delete)
    // scope=local → frame-private, preserved (allowed to shadow container var)
    const scope = (n.config?.scope as string | undefined) ?? 'auto'
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
        if (!VAR_NODE_KINDS.has(node.kind)) continue
        if (node.config?.varName !== oldName) continue
        if (!isContainerScope(node)) continue  // preserve scope=local
        node.config.varName = newName
      }
    })
  }

  function countUsage(name: string): number {
    if (!draft.value) return 0
    let count = 0
    walkAllGraphs(draft.value, g => {
      for (const node of g.nodes) {
        if (!VAR_NODE_KINDS.has(node.kind)) continue
        if (node.config?.varName !== name) continue
        if (!isContainerScope(node)) continue
        count++
      }
    })
    return count
  }

  function listUsageNodeIDs(name: string): string[] {
    if (!draft.value) return []
    const ids: string[] = []
    walkAllGraphs(draft.value, g => {
      for (const node of g.nodes) {
        if (!VAR_NODE_KINDS.has(node.kind)) continue
        if (node.config?.varName !== name) continue
        if (!isContainerScope(node)) continue
        ids.push(node.id)
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
        if (node.config?.varName !== name) return true
        if (!isContainerScope(node)) return true
        removedIDs.add(node.id)
        return false
      })
      g.edges = g.edges.filter(e => {
        const fromID = e.from.split('.')[0]
        const toID = e.to.split('.')[0]
        return !removedIDs.has(fromID) && !removedIDs.has(toID)
      })
    })
  }

  function reorderVars(fromIdx: number, toIdx: number): void {
    if (!draft.value || !draft.value.vars) return
    if (fromIdx < 0 || fromIdx >= draft.value.vars.length) return
    if (toIdx < 0 || toIdx >= draft.value.vars.length) return
    const [moved] = draft.value.vars.splice(fromIdx, 1)
    draft.value.vars.splice(toIdx, 0, moved)
  }

  return { addVar, renameVar, deleteVar, reorderVars, countUsage, listUsageNodeIDs }
}
