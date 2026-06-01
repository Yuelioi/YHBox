<template>
  <div class="flex items-center gap-1">
    <!-- 平时是普通文本框 (手填 vk 名, 如 f9 / space); 录制时变只读 + 高亮提示「按下任意键」。
         高亮用 NuxtUI 内建 highlight prop (ring 画在内层 input 上, 跟圆角对齐) —— 不要往外层
         包裹 div 糊 ring class, 那样 ring 是方角的跟圆角输入框对不上。 -->
    <UInput
      :model-value="modelValue"
      :placeholder="capturing ? t('hotkeyInput.press_key') : t('hotkeyInput.key_example')"
      size="sm"
      :readonly="capturing"
      :highlight="capturing"
      :color="capturing ? 'primary' : undefined"
      class="flex-1"
      @update:model-value="(v: string) => emit('update:modelValue', v)"
    />
    <!-- 键盘 icon: 点一下进录制态 (icon 变 X = 取消), 录制态再点取消。 -->
    <UButton
      size="xs"
      :variant="capturing ? 'solid' : 'soft'"
      :color="capturing ? 'primary' : 'neutral'"
      :icon="capturing ? 'i-tabler-x' : 'i-tabler-keyboard'"
      :title="capturing ? t('hotkeyInput.click_to_cancel') : t('hotkeyInput.click_to_record')"
      @click="capturing ? stopCapture() : startCapture()"
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

// 把 Web KeyboardEvent.key / code 映射到我们 vk 字符串。
// vk 在 backend / pkg/input 里需匹配 vkName 表（字母 / 数字 / Fn / 特殊键），单键，不带组合。
// 组合键 (Ctrl+R 这类) 不是节点 vk 概念 —— backend 一个 vk 只按一个键; 要组合走
// KeyHoldStart(Ctrl) → KeyPress(R) → KeyHoldStop(Ctrl) 的节点编排, 所以这里只录单键。
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
  }
  if (map[e.key]) return map[e.key]
  // F1..F12
  if (/^F([1-9]|1[0-2])$/.test(e.key)) return e.key
  // fallback: 直接用 key
  return e.key
}

function onKeyDown(e: KeyboardEvent) {
  if (!capturing.value) return
  // 忽略只按修饰键（用户可能正按 Shift 准备按字母）—— 单键 vk 不收修饰键本身
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
