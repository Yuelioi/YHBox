// frontend/src/stores/nodeRegistry.ts
// FE 启动调一次, 缓存所有 Spec + TypeSpec.
// 类型 RPC 自动同步 — 后端加节点 / 改 Spec → FE 重启自动拿到, 0 手抄.
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { NodeService } from '@bindings/yotta/internal/node'
import type { Spec, TypeSpec } from '@bindings/yotta/internal/node'

export const useNodeRegistryStore = defineStore('nodeRegistry', () => {
  const specs = ref(new Map<string, Spec>())
  const types = ref(new Map<string, TypeSpec>())
  const errorCodes = ref<string[]>([])
  const loaded = ref(false)

  async function loadAll() {
    const [specList, typeList, codeList] = await Promise.all([
      NodeService.GetAllNodeSpecs(),
      NodeService.GetAllTypes(),
      NodeService.GetErrorCodes(),
    ])
    specs.value = new Map(specList.map((s: Spec) => [s.kind, s]))
    types.value = new Map(typeList.map((t: TypeSpec) => [t.tag, t]))
    errorCodes.value = codeList as string[]
    loaded.value = true
  }

  function getSpec(kind: string): Spec | undefined {
    return specs.value.get(kind)
  }

  function getType(tag: string): TypeSpec | undefined {
    return types.value.get(tag)
  }

  function pinColor(typeTag: string): string {
    return types.value.get(typeTag)?.color || '#888'
  }

  return { specs, types, errorCodes, loaded, loadAll, getSpec, getType, pinColor }
})
