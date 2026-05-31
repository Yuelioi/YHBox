<template>
  <div class="flex items-center gap-1">
    <UInput
      :model-value="modelValue"
      :placeholder="capturing ? t('hotkeyInput.press_key') : t('hotkeyInput.key_example')"
      size="sm"
      :class="{ 'ring-1 ring-primary': capturing }"
      readonly
      class="flex-1 cursor-pointer"
      @focus="startCapture"
      @blur="stopCapture"
    />
    <UButton
      size="xs"
      :variant="capturing ? 'solid' : 'soft'"
      :color="capturing ? 'primary' : 'neutral'"
      :icon="capturing ? 'i-tabler-circle-dot' : 'i-tabler-keyboard'"
      :title="capturing ? t('hotkeyInput.click_to_cancel') : t('hotkeyInput.click_to_record')"
      @click="capturing ? stopCapture() : inputRef?.inputRef?.focus()"
    />
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const capturing = ref(false)
const inputRef = ref<any>(null)

// 把 Web KeyboardEvent.key / code 映射到我们 vk 字符串。
// vk 在 backend / pkg/input 里需匹配 vkName 表（字母 / 数字 / Fn / 特殊键）。
function keyToVK(e: KeyboardEvent): string {
  // 字母直接大写
  if (/^[a-zA-Z]$/.test(e.key)) return e.key.toUpperCase()
  // 数字
  if (/^[0-9]$/.test(e.key)) return e.key
  // 特殊键
  const map: Record<string, string> = {
    ' ': 'Space',
    Enter: 'Enter',
    Tab: 'Tab',
    Escape: 'Esc',
    Backspace: 'Back',
    Delete: 'Del',
    ArrowUp: 'Up',
    ArrowDown: 'Down',
    ArrowLeft: 'Left',
    ArrowRight: 'Right',
    Shift: 'Shift',
    Control: 'Ctrl',
    Alt: 'Alt',
  }
  if (map[e.key]) return map[e.key]
  // F1..F12
  if (/^F([1-9]|1[0-2])$/.test(e.key)) return e.key
  // fallback: 直接用 key
  return e.key
}

function onKeyDown(e: KeyboardEvent) {
  if (!capturing.value) return
  // 忽略只按修饰键（用户可能正按 Shift 准备按字母）
  if (e.key === 'Shift' || e.key === 'Control' || e.key === 'Alt' || e.key === 'Meta') return
  e.preventDefault()
  e.stopPropagation()
  emit('update:modelValue', keyToVK(e))
  stopCapture()
}

function startCapture() {
  capturing.value = true
  window.addEventListener('keydown', onKeyDown, { capture: true })
}

function stopCapture() {
  capturing.value = false
  window.removeEventListener('keydown', onKeyDown, { capture: true })
}

onUnmounted(stopCapture)
</script>
