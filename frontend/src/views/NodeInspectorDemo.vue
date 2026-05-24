<!--
NodeInspectorDemo — Phase 0 用户实测页. 临时, Phase 5 删.
左侧选 mock 节点 → 中间 inspector + canvas 同步渲染.
-->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useNodeRegistryStore } from '@/stores/nodeRegistry'
import NodeInspector from '@/components/nodeInspector/NodeInspector.vue'
import NodeCard from '@/components/canvas/NodeCard.vue'

const registry = useNodeRegistryStore()
const selectedKind = ref<string>('Mock_Math')
const configs = ref<Record<string, Record<string, any>>>({
  Mock_Math: {},
  Mock_Form: {},
  Mock_Vision: {},
})

const inspectorRef = ref<InstanceType<typeof NodeInspector> | null>(null)

onMounted(async () => {
  if (!registry.loaded) await registry.loadAll()
})

const selectedSpec = computed(() => registry.getSpec(selectedKind.value))
const mockKinds = ['Mock_Math', 'Mock_Form', 'Mock_Vision']

function onSubmit() {
  const ok = inspectorRef.value?.validate()
  if (!ok) {
    alert('校验失败 — 见红字')
  } else {
    alert('提交 OK\n\n' + JSON.stringify(configs.value[selectedKind.value], null, 2))
  }
}
</script>

<template>
  <div class="p-6 max-w-7xl mx-auto">
    <h1 class="text-xl font-semibold mb-4">Node Inspector Demo (Phase 0 验证)</h1>

    <div v-if="!registry.loaded" class="text-sm text-toned">Loading registry...</div>

    <div v-else class="grid grid-cols-12 gap-6">
      <!-- Left: mock 节点选择 -->
      <div class="col-span-2 space-y-2">
        <div class="text-xs text-toned">Mock 节点</div>
        <button v-for="k in mockKinds" :key="k"
                class="w-full text-left px-3 py-2 rounded text-sm"
                :class="selectedKind === k ? 'bg-primary text-white' : 'bg-muted'"
                @click="selectedKind = k">
          {{ k }}
        </button>
      </div>

      <!-- Middle: Inspector -->
      <div class="col-span-6 border border-default rounded-md bg-elevated">
        <NodeInspector
          v-if="selectedSpec"
          ref="inspectorRef"
          :spec="selectedSpec"
          :config="configs[selectedKind]"
          :node-i-d="`demo-${selectedKind}`"
          @update:config="(v: any) => configs[selectedKind] = v"
        />
        <div class="border-t border-default p-3">
          <UButton size="sm" color="primary" @click="onSubmit">提交 (validate)</UButton>
        </div>
      </div>

      <!-- Right: Canvas NodeCard -->
      <div class="col-span-4 space-y-3">
        <div class="text-xs text-toned">Canvas 视觉</div>
        <NodeCard v-if="selectedSpec" :spec="selectedSpec" />

        <details class="mt-4 text-xs">
          <summary class="cursor-pointer">Current config JSON</summary>
          <pre class="mt-2 p-2 bg-muted rounded overflow-x-auto">{{ JSON.stringify(configs[selectedKind], null, 2) }}</pre>
        </details>
      </div>
    </div>
  </div>
</template>
