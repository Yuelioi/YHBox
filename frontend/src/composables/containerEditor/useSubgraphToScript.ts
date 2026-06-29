// 子图一键转脚本 — 预览 modal 状态 + 两入口(节点右键 / 子图属性面板)共用的转换/插入动作.
// 转换器本体是纯函数 lib/subgraphToScript.ts, 这里只做 store 取数和落点.
import { ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { useNodeRegistryStore } from '@/stores/nodeRegistry'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { subgraphToScript, type SubgraphLike, type UnsupportedItem } from '@/lib/subgraphToScript'
import type { Container, GraphNode } from '@/lib/backend'

type ToastApi = { add: (opts: { title: string; description?: string; color?: string }) => void }

interface AddNodeOpts {
  kind: string
  pos: { x: number; y: number }
  config?: Record<string, unknown>
}

interface UseSubgraphToScriptOpts {
  draft: Ref<Container | null>
  addNode: (o: AddNodeOpts) => GraphNode | null
  toast: ToastApi
}

export interface ToScriptState {
  open: boolean
  sgLabel: string
  code: string
  unsupported: UnsupportedItem[]
  insertPos: { x: number; y: number } | null
}

export function useSubgraphToScript(opts: UseSubgraphToScriptOpts) {
  const { t } = useI18n()
  const registry = useNodeRegistryStore()
  const editorStore = useContainerEditorStore()
  const state = ref<ToScriptState>({ open: false, sgLabel: '', code: '', unsupported: [], insertPos: null })

  // 子图完整数据(含 graph)活在全局池 (2026-06-12 全局化) — 容器只引用不拥有。
  const subgraphs = (): SubgraphLike[] => editorStore.subgraphList as unknown as SubgraphLike[]

  function convert(sg: SubgraphLike, insertPos: { x: number; y: number } | null) {
    const subgraphsById = new Map(subgraphs().map((s) => [s.id, s]))
    const res = subgraphToScript(sg, {
      specFor: getSpec,
      bindable: new Set(registry.scriptBindableKinds),
      subgraphsById,
    })
    state.value = res.ok
      ? { open: true, sgLabel: sg.label, code: res.code, unsupported: [], insertPos }
      : { open: true, sgLabel: sg.label, code: '', unsupported: res.unsupported, insertPos: null }
  }

  // 节点右键入口: callee 从全局池找, 插入位 = 被转节点旁。
  function convertFromNode(node: GraphNode) {
    const sgID = (node.config as Record<string, unknown> | undefined)?.SubgraphID as string | undefined
    const sg = subgraphs().find((s) => s.id === sgID)
    if (!sg) {
      opts.toast.add({ title: t('toast.subgraph_not_set'), color: 'warning' })
      return
    }
    convert(sg, { x: (node.x ?? 0) + 80, y: (node.y ?? 0) + 80 })
  }

  function insert() {
    if (!state.value.code) return
    const defaults = (getSpec('Script')?.defaults?.literal ?? {}) as Record<string, unknown>
    opts.addNode({
      kind: 'Script',
      pos: state.value.insertPos ?? { x: 200 + Math.random() * 100, y: 200 + Math.random() * 100 },
      config: { literal: { ...defaults, Code: state.value.code } },
    })
    state.value.open = false
    // 反馈 = 画布上出现新节点, 不弹 toast (ui.md 反馈规范)。
  }

  return { state, convert, convertFromNode, insert }
}
