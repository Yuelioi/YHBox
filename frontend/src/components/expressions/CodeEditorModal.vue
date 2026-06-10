<!-- Script 代码放大编辑 modal — BaseModal 包大 CodeMirror (同 CodeInput 一套扩展 + 行号)。
     编辑的是 draft (modal 内 doc), 确认才回写 update:modelValue; 取消/关闭丢弃。 -->
<template>
  <BaseModal
    :open="open"
    :title="t('inspector.code_editor_title')"
    icon="i-tabler-code"
    size="4xl"
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <div
      ref="host"
      class="h-[70vh] bg-elevated/80 border border-default rounded-md overflow-hidden focus-within:border-emerald-500"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="emit('update:open', false)">
        {{ t('common.cancel') }}
      </UButton>
      <UButton color="primary" @click="confirm">
        {{ t('common.confirm') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorView, lineNumbers } from '@codemirror/view'
import type { Completion } from '@codemirror/autocomplete'
import BaseModal from '@/components/common/BaseModal.vue'
import { scriptEditorExtensions } from '@/lib/scriptCompletions'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  modelValue: string
  completions?: Completion[]
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  'update:modelValue': [v: string]
}>()

const host = ref<HTMLElement | null>(null)
let view: EditorView | null = null

const theme = EditorView.theme({
  '&': { backgroundColor: 'transparent', fontSize: '12px', height: '100%' },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { overflow: 'auto' },
  '.cm-content': { fontFamily: 'ui-monospace, monospace', padding: '6px 0' },
  '.cm-line': { padding: '0 8px' },
  '.cm-gutters': { backgroundColor: 'transparent', border: 'none' },
  '.cm-tooltip': { fontSize: '12px' },
}, { dark: true })

// modal 内容随 open 挂/卸 (UModal 懒渲染) — editor 跟着 open 建/毁, 开时灌当前代码当 draft。
watch(() => props.open, async (open) => {
  if (open) {
    await nextTick()
    if (!host.value) return
    view = new EditorView({
      parent: host.value,
      doc: props.modelValue ?? '',
      extensions: [
        ...scriptEditorExtensions({
          completions: () => props.completions ?? [],
          placeholder: props.placeholder,
        }),
        lineNumbers(),
        theme,
      ],
    })
    view.focus()
  } else {
    view?.destroy()
    view = null
  }
})

function confirm() {
  if (view) emit('update:modelValue', view.state.doc.toString())
  emit('update:open', false)
}

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})
</script>
