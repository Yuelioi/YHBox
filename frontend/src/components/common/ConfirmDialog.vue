<!--
ConfirmDialog 通用确认对话框（基于 NuxtUI UModal）。
- 支持纯确认 / 带 input 输入两种模式
- App.vue 全局挂一个实例，通过 useConfirm() Promise API 触发
- 任何 view / composable 都可调 confirm({...}) 弹此对话框
-->
<template>
  <BaseModal :open="open" :title="title" size="md" :show-close="false" @update:open="onUpdateOpen">
    <div data-testid="confirm-dialog" class="space-y-3">
      <p v-if="description" class="text-xs text-toned leading-relaxed whitespace-pre-line">
        {{ description }}
      </p>
      <div v-if="hasInput" class="space-y-1.5">
        <label v-if="inputLabel" class="text-xs text-toned">{{ inputLabel }}</label>
        <UInput
          ref="inputRef"
          v-model="inputValue"
          :placeholder="inputPlaceholder"
          size="sm"
          @keyup.enter="onConfirm"
        />
      </div>
    </div>

    <template #footer>
      <UButton
        data-testid="confirm-cancel"
        size="sm"
        variant="ghost"
        color="neutral"
        @click="onCancel"
        >{{ cancelTextResolved }}</UButton
      >
      <UButton data-testid="confirm-accept" size="sm" :color="colorResolved" @click="onConfirm">{{
        confirmTextResolved
      }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'
import BaseModal from '@/components/common/BaseModal.vue'

const { t } = useI18n()

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

watch(
  () => props.open,
  (v) => {
    if (v) inputValue.value = props.inputDefault ?? ''
  },
)

// 打开即聚焦并全选默认值 — 直接打字替换, 回车接受。纯确认框无输入, 用
// open && hasInput 守门, 不让空抢焦点跑 12 帧重试 + DEV 告警。
const inputRef = ref<any>(null)
const openWithInput = computed(() => props.open && hasInput.value)
useAutoFocusOnOpen(openWithInput, inputRef, { selectAll: true })

const confirmTextResolved = computed(() => props.confirmText ?? t('common.confirm'))
const cancelTextResolved = computed(() => props.cancelText ?? t('common.cancel'))
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
