// useFlowInteraction 画布 drag / drop 交互。
// 支持两种 dataTransfer：
//   - application/x-yhbox-node：NodePalette 节点拖入（kind 字符串）
//   - application/yhfish-library-item：LibraryView 卡片拖入（copy-on-use 生成独立子图副本）
// 节点 drop 时若 cursor 命中某条 edge 且新节点是 1in/1out，自动断边重连（A→B 变 A→C→B）
import type { Ref, ComputedRef } from 'vue'
import { backend, type Container, type Graph } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { PIN_SPECS } from '@/components/containers/pinSpec'

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

  function onCanvasDragOver(e: DragEvent) {
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
  }

  async function onCanvasDrop(e: DragEvent) {
    // 先检测库拖入（优先级高于节点拖入）
    const libData = e.dataTransfer?.getData('application/yhfish-library-item')
    if (libData) {
      await handleLibraryDrop(e, libData)
      return
    }
    // 节点 palette 拖入
    const kind = e.dataTransfer?.getData('application/x-yhbox-node')
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
        title: '已插入节点并断边重连',
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
        toast?.add({ title: '暂不支持拖入库模板（请走容器内 template 管理）', color: 'warning' })
        return
      }
      const target = e.currentTarget as HTMLElement
      const rect = target.getBoundingClientRect()
      const px = e.clientX - rect.left
      const py = e.clientY - rect.top
      const pos = project({ x: px, y: py })

      // 走后端 copy-on-use（生成独立子图副本，1:1 自然成立）
      const newSg = (await backend.library.copyToContainer(parsed.id, draft.value.id)) as any
      // 在落点加 Subgraph 调用节点
      const newNode = {
        id: 'n-call-' + Math.random().toString(36).slice(2, 8),
        kind: 'Subgraph',
        x: pos.x,
        y: pos.y,
        config: { subgraphId: newSg.id },
        createdAt: new Date().toISOString(),
      }
      activeGraph.value.nodes.push(newNode)
      if (refreshSubgraphStore) await refreshSubgraphStore()
      if (syncFlowFromDraft) syncFlowFromDraft()
      toast?.add({ title: `已导入子图: ${newSg.label}`, color: 'success', icon: 'i-tabler-check' })

      // E.9 X3 hybrid：导入的子图 RecordingContext 与本机 mouseCounts360 不一致时弹双段对话框
      const sourceCounts: number = newSg.recordingContext?.mouseCounts360 ?? 0
      const localCounts: number =
        (settingsStore as any)?.data?.ui?.mouseCounts360 ??
        (settingsStore as any)?.settings?.ui?.mouseCounts360 ??
        0
      if (sourceCounts > 0 && localCounts > 0 && sourceCounts !== localCounts) {
        await handleImportSyncCalibration(sourceCounts, localCounts)
      }
    } catch (err) {
      console.error('library drop failed', err)
      toast?.add({ title: '导入失败', description: String(err), color: 'error' })
    }
  }

  // E.9: X3 hybrid — 库子图导入后 RecordingContext 与本机不一致时的处理
  // 三选一：同步所有 / 仅本容器 / 不改。v1 串两次 confirm 实现。
  async function handleImportSyncCalibration(sourceCounts: number, localCounts: number) {
    const yes1 = await confirm({
      title: '导入的子图来自另一台机器',
      description: `源 360° counts: ${sourceCounts}\n你的本机值: ${localCounts}\n\n是否把本机值同步到所有本地容器？（推荐）`,
      confirmText: '同步所有容器',
      cancelText: '不同步',
      color: 'primary',
    })
    if (yes1 === true) {
      await backend.containers.syncLocalMouseCalibration(localCounts)
      toast?.add({ title: '已同步所有容器', color: 'success' })
      return
    }
    const yes2 = await confirm({
      title: '仅修改本容器？',
      description: '仅把当前容器主图 MouseCalibration 改为本机值。其他容器保持不变。',
      confirmText: '仅本容器',
      cancelText: '不改（保持源值）',
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
      toast?.add({ title: '已修改本容器主图 MouseCalibration', color: 'success' })
    }
  }

  return { onCanvasDragOver, onCanvasDrop }
}
