<template>
  <article
    data-testid="workflow-annotation"
    ref="surface"
    class="min-w-48 resize overflow-auto rounded-lg border bg-warning/10 shadow-sm"
    :class="
      selected
        ? 'border-warning/45 ring-2 ring-primary/80 ring-offset-2 ring-offset-default'
        : 'border-warning/45'
    "
    :style="{ width: `${annotation.size.width}px`, height: `${annotation.size.height}px` }"
    @pointerup="commitSize"
  >
    <header
      class="workflow-node-drag-handle flex cursor-grab items-center gap-2 border-b border-warning/25 px-3 py-2 text-warning"
    >
      <UIcon name="i-tabler-note" class="size-4" />
      <span class="text-[10px] font-semibold uppercase tracking-wider">Comment</span>
    </header>
    <UTextarea
      v-model="text"
      class="nodrag nopan h-[calc(100%-37px)]"
      autoresize
      variant="none"
      :rows="3"
      placeholder="Add a note…"
      @blur="commitText"
      @pointerdown.stop
    />
  </article>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Annotation } from '../../../../contracts/workflow/3.1/workflow-source'

const props = defineProps<{ annotation: Annotation; selected?: boolean }>()
const emit = defineEmits<{ update: [annotation: Annotation] }>()
const surface = ref<HTMLElement | null>(null)
const text = ref(props.annotation.text)

watch(
  () => props.annotation.text,
  (value) => {
    text.value = value
  },
)

function commitText(): void {
  if (text.value !== props.annotation.text)
    emit('update', { ...props.annotation, text: text.value })
}

function commitSize(): void {
  const element = surface.value
  if (!element) return
  const width = Math.max(80, Math.round(element.offsetWidth))
  const height = Math.max(48, Math.round(element.offsetHeight))
  if (width !== props.annotation.size.width || height !== props.annotation.size.height) {
    emit('update', { ...props.annotation, size: { width, height } })
  }
}
</script>
