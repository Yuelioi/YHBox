<!--
ConfirmDialog 通用确认对话框（基于 NuxtUI UModal）。
- 支持纯确认 / 带 input 输入两种模式
- App.vue 全局挂一个实例，通过 useConfirm() Promise API 触发
- 任何 view / composable 都可调 confirm({...}) 弹此对话框
-->
<template>
  <UModal :open="open" @update:open="onUpdateOpen" :ui="{ content: 'sm:max-w-[480px]' }">
    <template #content>
      <UCard>
        <template #header>
          <h3 class="text-sm font-medium">{{ title }}</h3>
        </template>

        <div class="space-y-3">
          <p v-if="description" class="text-xs text-toned leading-relaxed whitespace-pre-line">{{ description }}</p>
          <div v-if="hasInput" class="space-y-1.5">
            <label v-if="inputLabel" class="text-xs text-toned">{{ inputLabel }}</label>
            <UInput
              v-model="inputValue"
              :placeholder="inputPlaceholder"
              size="sm"
              autofocus
              @keyup.enter="onConfirm"
            />
          </div>
        </div>

        <template #footer>
          <div class="flex items-center justify-end gap-2">
            <UButton size="sm" variant="ghost" color="neutral" @click="onCancel">{{ cancelTextResolved }}</UButton>
            <UButton size="sm" :color="colorResolved" @click="onConfirm">{{ confirmTextResolved }}</UButton>
          </div>
        </template>
      </UCard>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  open: boolean
  title: string
  description?: string
  confirmText?: string
  cancelText?: string
  color?: 'primary' | 'error' | 'warning' | 'neutral'
  inputDefault?: string
  inputLabel?: string
  inputPlaceholder?: string
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  resolve: [result: boolean | string]
}>()

const hasInput = computed(() => props.inputDefault !== undefined)
const inputValue = ref(props.inputDefault ?? '')

watch(() => props.open, (v) => {
  if (v) inputValue.value = props.inputDefault ?? ''
})

const confirmTextResolved = computed(() => props.confirmText ?? '确定')
const cancelTextResolved = computed(() => props.cancelText ?? '取消')
const colorResolved = computed(() => props.color ?? 'primary')

function onConfirm() {
  emit('resolve', hasInput.value ? inputValue.value : true)
  emit('update:open', false)
}
function onCancel() {
  emit('resolve', hasInput.value ? '' : false)
  emit('update:open', false)
}
function onUpdateOpen(v: boolean) {
  if (!v) {
    emit('resolve', hasInput.value ? '' : false)
  }
  emit('update:open', v)
}
</script>
