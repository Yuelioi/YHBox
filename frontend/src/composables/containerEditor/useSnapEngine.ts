// Pin-aware snap engine (PS smart-guides style).
// 从 ContainerEditorView 抽离 — 见 backlog C1.
//
// 工作流:
//   drag 中 → onSnapNodeDrag 算引导线 (cyan Y / magenta X), 写 snapGuides
//   drag 停 → onSnapNodeDragStop 选最近 target, updateNode(vue-flow store) + applyDraftMutation(persistence)
//
// 用 event.node.position (vue-flow store 的 live 坐标) 而非 flowNodes.value:
// v-model store→model watcher shallow 监听 array ref + length, drag 中 element.position
// 突变不触发同步 → flowNodes[i].position 在 dragStop 时仍是拖动前坐标. 详见 scar
// 2026-05-28-vue-flow-store-vmodel-shallow-sync.md.

import { ref, type Ref } from 'vue'
import { useVueFlow, type NodeDragEvent } from '@vue-flow/core'
import type { Container, Graph } from '@/lib/backend'
import type { SidebarPrefs } from '@/composables/editor/useSidebarPrefs'
import type { SnapGuide } from '@/components/containers/SnapGuideOverlay.vue'
import { SNAP_ANCHOR_Y_OFFSET, SNAP_EPSILON } from './constants'

interface UseSnapEngineOpts {
  sidebarPrefs: Ref<SidebarPrefs>
  activeGraph: Ref<Graph | null>
  applyDraftMutation: (mutator: (draft: Container) => void) => void
}

export function useSnapEngine(opts: UseSnapEngineOpts) {
  const { sidebarPrefs, activeGraph, applyDraftMutation } = opts
  const { updateNode } = useVueFlow()

  const snapGuides = ref<SnapGuide[]>([])

  function onSnapNodeDrag(event: NodeDragEvent) {
    // wantSnap: toolbar toggle XOR Alt key — Alt inverts whichever the toggle is (PS/Figma pattern)
    const wantSnap = sidebarPrefs.value.snapEnabled !== event.event.altKey
    if (!wantSnap) {
      snapGuides.value = []
      return
    }

    const draggedID: string = event.node.id
    const yOff = SNAP_ANCHOR_Y_OFFSET

    // event.node.position is the live flow coordinate (updated by vue-flow during drag)
    const draggedX: number = event.node.position?.x ?? 0
    const draggedY: number = event.node.position?.y ?? 0
    const draggedAnchorY = draggedY + yOff

    const guides: SnapGuide[] = []
    const nodes = activeGraph.value?.nodes ?? []
    for (const other of nodes) {
      if (other.id === draggedID) continue
      const otherAnchorY = other.y + SNAP_ANCHOR_Y_OFFSET
      if (Math.abs(otherAnchorY - draggedAnchorY) <= SNAP_EPSILON) {
        // Horizontal cyan guide at otherAnchorY — coords stay in flow space;
        // SnapGuideOverlay applies the vue-flow viewport transform via <g :transform>.
        const lineFlowY = otherAnchorY
        const leftFlowX = Math.min(other.x, draggedX) - 30
        const rightFlowX = Math.max(other.x + 220, draggedX + 220) + 30
        guides.push({
          axis: 'y',
          x1: leftFlowX,
          y1: lineFlowY,
          x2: rightFlowX,
          y2: lineFlowY,
        })
      }
      // X-axis (vertical magenta guide) — left-edge alignment
      if (Math.abs(other.x - draggedX) <= SNAP_EPSILON) {
        const lineFlowX = other.x
        const topFlowY = Math.min(other.y, draggedY) - 30
        const botFlowY = Math.max(other.y + 80, draggedY + 80) + 30
        guides.push({
          axis: 'x',
          x1: lineFlowX,
          y1: topFlowY,
          x2: lineFlowX,
          y2: botFlowY,
        })
      }
    }
    snapGuides.value = guides
  }

  function onSnapNodeDragStop(event: NodeDragEvent) {
    snapGuides.value = []

    const wantSnap = sidebarPrefs.value.snapEnabled !== event.event.altKey
    if (!wantSnap) return

    const draggedID: string = event.node.id

    // event.node.position 是 vue-flow store 的 live 坐标 (跟 onSnapNodeDrag 同源).
    // 不能读 flowNodes.value: v-model:nodes 的 store→model watcher (vue-flow-core
    // pauseStore) shallow 监听 array ref + length, drag 中 element.position 突变
    // 不触发同步 → flowNodes[i].position 在 dragStop 时仍是拖动前坐标 → snap 比对全错.
    const yOff = SNAP_ANCHOR_Y_OFFSET
    const draggedX = event.node.position?.x ?? 0
    const draggedY = event.node.position?.y ?? 0
    const draggedAnchorY = draggedY + yOff

    let bestY: { delta: number; targetAnchorY: number } | null = null
    let bestX: { delta: number; targetX: number } | null = null

    for (const other of activeGraph.value?.nodes ?? []) {
      if (other.id === draggedID) continue
      const otherAnchorY = other.y + SNAP_ANCHOR_Y_OFFSET
      const dy = otherAnchorY - draggedAnchorY
      if (Math.abs(dy) <= SNAP_EPSILON) {
        if (!bestY || Math.abs(dy) < Math.abs(bestY.delta)) {
          bestY = { delta: dy, targetAnchorY: otherAnchorY }
        }
      }
      const dx = other.x - draggedX
      if (Math.abs(dx) <= SNAP_EPSILON) {
        if (!bestX || Math.abs(dx) < Math.abs(bestX.delta)) {
          bestX = { delta: dx, targetX: other.x }
        }
      }
    }

    if (bestY || bestX) {
      const finalX = bestX ? bestX.targetX : draggedX
      const finalY = bestY ? bestY.targetAnchorY - yOff : draggedY

      // 1. Tell vue-flow's internal node store about the snapped position.
      //    Position 突变本身不触发 v-model 同步 (见上); 这里直显 store 让 computedPosition
      //    watcher (vue-flow-core.mjs:9534) 立刻重算 → DOM transform 跳到 snap target.
      updateNode(draggedID, { position: { x: finalX, y: finalY } })

      // 2. Persist to draft (authoritative source-of-truth for save/reload).
      applyDraftMutation((d) => {
        const g = activeGraph.value
        if (!g) return
        const node = g.nodes.find((n) => n.id === draggedID)
        if (!node) return
        node.x = finalX
        node.y = finalY
      })
    }
  }

  return { snapGuides, onSnapNodeDrag, onSnapNodeDragStop }
}
