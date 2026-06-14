// 节点插入落点 (flow 坐标) 的统一来源 — 编辑器里「新节点放哪」就这一个文件。
// 两类落点:
//   1. 视口中心  viewportCenter / viewportCenterForNode
//      用于: 库插入 (子图·clip) / NodeExplorer picker (无鼠标位时) / 录制产物 / 变量+ / snippet 单击
//   2. 指针位置  screenPointToFlow
//      用于: 画布拖放落点 / snippet 快捷键 (最后鼠标位) / Tab picker (唤起那刻鼠标位) / pin 拖到空白弹菜单
//
// 为什么视口中心不是 window 中心 (老 onInsertIncVar 那套): vue-flow 画布被左 rail / sidebar /
// toolbar 挤占, window.innerWidth/2 ≠ 画布可视中心。取 vue-flow 容器 boundingRect 的中心 →
// screenToFlowCoordinate 反算, 才是用户真正看到的那块画布的正中。
//
// useVueFlow() 在编辑器内任意 composable 调到的是同一 flow 实例 (注入共享), 所以这里拿的
// vueFlowRef / screenToFlowCoordinate 跟 view 那份一致。

import { useVueFlow } from '@vue-flow/core'

// 节点视觉居中补偿: vue-flow 的 node.x/y 是左上角。落在中心时节点整体偏右下, 减半个标准
// 身位让节点中心对准落点。标准节点 ~240×90 (ContainerFlowNode)。
// —— 只对「视口中心」类用; 「指针位置」类不减 (指针 = 你抓的那个点 = 节点左上角, 减了反而错位)。
const NODE_HALF_W = 120
const NODE_HALF_H = 45

export function useInsertPoint() {
  const { vueFlowRef, screenToFlowCoordinate } = useVueFlow()

  // 视口中心 (flow 坐标). 容器未挂载时退化 (0,0)。
  function viewportCenter(): { x: number; y: number } {
    const el = vueFlowRef.value
    if (!el) return { x: 0, y: 0 }
    const r = el.getBoundingClientRect()
    return screenToFlowCoordinate({ x: r.left + r.width / 2, y: r.top + r.height / 2 })
  }

  // 在视口中心放节点时该用的左上角坐标 (减半身位 → 节点视觉居中)。
  // 「插入节点到当前视图正中」的标准接口。
  function viewportCenterForNode(): { x: number; y: number } {
    const c = viewportCenter()
    return { x: c.x - NODE_HALF_W, y: c.y - NODE_HALF_H }
  }

  // 屏幕/客户区坐标 → flow 坐标。「插到指针位置」的标准接口。
  // 不做居中补偿: 指针落点即节点左上角。
  function screenPointToFlow(p: { x: number; y: number }): { x: number; y: number } {
    return screenToFlowCoordinate(p)
  }

  return { viewportCenter, viewportCenterForNode, screenPointToFlow }
}
