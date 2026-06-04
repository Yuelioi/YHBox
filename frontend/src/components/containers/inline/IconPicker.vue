<template>
  <!-- 共享图标选择器 — 数据源 visualRegistry.ICONS. modelValue = 完整 tabler 名 (i-tabler-xxx). -->
  <div class="flex flex-wrap gap-1.5">
    <button
      v-for="entry in ICONS"
      :key="entry.icon"
      type="button"
      class="icp-chip"
      :class="{ 'is-selected': modelValue === entry.icon }"
      @click="emit('update:modelValue', entry.icon)"
    >
      <UIcon :name="entry.icon" class="size-4" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ICONS } from '../visualRegistry'

defineProps<{
  modelValue: string | undefined | null
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()
</script>

<style scoped>
.icp-chip {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--ui-text-toned);
  cursor: pointer;
  transition: all 120ms ease;
}
.icp-chip:hover {
  background: rgba(255, 255, 255, 0.12);
  color: var(--ui-text-default);
}
.icp-chip.is-selected {
  background: var(--ui-primary, #6366f1);
  border-color: var(--ui-primary, #6366f1);
  color: white;
}
</style>
