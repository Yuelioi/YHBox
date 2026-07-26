<!-- 空状态: icon 徽框 (raised) + 标题 + 可选说明 + action 槽 (CTA)。文字走 props (消费方传 i18n)。 -->
<template>
  <div
    class="rounded-xl border border-dashed border-default bg-default/40 px-6 py-12 text-center"
    :class="{ 'm-3': inset }"
  >
    <div class="mx-auto mb-3 flex size-12 items-center justify-center rounded-xl raised-surface">
      <UIcon :name="icon" class="size-6 text-muted" />
    </div>
    <p class="text-sm font-medium text-highlighted">{{ title }}</p>
    <p v-if="description" class="mx-auto mt-1 max-w-xs text-xs text-muted">{{ description }}</p>
    <div v-if="$slots.action" class="mt-4 flex justify-center">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
const { inset = false } = defineProps<{
  icon: string
  title: string
  description?: string
  /** Reserve a standard gutter when rendered inside an edge-to-edge list surface. */
  inset?: boolean
}>()
</script>
