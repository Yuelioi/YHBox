// useFlowInteraction 画布 drag / drop 交互。
// 支持两种 dataTransfer：
//   - application/x-yotta-node：NodePalette 节点拖入（kind 字符串）
//   - application/yotta-library-item：LibraryView 卡片拖入（copy-on-use 生成独立子图副本）
// 节点 drop 时若 cursor 命中某条 edge 且新节点是 1in/1out，自动断边重连（A→B 变 A→C→B）
import type { Ref, ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type Container, type Graph } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { PIN_SPECS } from '@/components/containers/pinSpec'
import { randID } from './ids'

const NODE_W = 220
const NODE_H = 90
const EDGE_HIT_THRESHOLD = 60 // canvas px

// 1in1out 判定: 只有 'in' / 'out' 一对默认 pin 的节点 (可安全断边插入)
function isSingleInOut(kind: string): boolean {
  const s = PIN_SPECS[kind]
  if (!s) return false
  return (
    s.execIn.length === 1 &&
    s.execIn[0] === 'in' &&
    s.execOut.length === 1 &&
    s.execOut[0] === 'out'
  )
}

// 点到线段距离 (canvas 坐标系)
function distToSegment(px: number, py: number, ax: number, ay: number, bx: number, by: number) {
  const dx = bx - ax
  const dy = by - ay
  const len2 = dx * dx + dy * dy
  if (len2 === 0) return Math.hypot(px - ax, py - ay)
  let t = ((px - ax) * dx + (py - ay) * dy) / len2
  t = Math.max(0, Math.min(1, t))
  return Math.hypot(px - (ax + t * dx), py - (ay + t * dy))
}

// 在当前 graph 里找离 (px, py) 最近的 edge, 距离 < threshold 才命中
function findEdgeNearPoint(graph: Graph, px: number, py: number): Graph['edges'][number] | null {
  if (!graph?.edges || !graph?.nodes) return null
  let best: Graph['edges'][number] | null = null
  let bestD = EDGE_HIT_THRESHOLD
  for (const e of graph.edges) {
    const srcID = e.from.split('.')[0]
    const tgtID = e.to.split('.')[0]
    const s = graph.nodes.find((n) => n.id === srcID)
    const t = graph.nodes.find((n) => n.id === tgtID)
    if (!s || !t) continue
    const ax = s.x + NODE_W / 2
    const ay = s.y + NODE_H / 2
    const bx = t.x + NODE_W / 2
    const by = t.y + NODE_H / 2
    const d = distToSegment(px, py, ax, ay, bx, by)
    if (d < bestD) {
      bestD = d
      best = e
    }
  }
  return best
}

export function useFlowInteraction(opts: {
  project: (pt: { x: number; y: number }) => { x: number; y: number }
  onAddNode: (kind: string, atX?: number, atY?: number) => Promise<string | null>
  draft?: Ref<Container | null>
  activeGraph?: ComputedRef<Graph | null>
  syncFlowFromDraft?: () => void
  refreshSubgraphStore?: () => Promise<void>
  toast?: { add: (o: Record<string, unknown>) => unknown }
}) {
  const { project, onAddNode, draft, activeGraph, syncFlowFromDraft, refreshSubgraphStore, toast } = opts
  const settingsStore = useSettingsStore()
  const { confirm } = useConfirm()
  const { t } = useI18n()

  function onCanvasDragOver(e: DragEvent) {
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
  }

  async function onCanvasDrop(e: DragEvent) {
    // 先检测库拖入（优先级高于节点拖入）
    const libData = e.dataTransfer?.getData('application/yotta-library-item')
    if (libData) {
      await handleLibraryDrop(e, libData)
      return
    }
    // 节点 palette 拖入
    const kind = e.dataTransfer?.getData('application/x-yotta-node')
    if (!kind) return
    const target = e.currentTarget as HTMLElement
    const rect = target.getBoundingClientRect()
    const px = e.clientX - rect.left
    const py = e.clientY - rect.top
    const pos = project({ x: px, y: py })

    // 节点是 1in/1out 且 cursor 命中某条 edge → 自动断边重连
    const hitEdge =
      activeGraph?.value && isSingleInOut(kind)
        ? findEdgeNearPoint(activeGraph.value, pos.x, pos.y)
        : null

    const newID = await onAddNode(kind, pos.x, pos.y)

    if (hitEdge && newID && activeGraph?.value) {
      const oldFrom = hitEdge.from
      const oldTo = hitEdge.to
      activeGraph.value.edges = activeGraph.value.edges.filter((x) => x !== hitEdge)
      activeGraph.value.edges.push({ from: oldFrom, to: `${newID}.in` })
      activeGraph.value.edges.push({ from: `${newID}.out`, to: oldTo })
      syncFlowFromDraft?.()
      toast?.add({
        title: t('flowInteraction.inserted_and_relinked'),
        color: 'success',
        icon: 'i-tabler-arrow-merge',
        duration: 3000,
      })
    }
  }

  async function handleLibraryDrop(e: DragEvent, libData: string) {
    if (!draft?.value || !activeGraph?.value) return
    try {
      const parsed = JSON.parse(libData)
      if (parsed.kind !== 'subgraph') {
        toast?.add({ title: t('flowInteraction.no_template_drop'), color: 'warning' })
        return
      }
      const target = e.currentTarget as HTMLElement
      const rect = target.getBoundingClientRect()
      const px = e.clientX - rect.left
      const py = e.clientY - rect.top
      const pos = project({ x: px, y: py })

      await backend.library.importToContainer(parsed.id, draft.value.id)
      const newSubgraphID: string = parsed.id
      const newNode = {
        id: randID('n-call'),
        kind: 'Subgraph',
        x: pos.x,
        y: pos.y,
        config: { SubgraphID: newSubgraphID },
        createdAt: new Date().toISOString(),
      }
      activeGraph.value.nodes.push(newNode)
      if (refreshSubgraphStore) await refreshSubgraphStore()
      if (syncFlowFromDraft) syncFlowFromDraft()
      toast?.add({ title: t('flowInteraction.subgraph_imported', { id: newSubgraphID }), color: 'success', icon: 'i-tabler-check' })
    } catch (err) {
      console.error('library drop failed', err)
      toast?.add({ title: t('toast.import_failed'), description: String(err), color: 'error' })
    }
  }

  // E.9: X3 hybrid — 库子图导入后 RecordingContext 与本机不一致时的处理
  // 三选一：同步所有 / 仅本容器 / 不改。v1 串两次 confirm 实现。
  async function handleImportSyncCalibration(sourceCounts: number, localCounts: number) {
    const yes1 = await confirm({
      title: t('recording.foreign_machine'),
      description: t('recording.counts_mismatch_desc', { src: sourceCounts, local: localCounts }),
      confirmText: t('recording.sync_all'),
      cancelText: t('recording.no_sync'),
      color: 'primary',
    })
    if (yes1 === true) {
      await backend.containers.syncLocalMouseCalibration(localCounts)
      toast?.add({ title: t('recording.sync_success'), color: 'success' })
      return
    }
    const yes2 = await confirm({
      title: t('recording.modify_single_title'),
      description: t('recording.modify_single_desc'),
      confirmText: t('recording.modify_single_only'),
      cancelText: t('recording.modify_keep_source'),
      color: 'primary',
    })
    if (yes2 === true && draft?.value) {
      // 改主图 MouseCalibration 节点 counts360
      for (const n of draft.value.graph.nodes) {
        if (n.kind === 'MouseCalibration') {
          n.config = { ...(n.config ?? {}), counts360: localCounts }
        }
      }
      syncFlowFromDraft?.()
      toast?.add({ title: t('flowInteraction.modified_local_mouse_calibration'), color: 'success' })
    }
  }

  return { onCanvasDragOver, onCanvasDrop }
}
