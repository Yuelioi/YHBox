<template>
  <div class="space-y-2">
    <div class="flex items-center gap-2">
      <UButton
        v-if="!listening"
        type="button"
        color="neutral"
        variant="outline"
        icon="i-tabler-keyboard"
        class="min-w-0 flex-1 justify-start"
        @click="startListening"
      >
        <span v-if="modelValue.length" class="flex min-w-0 flex-wrap gap-1">
          <UKbd v-for="key in modelValue" :key="key">{{ key }}</UKbd>
        </span>
        <span v-else class="text-dimmed">{{ t('workflow.inspector.record_key_chord') }}</span>
      </UButton>
      <button
        v-else
        ref="listenerEl"
        type="button"
        class="min-w-0 flex-1 rounded-md border border-primary bg-primary/10 px-3 py-2 text-left text-xs text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        @keydown="onKeydown"
        @keyup="onKeyup"
        @blur="stopListening"
      >
        {{ t('workflow.inspector.record_key_chord_active') }}
      </button>
      <UButton
        v-if="listening"
        type="button"
        color="neutral"
        variant="ghost"
        size="xs"
        :label="t('common.cancel')"
        @mousedown.prevent
        @click="stopListening"
      />
      <UButton
        v-else-if="modelValue.length"
        type="button"
        color="neutral"
        variant="ghost"
        size="xs"
        icon="i-tabler-x"
        :aria-label="t('workflow.inspector.clear')"
        @click="emit('update:modelValue', [])"
      />
    </div>
    <p class="text-[11px] leading-5 text-muted">
      {{ t('workflow.inspector.record_key_chord_hint') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { keyChordFromKeyboardEvent, modifierChordFromKeyboardEvent } from './keyChord'

defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()
const { t } = useI18n()
const listening = ref(false)
const listenerEl = ref<HTMLElement | null>(null)
let hotkeysPaused = false

async function startListening(): Promise<void> {
  if (listening.value) return
  listening.value = true
  await backend.hotkeys.pause()
  hotkeysPaused = true
  await nextTick()
  listenerEl.value?.focus()
}

async function stopListening(): Promise<void> {
  listening.value = false
  if (!hotkeysPaused) return
  hotkeysPaused = false
  await backend.hotkeys.resume()
}

function onKeydown(event: KeyboardEvent): void {
  event.preventDefault()
  event.stopPropagation()
  const chord = keyChordFromKeyboardEvent(event)
  if (!chord) return
  emit('update:modelValue', chord)
  void stopListening()
}

function onKeyup(event: KeyboardEvent): void {
  event.preventDefault()
  event.stopPropagation()
  const chord = modifierChordFromKeyboardEvent(event)
  if (!chord) return
  emit('update:modelValue', chord)
  void stopListening()
}

onBeforeUnmount(() => {
  if (hotkeysPaused)
    void backend.hotkeys.resume().catch((error) => console.error('resume hotkeys failed', error))
})
</script>
