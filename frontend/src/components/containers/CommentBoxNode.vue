<template>
  <!-- CommentBox — pure-UI grouping affordance. 无 exec, 无 data, 无 handle.
       Visual only: dashed-border tinted rectangle, 放在常规节点背后.
       Size + label + color 都从 node.config 读. -->
  <div
    class="comment-box rounded-md border-2 border-dashed p-2 select-none"
    :style="boxStyle"
  >
    <div
      class="text-xs font-semibold opacity-80 truncate"
      :style="{ color: color }"
    >
      {{ label }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  id: string
  data: { kind: string; config?: Record<string, any> }
  selected?: boolean
}>()

const label = computed<string>(() => props.data?.config?.label ?? '注释')
const color = computed<string>(() => props.data?.config?.color ?? '#fbbf24')
const width = computed<number>(() => Number(props.data?.config?.width ?? 200))
const height = computed<number>(() => Number(props.data?.config?.height ?? 150))

const boxStyle = computed(() => ({
  width: width.value + 'px',
  height: height.value + 'px',
  background: color.value + '20', // 12% alpha overlay
  borderColor: color.value,
  boxShadow: props.selected ? `0 0 0 2px ${color.value}` : 'none',
  pointerEvents: 'auto' as const,
}))
</script>

<style scoped>
.comment-box {
  position: relative;
}
</style>
