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
import { keyEventToVK } from './keyCapture'

const { t } = useI18n()

defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const capturing = ref(false)

function onKeyDown(e: KeyboardEvent) {
  if (!capturing.value) return
  // Meta (Win 键) backend vkMap 不认 → 录进去 runtime 报 INVALID_VK, 忽略掉。
  // Ctrl/Shift/Alt 照录 —— 录制是手动进入的, 用户按哪个就是要哪个。
  if (e.key === 'Meta') return
  e.preventDefault()
  e.stopPropagation()
  emit('update:modelValue', keyEventToVK(e))
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
