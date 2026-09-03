<template>
  <article
    data-testid="workflow-annotation"
    ref="surface"
    class="flex min-w-48 resize flex-col overflow-hidden rounded-lg border bg-warning/10 shadow-sm"
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
      <span class="min-w-0 flex-1 truncate text-[10px] font-semibold uppercase tracking-wider">
        {{ t('workflow.graphs.comment_label') }}
      </span>
      <button
        data-testid="workflow-annotation-edit"
        type="button"
        class="nodrag nopan -m-1 grid size-7 shrink-0 place-items-center rounded-md text-warning/80 transition-colors hover:bg-warning/15 hover:text-warning focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
        :aria-label="
          editing ? t('workflow.graphs.comment_preview') : t('workflow.graphs.comment_edit')
        "
        :title="editing ? t('workflow.graphs.comment_preview') : t('workflow.graphs.comment_edit')"
        @pointerdown.prevent.stop
        @click.stop="toggleEditing"
      >
        <UIcon :name="editing ? 'i-tabler-eye' : 'i-tabler-pencil'" class="size-4" />
      </button>
    </header>
    <textarea
      v-if="editing"
      data-testid="workflow-annotation-editor"
      ref="editor"
      v-model="text"
      class="nodrag nopan block min-h-0 w-full flex-1 resize-none overflow-y-auto bg-transparent px-3 py-2 text-xs leading-5 text-highlighted outline-none placeholder:text-warning/50"
      :placeholder="t('workflow.graphs.comment_placeholder')"
      @blur="finishEditing"
      @pointerdown.stop
    />
    <!-- markdown-it is configured with html:false; annotation text cannot inject raw HTML. -->
    <div
      v-else
      data-testid="workflow-annotation-content"
      class="workflow-annotation-markdown nodrag nopan min-h-0 flex-1 cursor-text overflow-y-auto px-3 py-2 text-xs leading-5 text-highlighted select-text"
      tabindex="0"
      @dblclick.stop="beginEditing"
      @pointerdown.stop
      v-html="renderedText"
    />
  </article>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Annotation } from '../../../../contracts/workflow/current/workflow-source'
import { editableAnnotationUpdate } from './annotationUpdate'

const props = defineProps<{ annotation: Annotation; selected?: boolean }>()
const emit = defineEmits<{ update: [annotation: Annotation] }>()
const { t } = useI18n()
const surface = ref<HTMLElement | null>(null)
const editor = ref<HTMLTextAreaElement | null>(null)
const text = ref(props.annotation.text)
const editing = ref(!text.value.trim())
const renderedText = ref('')
let renderRevision = 0

watch(
  text,
  async (value) => {
    const revision = ++renderRevision
    const { renderMarkdown } = await import('@/lib/markdown')
    if (revision === renderRevision) renderedText.value = renderMarkdown(value)
  },
  { immediate: true },
)

watch(
  () => props.annotation.text,
  (value) => {
    text.value = value
  },
)

function commitText(): void {
  if (text.value !== props.annotation.text)
    emit('update', editableAnnotationUpdate(props.annotation, { text: text.value }))
}

function beginEditing(): void {
  editing.value = true
  void nextTick(() => {
    const element = editor.value
    if (!element) return
    element.focus({ preventScroll: true })
    element.setSelectionRange(0, 0)
    element.scrollTop = 0
  })
}

function finishEditing(): void {
  commitText()
  if (text.value.trim()) editing.value = false
}

function toggleEditing(): void {
  if (editing.value) finishEditing()
  else beginEditing()
}

function commitSize(): void {
  const element = surface.value
  if (!element) return
  const width = Math.max(80, Math.round(element.offsetWidth))
  const height = Math.max(48, Math.round(element.offsetHeight))
  if (width !== props.annotation.size.width || height !== props.annotation.size.height) {
    emit('update', editableAnnotationUpdate(props.annotation, { size: { width, height } }))
  }
}
</script>

<style scoped>
.workflow-annotation-markdown {
  overflow-wrap: anywhere;
}

.workflow-annotation-markdown :deep(> :first-child) {
  margin-top: 0;
}

.workflow-annotation-markdown :deep(> :last-child) {
  margin-bottom: 0;
}

.workflow-annotation-markdown :deep(p),
.workflow-annotation-markdown :deep(ul),
.workflow-annotation-markdown :deep(ol),
.workflow-annotation-markdown :deep(blockquote),
.workflow-annotation-markdown :deep(pre) {
  margin-block: 0.45rem;
}

.workflow-annotation-markdown :deep(h1),
.workflow-annotation-markdown :deep(h2),
.workflow-annotation-markdown :deep(h3) {
  margin-block: 0.75rem 0.3rem;
  color: var(--ui-text-highlighted);
  font-weight: 700;
  line-height: 1.3;
}

.workflow-annotation-markdown :deep(h1) {
  font-size: 1rem;
}

.workflow-annotation-markdown :deep(h2) {
  font-size: 0.875rem;
}

.workflow-annotation-markdown :deep(h3) {
  font-size: 0.75rem;
}

.workflow-annotation-markdown :deep(ul),
.workflow-annotation-markdown :deep(ol) {
  padding-inline-start: 1.25rem;
}

.workflow-annotation-markdown :deep(ul) {
  list-style: disc;
}

.workflow-annotation-markdown :deep(ol) {
  list-style: decimal;
}

.workflow-annotation-markdown :deep(strong) {
  color: var(--ui-text-highlighted);
  font-weight: 700;
}

.workflow-annotation-markdown :deep(code) {
  border-radius: 0.25rem;
  background: color-mix(in srgb, var(--ui-warning) 12%, transparent);
  padding: 0.08rem 0.25rem;
  font-family: var(--font-mono);
  font-size: 0.9em;
}

.workflow-annotation-markdown :deep(blockquote) {
  border-inline-start: 1px solid color-mix(in srgb, var(--ui-warning) 45%, transparent);
  padding-inline-start: 0.65rem;
  color: color-mix(in srgb, var(--ui-text-highlighted) 75%, var(--ui-warning));
}

.workflow-annotation-markdown :deep(a) {
  color: var(--ui-primary);
  text-decoration: underline;
  text-underline-offset: 0.15em;
}
</style>
