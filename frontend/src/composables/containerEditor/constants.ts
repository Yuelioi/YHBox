// Container Editor — shared layout / interaction constants.
//
// 集中 magic number, 让 view 跟 composables 都引一份. backlog C2.

/** Pin auto-connect threshold (flow-coord px). 拖 pin 出来松手, 跟最近 pin 距离小于此则自动连. */
export const AUTO_CONNECT_THRESHOLD_FLOW_PX = 30

/** Snap 容差 (flow-coord px). 拖节点跟另一节点 anchor Y/X 差小于此则吸附. */
export const SNAP_EPSILON = 4

/** 节点 exec-in handle 相对 node.y 的 Y 偏移. 所有 kind 用同 ContainerFlowNode 布局, 固定 50. */
export const SNAP_ANCHOR_Y_OFFSET = 50

/** 节点视觉宽 (px). centerOnNode 用. */
export const NODE_VISUAL_WIDTH = 200

/** 节点视觉高 (px). centerOnNode 用. */
export const NODE_VISUAL_HEIGHT = 100

/**
 * 把 viewport 居中到节点中心 (而非 node.x/y, 那是 top-left).
 * setCenter 是 vue-flow 的 viewport API.
 */
export function centerOnNode(
  setCenter: (x: number, y: number, opts?: { duration?: number; zoom?: number }) => void,
  node: { x: number; y: number },
  opts: { duration?: number; zoom?: number } = { duration: 300, zoom: 1 },
) {
  setCenter(node.x + NODE_VISUAL_WIDTH / 2, node.y + NODE_VISUAL_HEIGHT / 2, opts)
}
