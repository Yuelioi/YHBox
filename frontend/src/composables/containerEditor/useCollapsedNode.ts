// v4 §9.2 CollapsedNode composable.
//
// Status: SCAFFOLD (v4 Phase D.4-D7 minimal landing).
// - Wires up the operations interface (collapse / expand / upgrade) so view code can
//   call into a single place without scattering algorithm logic.
// - Real algorithm is NON-TRIVIAL — see TODOs below. Currently each op is a stub that
//   throws or no-ops with a console warn. v4 Phase D polish (post-MVP) fills these in.
// - CollapsedNode runtime works TODAY if users hand-author JSON: container.subgraphs[]
//   includes a Subgraph with isAnonymous:true and a CollapsedNode in main graph references
//   it via config.subgraphId. Backend validators catch misuse (D2+D3 shipped).
//
// Required real backend RPCs (not yet exposed): backend.containers.createSubgraph,
// backend.containers.deleteSubgraph, backend.containers.updateSubgraphMeta. Without
// them the operations can only mutate the in-memory draft (re-saved by onSave).
import type { Ref } from 'vue'

interface CollapseDeps {
  draft: Ref<any>
  activeGraph: Ref<any>
  syncFlowFromDraft: () => void
  refreshSubgraphStore?: () => Promise<void>
}

function warnOnce(msg: string) {
  // eslint-disable-next-line no-console
  console.warn('[useCollapsedNode]', msg)
}

export function useCollapsedNode(_deps: CollapseDeps) {
  /**
   * TODO (D4): Collapse selected node IDs into a CollapsedNode + isAnonymous Subgraph.
   * Algorithm (per spec §9.2):
   *   1. New isAnonymous Subgraph (uuid + isAnonymous=true + auto-layout for internal pos)
   *   2. Move selected nodes from current graph → new Subgraph.Graph.Nodes
   *   3. Insert SubgraphInput marker + SubgraphOutput markers (one per exit)
   *   4. Scan boundary edges:
   *      - external → internal (data + exec): create InputParam (data) / SubgraphInput edge (exec); rewire edge to CollapsedNode pin
   *      - internal → external: create OutputPin decl + SubgraphOutput marker; rewire to CollapsedNode out pin
   *   5. Insert CollapsedNode in parent graph pointing to new sg (subgraphId + label)
   *   6. Save: refreshSubgraphStore + persist draft
   * Edge cases: handle nested CollapsedNode in selection (forbid for v4 MVP),
   * preserve loopback edges (Loop body in selection), handle multiple disconnected components.
   */
  async function collapse(_selectedNodeIDs: string[]): Promise<void> {
    warnOnce('collapse: not yet implemented; manually author CollapsedNode JSON for now')
  }

  /**
   * TODO (D5): Expand reverse — move CollapsedNode internals back to parent graph,
   * delete the backing isAnonymous Subgraph. Reconnect boundary edges.
   */
  async function expand(_collapsedNodeID: string): Promise<void> {
    warnOnce('expand: not yet implemented')
  }

  /**
   * TODO (D7): Flip backing Subgraph.isAnonymous=false; change node Kind CollapsedNode → Subgraph.
   * After upgrade the subgraph is reusable and appears in palette / candidate dropdowns.
   */
  async function upgradeToSubgraph(_collapsedNodeID: string): Promise<void> {
    warnOnce('upgradeToSubgraph: not yet implemented')
  }

  return { collapse, expand, upgradeToSubgraph }
}
