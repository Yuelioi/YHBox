<template>
  <!-- 共享色块选择器 — 数据源 visualRegistry.PALETTE (单一真源).
       valueMode='key' (默认) 存 palette key (CommentBox); 'hex' 存 hex (Snippet 旧数据). -->
  <div class="flex flex-wrap gap-1.5">
    <button
      v-for="key in PALETTE_KEYS"
      :key="key"
      type="button"
      class="cpp-chip"
      :class="{ 'is-selected': isSelected(key) }"
      :style="{ background: PALETTE[key].hex }"
      :title="t(PALETTE[key].labelKey)"
      @click="pick(key)"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { PALETTE, PALETTE_KEYS } from '../visualRegistry'

const { t } = useI18n()

const props = defineProps<{
  modelValue: string | undefined | null
  /** 'key' 存 palette key (默认); 'hex' 存 hex 值. */
  valueMode?: 'key' | 'hex'
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

function valOf(key: string): string {
  return props.valueMode === 'hex' ? PALETTE[key].hex : key
}
function isSelected(key: string): boolean {
  if (props.valueMode === 'hex') {
    return (props.modelValue ?? '').toLowerCase() === PALETTE[key].hex.toLowerCase()
  }
  return props.modelValue === key
}
function pick(key: string) {
  emit('update:modelValue', valOf(key))
}
</script>

<style scoped>
.cpp-chip {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 120ms ease;
}
.cpp-chip:hover {
  transform: scale(1.12);
}
.cpp-chip.is-selected {
  border-color: rgba(255, 255, 255, 0.85);
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.2);
}
</style>
