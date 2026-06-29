<template>
  <div class="flex items-center gap-2">
    <button
      v-if="!listening"
      type="button"
      :class="[
        'flex-1 px-3 py-1.5 rounded-md text-sm text-left font-mono transition-colors cursor-pointer',
        modelValue
          ? 'bg-elevated border border-default text-default hover:border-accented'
          : 'bg-elevated/50 border border-dashed border-default text-dimmed hover:text-muted hover:border-accented',
      ]"
      @click="startListening"
    >
      {{ modelValue || t('hotkeyInput.click_to_set') }}
    </button>
    <div
      v-else
      ref="listenerEl"
      tabindex="0"
      class="flex-1 px-3 py-1.5 rounded-md bg-primary/10 border border-primary text-sm text-primary cursor-pointer focus:outline-none"
      @keydown="onKeydown"
      @blur="stopListening"
    >
      <span class="animate-pulse">{{ t('hotkeyInput.instruction') }}</span>
    </div>
    <UButton
      v-if="modelValue && !listening"
      icon="i-tabler-x"
      size="xs"
      variant="ghost"
      color="neutral"
      @click="emit('update:modelValue', '')"
    />
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'

const { t } = useI18n()

defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const listening = ref(false)
const listenerEl = ref<HTMLElement | null>(null)

// 捕获模式时暂停所有 OS hotkey，让 webview 能收到正常 KEYDOWN。
// 否则用户按已注册的组合（如 Ctrl+Shift+1）会被 Win32 RegisterHotKey 拦截
// 直接派发给原 action，webview 收不到键，捕获失败。
async function startListening() {
  listening.value = true
  await backend.hotkeys.pause()
  await nextTick()
  listenerEl.value?.focus()
}

async function stopListening() {
  listening.value = false
  await backend.hotkeys.resume()
}

function onKeydown(e: KeyboardEvent) {
  // 阻止任何浏览器默认行为（Ctrl+R 刷新等）
  e.preventDefault()
  e.stopPropagation()

  if (e.code === 'Escape') {
    stopListening()
    return
  }

  // Backspace 清空绑定
  if (e.code === 'Backspace') {
    emit('update:modelValue', '')
    stopListening()
    return
  }

  // 纯 modifier 按键（用户还在配主键）→ 不做事，等主键
  if (isModifierOnly(e.code)) return

  const mods: string[] = []
  if (e.ctrlKey) mods.push('Ctrl')
  if (e.shiftKey) mods.push('Shift')
  if (e.altKey) mods.push('Alt')

  const key = mapKey(e.code)
  if (!key) return // 不支持的键

  emit('update:modelValue', [...mods, key].join('+'))
  stopListening()
}

function isModifierOnly(code: string): boolean {
  return (
    code === 'ControlLeft' ||
    code === 'ControlRight' ||
    code === 'ShiftLeft' ||
    code === 'ShiftRight' ||
    code === 'AltLeft' ||
    code === 'AltRight' ||
    code === 'MetaLeft' ||
    code === 'MetaRight'
  )
}

// mapKey 把 KeyboardEvent.code 映射到 backend parseHotkeyString 接受的格式。
function mapKey(code: string): string {
  if (/^Key[A-Z]$/.test(code)) return code.slice(3)
  if (/^Digit[0-9]$/.test(code)) return code.slice(5)
  if (/^F([1-9]|1[0-2])$/.test(code)) return code

  switch (code) {
    case 'Space':
      return 'Space'
    case 'Enter':
      return 'Enter'
    case 'Tab':
      return 'Tab'
    case 'Delete':
      return 'Delete'
    case 'Insert':
      return 'Insert'
    case 'Home':
      return 'Home'
    case 'End':
      return 'End'
    case 'PageUp':
      return 'PgUp'
    case 'PageDown':
      return 'PgDn'
    case 'ArrowUp':
      return 'Up'
    case 'ArrowDown':
      return 'Down'
    case 'ArrowLeft':
      return 'Left'
    case 'ArrowRight':
      return 'Right'
    case 'Period':
      return '.'
    case 'Comma':
      return ','
  }
  return ''
}
</script>
