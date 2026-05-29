<!--
NodeCard — 节点 canvas 视觉. 多 exec 出口 + Data 字段 hover tooltip.
不做 nested pin visual; Data 字段走 hover 展开.
下游节点连 exec line 后会自动注入上游 Data 字段, 在 hover tooltip 显示.
-->
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Spec, DataField } from '@bindings/yhbox/internal/node'
import { useNodeRegistryStore } from '@/stores/nodeRegistry'

const { t, te } = useI18n()
const props = defineProps<{ spec: Spec }>()
const registry = useNodeRegistryStore()
const hoveredOut = ref<string | null>(null)

const execIns = computed(() => props.spec.inputs.filter(i => i.type === 'Exec'))
const dataIns = computed(() => props.spec.inputs.filter(i => i.type !== 'Exec' && !i.advanced))
const execOuts = computed(() => props.spec.outputs.filter(o => o.type === 'Exec'))
const dataOuts = computed(() => props.spec.outputs.filter(o => o.type !== 'Exec'))

function pinColor(typeTag: string): string {
  return registry.pinColor(typeTag)
}

// i18n lookup helpers — key 缺失 fallback raw name (e.g. dynamic Subgraph pin / 节点未注册).
function nodeLabel(): string {
  const k = `node.${props.spec.kind}.label`
  return te(k) ? t(k) : props.spec.kind
}
function inputLabel(name: string): string {
  const k = `node.${props.spec.kind}.input.${name}.label`
  return te(k) ? t(k) : name
}
function outputLabel(name: string): string {
  const k = `node.${props.spec.kind}.output.${name}.label`
  return te(k) ? t(k) : name
}
function dataFieldHint(outName: string, fieldName: string): string {
  const k = `node.${props.spec.kind}.output.${outName}.data.${fieldName}.hint`
  return te(k) ? t(k) : ''
}
</script>

<template>
  <div class="node-card border border-default rounded-md bg-elevated min-w-[200px] shadow-sm">
    <div class="px-3 py-2 border-b border-default bg-muted">
      <div class="text-sm font-medium">{{ nodeLabel() }}</div>
      <div class="text-xs text-toned">{{ spec.category }}</div>
    </div>

    <div class="px-3 py-2 flex justify-between">
      <!-- Inputs (left) -->
      <div class="space-y-1">
        <div v-for="i in execIns" :key="i.name" class="flex items-center gap-2">
          <div class="w-2.5 h-2.5 bg-white" />
          <span class="text-xs">{{ inputLabel(i.name) }}</span>
        </div>
        <div v-for="i in dataIns" :key="i.name" class="flex items-center gap-2">
          <div class="w-2.5 h-2.5 rounded-full" :style="{ background: pinColor(i.type) }" />
          <span class="text-xs">{{ inputLabel(i.name) }}</span>
        </div>
      </div>

      <!-- Outputs (right) -->
      <div class="space-y-1">
        <div v-for="o in execOuts" :key="o.name"
             class="flex items-center justify-end gap-2 relative"
             @mouseenter="hoveredOut = o.name"
             @mouseleave="hoveredOut = null">
          <span class="text-xs">{{ outputLabel(o.name) }}</span>
          <div class="w-2.5 h-2.5 bg-white" />
          <!-- hover tooltip: Data 字段列表 -->
          <div v-if="hoveredOut === o.name && o.data && o.data.length > 0"
               class="absolute left-full top-0 ml-2 z-50 bg-elevated border border-default rounded-md p-2 shadow-md min-w-[220px]">
            <div class="text-xs text-toned mb-1">Data fields:</div>
            <div v-for="d in (o.data as DataField[])" :key="d.name"
                 class="flex items-center gap-2 py-0.5">
              <div class="w-2 h-2 rounded-full" :style="{ background: pinColor(d.type) }" />
              <span class="text-xs font-mono">{{ d.name }}</span>
              <span class="text-xs text-toned">: {{ d.type }}</span>
              <span v-if="d.optional" class="text-xs text-toned italic">({{ t('common.optional') }})</span>
            </div>
            <div v-for="d in (o.data as DataField[]).filter(d => dataFieldHint(o.name, d.name))" :key="d.name + ':doc'"
                 class="text-xs text-toned mt-1 italic">
              {{ d.name }}: {{ dataFieldHint(o.name, d.name) }}
            </div>
          </div>
        </div>
        <div v-for="o in dataOuts" :key="o.name" class="flex items-center justify-end gap-2">
          <span class="text-xs">{{ outputLabel(o.name) }}</span>
          <div class="w-2.5 h-2.5 rounded-full" :style="{ background: pinColor(o.type) }" />
        </div>
      </div>
    </div>
  </div>
</template>
