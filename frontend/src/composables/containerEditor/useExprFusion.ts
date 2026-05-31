// Expr fusion — 把 `A.value → B.in_x` 的 Expr 链合并成单个 Expr.
//
// Algorithm:
//   1. Verify A.kind === 'Expr', B.kind === 'Expr', A.value has fan-out 1 to B.targetPin.
//   2. Rename A.inputs[] entries that collide with B.inputs[] names (suffix with int).
//   3. In A.expr string, substitute renamed input identifiers (regex word-boundary).
//   4. In B.expr string, replace identifier matching targetPin with `(<A.expr-renamed>)`.
//   5. B.inputs: drop targetPin entry, append A's (renamed) inputs.
//   6. Reroute A's incoming data edges to B (with renamed pin name).
//   7. Delete A node + A→B edge.
//
// Caveats:
//   - Pin literal on B.targetPin is dropped (replaced by inlined expr) — config.literal[targetPin] removed.
//   - Identifier collision detection uses simple set; doesn't account for shadowing.
//   - Does NOT modify position — caller may want to call auto-layout afterwards.
import type { Ref } from 'vue'
import { edgeKind } from '@/components/containers/pinSpec'

interface FusionDeps {
  activeGraph: Ref<any>
  syncFlowFromDraft: () => void
}

interface ExprInput {
  name: string
  type: string
}

export function useExprFusion(deps: FusionDeps) {
  /**
   * fuse merges Expr `sourceNodeID` into Expr `targetNodeID`'s `targetInputName` pin.
   * Returns true on success, false if preconditions fail (caller may show toast).
   */
  function fuse(sourceNodeID: string, targetNodeID: string, targetInputName: string): boolean {
    if (!deps.activeGraph.value) return false
    const g = deps.activeGraph.value
    const a = g.nodes.find((n: any) => n.id === sourceNodeID)
    const b = g.nodes.find((n: any) => n.id === targetNodeID)
    if (!a || !b || a.kind !== 'Expr' || b.kind !== 'Expr') return false

    const aExpr = String(a.config?.expr ?? '')
    const bExpr = String(b.config?.expr ?? '')
    const aInputs: ExprInput[] = (a.config?.inputs ?? []).map((i: any) => ({ name: String(i.name), type: String(i.type ?? 'any') }))
    const bInputs: ExprInput[] = (b.config?.inputs ?? []).map((i: any) => ({ name: String(i.name), type: String(i.type ?? 'any') }))

    // Step 2: rename A.inputs to avoid collision with B's input names
    // (and the targetPin we're about to remove from B but still must avoid).
    // Note: targetInputName is being removed from B, but we still avoid it to keep the
    // intermediate state coherent in case fusion is interrupted / re-run.
    const usedNames = new Set<string>(bInputs.map((i) => i.name))
    const renameMap: Record<string, string> = {}
    const renamedAInputs: ExprInput[] = aInputs.map((inp) => {
      let newName = inp.name
      let suffix = 1
      // Single loop handles both collisions: usedNames AND targetInputName conflict.
      // (Previous version had a guard `newName !== inp.name` that caused this loop to
      // never fire on the very first collision, leaving duplicate names in B.inputs.)
      while (usedNames.has(newName) || newName === targetInputName) {
        newName = inp.name + suffix
        suffix++
      }
      renameMap[inp.name] = newName
      usedNames.add(newName)
      return { name: newName, type: inp.type }
    })

    // Step 3: substitute identifiers in A.expr per renameMap (word-boundary regex).
    let renamedAExpr = aExpr
    for (const [oldName, newName] of Object.entries(renameMap)) {
      if (oldName === newName) continue
      renamedAExpr = renamedAExpr.replace(new RegExp('\\b' + escapeRe(oldName) + '\\b', 'g'), newName)
    }

    // Step 4: replace targetPin identifier in B.expr with parenthesized A.expr.
    const newBExpr = bExpr.replace(
      new RegExp('\\b' + escapeRe(targetInputName) + '\\b', 'g'),
      '(' + renamedAExpr + ')',
    )

    // Step 5: B.inputs gets targetPin removed + A.inputs (renamed) appended.
    const newBInputs = bInputs.filter((i) => i.name !== targetInputName).concat(renamedAInputs)
    b.config = b.config ?? {}
    b.config.expr = newBExpr
    b.config.inputs = newBInputs
    // Drop literal that fed the now-removed targetPin (if any).
    if (b.config.literal && typeof b.config.literal === 'object') {
      delete b.config.literal[targetInputName]
    }
    // Carry A's per-input literals to B under the renamed keys. A's node (and its
    // config.literal) is deleted in step 7, so an unconnected A input holding a literal
    // would otherwise lose its value — the fused expr's renamed pin then evaluates empty,
    // silently changing the result. Literals on pins that get rewired below are ignored at
    // runtime, so copying unconditionally per renameMap is safe.
    const aLiteral = a.config?.literal as Record<string, unknown> | undefined
    if (aLiteral && typeof aLiteral === 'object') {
      b.config.literal = b.config.literal ?? {}
      for (const [oldName, newName] of Object.entries(renameMap)) {
        if (oldName in aLiteral) b.config.literal[newName] = aLiteral[oldName]
      }
    }

    // Helper: derive edge kind from (srcNode.kind, srcPin) since there is no edge.kind field.
    const isDataEdge = (e: any): boolean => {
      const [src, srcPin] = String(e.from ?? '').split('.')
      const srcNode = g.nodes.find((n: any) => n.id === src)
      return srcNode ? edgeKind(srcNode.kind, srcPin) === 'data' : false
    }

    // Step 6: reroute A's incoming data edges → B using renamed pin names.
    for (const e of g.edges) {
      if (!isDataEdge(e)) continue
      if (!e.to.startsWith(a.id + '.')) continue
      const [, pin] = e.to.split('.')
      const newPin = renameMap[pin] ?? pin
      e.to = `${b.id}.${newPin}`
    }

    // Step 7: remove the A→B edge then remove A node.
    // Order matters: isDataEdge looks up source node in g.nodes; remove edges first so
    // the lookup still resolves A's kind (else outgoing A edges would survive the filter).
    g.edges = g.edges.filter((e: any) => {
      if (!isDataEdge(e)) return true
      // drop the specific A.value → B.targetInputName edge AND any other A→B edges (defensive).
      return !(e.from.startsWith(a.id + '.') && e.to.startsWith(b.id + '.'))
    })
    g.nodes = g.nodes.filter((n: any) => n.id !== a.id)

    deps.syncFlowFromDraft()
    return true
  }

  return { fuse }
}

function escapeRe(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
