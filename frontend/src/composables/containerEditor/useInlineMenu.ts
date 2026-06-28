// InlineContextMenu (右键画布空白 / 拖 pin 到空白 → 添加节点) 状态机.
//
// 状态机:
//   pin-drag start → onVfConnectStart 记 connectionStart
//   pin-drag end (空) → onVfConnectEnd 打开 inlineMenu (setTimeout 让 @connect 先 fire)
//   pin-drag end (成) → onConnect 调 markConnectSuccess, onVfConnectEnd 看到 flag 退出
//   右键空白 → onCanvasContextMenu / 双击空白 → onPaneDoubleClick
//   menu pick → onInlineMenuPick 加节点 + 自动 wire 回 source pin (有则)

import { ref, type Ref } from 'vue'
import type { Container, Graph, GraphNode, GraphEdge } from '@/lib/backend'
import { dataInTypeFor, dataOutTypeFor, getSpec } from '@/components/containers/nodeRegistry/registry'
import { isCompatibleType, type VarType } from '@/lib/variableRef'
import { newNodeID } from './ids'
import { useInsertPoint } from './useInsertPoint'
import type { PinContext as InlinePinContext } from '@/components/containers/InlineContextMenu.vue'

interface InlineMenuState {
  open: boolean
  position: { x: number; y: number }  // viewport (screen) coords
  flowPos: { x: number; y: number }   // flow canvas coords
  pinContext?: InlinePinContext
  sourcePin?: { nodeID: string; pinName: string; side: 'input' | 'output'; isExec?: boolean }
}

interface UseInlineMenuOpts {
  activeGraph: Ref<Graph | null>
  applyDraftMutation: (mutator: (draft: Container) => void) => void
  syncFlowFromDraft: () => void
}

export function useInlineMenu(opts: UseInlineMenuOpts) {
  const { activeGraph, applyDraftMutation, syncFlowFromDraft } = opts
  const { screenPointToFlow } = useInsertPoint()

  const inlineMenu = ref<InlineMenuState>({
    open: false,
    position: { x: 0, y: 0 },
    flowPos: { x: 0, y: 0 },
  })

  // 当前拖的 pin (drag-from-pin 起点)
  const connectionStart = ref<{
    nodeId: string
    handleId: string
    handleType: 'source' | 'target'
  } | null>(null)
  // onConnect 成功置 true, onVfConnectEnd 退出避免开 menu
  const connectMadeThisGesture = ref(false)

  function isValidVueFlowConnection(conn: {
    source: string
    target: string
    sourceHandle?: string | null
    targetHandle?: string | null
  }): boolean {
    if (!conn.sourceHandle || !conn.targetHandle) return false
    const srcNode = activeGraph.value?.nodes?.find((n) => n.id === conn.source)
    const tgtNode = activeGraph.value?.nodes?.find((n) => n.id === conn.target)
    if (!srcNode || !tgtNode) return true

    const srcOutType = dataOutTypeFor(srcNode.kind, conn.sourceHandle)
    const tgtInType = dataInTypeFor(tgtNode.kind, conn.targetHandle, tgtNode.config as Record<string, unknown>)

    // 非 data pin (exec 或 unknown kind) → 放行
    if (!srcOutType || !tgtInType) return true

    return isCompatibleType(srcOutType as VarType, tgtInType as VarType)
  }

  function onVfConnectStart(params: {
    event?: MouseEvent
    nodeId?: string
    handleId: string | null
    handleType?: 'source' | 'target'
  }) {
    // 每次新拖拽起手先清 flag: 成功连线那次 markConnectSuccess 已清空 connectionStart,
    // onVfConnectEnd 走早退分支没机会重置 connectMadeThisGesture, 上一手的 true 会残留到这手 —
    // 不清的话「连完线紧接着拖出口」会被误判成功而吞掉菜单 (有概率拖不出来的真因)。
    connectMadeThisGesture.value = false
    if (params.nodeId && params.handleId && params.handleType) {
      connectionStart.value = {
        nodeId: params.nodeId,
        handleId: params.handleId,
        handleType: params.handleType,
      }
    }
  }

  function onVfConnectEnd(event?: MouseEvent) {
    if (!connectionStart.value) return

    const clientX = event?.clientX ?? 0
    const clientY = event?.clientY ?? 0

    const start = connectionStart.value
    connectionStart.value = null

    // 拖到空白 → 开 menu. setTimeout 让 vue-flow @connect 同步先 fire;
    // 若成功 markConnectSuccess 已置 flag, 这里看到 flag 退出.
    const startCopy = { ...start }
    setTimeout(() => {
      if (connectMadeThisGesture.value) {
        connectMadeThisGesture.value = false
        return
      }

      const node = activeGraph.value?.nodes?.find((n) => n.id === startCopy.nodeId)
      if (!node) return

      const pinType =
        startCopy.handleType === 'source'
          ? dataOutTypeFor(node.kind, startCopy.handleId)
          : dataInTypeFor(node.kind, startCopy.handleId, node.config as Record<string, unknown>)

      const isExec = !pinType
      const flowPos = screenPointToFlow({ x: clientX, y: clientY })

      inlineMenu.value = {
        open: true,
        position: { x: clientX, y: clientY },
        flowPos,
        // Exec pin 不过滤类型, header "添加节点"; data pin 按 type 过滤, header "⊕ 接受 <type>"
        pinContext: isExec
          ? undefined
          : {
              pinType: pinType as VarType,
              side: startCopy.handleType === 'source' ? 'output' : 'input',
            },
        sourcePin: {
          nodeID: startCopy.nodeId,
          pinName: startCopy.handleId,
          side: startCopy.handleType === 'source' ? 'output' : 'input',
          isExec,
        },
      }
    }, 0)
  }

  function openInlineMenuAt(clientX: number, clientY: number) {
    const flowPos = screenPointToFlow({ x: clientX, y: clientY })
    inlineMenu.value = {
      open: true,
      position: { x: clientX, y: clientY },
      flowPos,
      pinContext: undefined,
      sourcePin: undefined,
    }
  }

  function onCanvasContextMenu(e: MouseEvent) {
    e.preventDefault()
    openInlineMenuAt(e.clientX, e.clientY)
  }

  function onPaneDoubleClick(e: MouseEvent) {
    openInlineMenuAt(e.clientX, e.clientY)
  }

  function onInlineMenuPick(kind: string) {
    const ctx = inlineMenu.value
    applyDraftMutation((_d) => {
      const g = activeGraph.value
      if (!g) return

      const newID = newNodeID(kind)
      const node: GraphNode = {
        id: newID,
        kind,
        x: ctx.flowPos.x,
        y: ctx.flowPos.y,
        config: { ...getSpec(kind)?.defaults },
        createdAt: new Date().toISOString(),
      } as GraphNode
      g.nodes.push(node)

      // pin-context: 拖 pin 后自动 wire 回 source
      if (ctx.sourcePin) {
        const spec = getSpec(kind)
        if (spec) {
          if (ctx.sourcePin.isExec) {
            // Exec-out drag → source exec-out → newNode 第一个 exec-in (通常 "in")
            if (ctx.sourcePin.side === 'output') {
              const firstExecIn = spec.execIn?.[0]
              if (firstExecIn) {
                ;(g.edges as GraphEdge[]).push({
                  from: `${ctx.sourcePin.nodeID}.${ctx.sourcePin.pinName}`,
                  to: `${newID}.${firstExecIn}`,
                } as GraphEdge)
              }
            } else {
              // Exec-in drag → newNode 第一个 exec-out → source exec-in
              const firstExecOut = spec.execOut?.[0] ?? 'out'
              ;(g.edges as GraphEdge[]).push({
                from: `${newID}.${firstExecOut}`,
                to: `${ctx.sourcePin.nodeID}.${ctx.sourcePin.pinName}`,
              } as GraphEdge)
            }
          } else if (ctx.pinContext) {
            if (ctx.pinContext.side === 'output') {
              // 拖 output pin → source.pin → newNode 第一个 compatible dataIn
              const candidates = Object.entries(spec.dataIn ?? {}).filter(([, t]) =>
                isCompatibleType(ctx.pinContext!.pinType, t as VarType),
              )
              if (candidates.length > 0) {
                ;(g.edges as GraphEdge[]).push({
                  from: `${ctx.sourcePin.nodeID}.${ctx.sourcePin.pinName}`,
                  to: `${newID}.${candidates[0][0]}`,
                } as GraphEdge)
              }
            } else {
              // 拖 input pin → newNode 第一个 compatible dataOut → source.pin
              const candidates = Object.entries(spec.dataOut ?? {}).filter(([, t]) =>
                isCompatibleType(t as VarType, ctx.pinContext!.pinType),
              )
              if (candidates.length > 0) {
                ;(g.edges as GraphEdge[]).push({
                  from: `${newID}.${candidates[0][0]}`,
                  to: `${ctx.sourcePin.nodeID}.${ctx.sourcePin.pinName}`,
                } as GraphEdge)
              }
            }
          }
        }
      }
    })
    syncFlowFromDraft()
  }

  // view 的 onConnect (wrap _onConnectBase) 调这个 — 告诉 onVfConnectEnd 别开 menu.
  function markConnectSuccess() {
    connectMadeThisGesture.value = true
    connectionStart.value = null
  }

  return {
    inlineMenu,
    isValidVueFlowConnection,
    onVfConnectStart,
    onVfConnectEnd,
    onCanvasContextMenu,
    openInlineMenuAt,
    onPaneDoubleClick,
    onInlineMenuPick,
    markConnectSuccess,
  }
}
