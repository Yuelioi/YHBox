// 编辑器内部跨组件 signal bus. 替原 window.dispatchEvent('expr-fuse', ...) 全局总线.
// Pinia store 让请求 source/target 都明确 + Vue devtools 可见 + 无全局 listener
// 泄漏风险. backlog C10.
//
// 当前用例:
// - NodeInspector "合并到下游 Expr" 按钮 → requestExprFusion(...)
// - ContainerEditorView watch pendingExprFusion → 调 useExprFusion.fuse() → clear

import { defineStore } from 'pinia'

export interface ExprFusionRequest {
  sourceID: string
  targetID: string
  targetPin: string
}

export const useEditorBusStore = defineStore('editorBus', {
  state: () => ({
    pendingExprFusion: null as ExprFusionRequest | null,
  }),
  actions: {
    requestExprFusion(req: ExprFusionRequest) {
      this.pendingExprFusion = req
    },
    clearExprFusion() {
      this.pendingExprFusion = null
    },
  },
})
