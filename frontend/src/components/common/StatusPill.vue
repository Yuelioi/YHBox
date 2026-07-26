<!-- 状态药丸: status 驱动配色 (字面表)。文字走 slot 或 label; dot=true 加前导圆点。 -->
<template>
  <span
    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold"
    :class="pillClass"
  >
    <span v-if="dot" class="size-1.5 rounded-full bg-current" />
    <slot>{{ label }}</slot>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { statusPillClass, type PillStatus } from './StatusPill.helpers'

const props = defineProps<{ status: PillStatus; label?: string; dot?: boolean }>()
const pillClass = computed(() => statusPillClass(props.status))
</script>
