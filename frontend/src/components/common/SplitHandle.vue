<template>
  <div
    class="shrink-0 group relative select-none"
    :class="vertical ? 'h-1 w-full cursor-row-resize' : 'w-1 h-full cursor-col-resize'"
    @pointerdown="onDown"
  >
    <div
      class="absolute inset-0 transition-colors"
      :class="dragging ? 'bg-primary/40' : 'group-hover:bg-primary/20'"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  /** Current width in px. */
  modelValue: number
  /** If true, dragging right increases value (right-side handle on left of pane). */
  reverse?: boolean
  /** If true, handle is horizontal (row resize). */
  vertical?: boolean
  min: number
  max: number
}>()

const emit = defineEmits<{ 'update:modelValue': [value: number] }>()

const dragging = ref(false)
let startX = 0
let startY = 0
let startVal = 0

function onDown(e: PointerEvent) {
  dragging.value = true
  startX = e.clientX
  startY = e.clientY
  startVal = props.modelValue
  ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}

function onMove(e: PointerEvent) {
  const delta = props.vertical ? e.clientY - startY : e.clientX - startX
  const next = props.reverse ? startVal - delta : startVal + delta
  const clamped = Math.min(props.max, Math.max(props.min, next))
  emit('update:modelValue', clamped)
}

function onUp() {
  dragging.value = false
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', onUp)
}
</script>
