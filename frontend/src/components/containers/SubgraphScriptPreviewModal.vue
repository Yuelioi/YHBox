<!-- 子图转脚本预览 modal — 成功态: CodeMirror 只读代码 + 复制/插入; 拒转态: 不支持节点
     清单(kind + 原因), 不出半对的代码。外壳 BaseModal, 编辑器外观同款 editorTheme。 -->
<template>
  <BaseModal
    :open="open"
    :title="t('subgraphScript.title', { name: sgLabel })"
    icon="i-tabler-code"
    size="3xl"
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <div v-if="failed" class="space-y-3">
      <p class="text-sm text-warning flex items-center gap-1.5">
        <UIcon name="i-tabler-alert-triangle" class="size-4 shrink-0" />
        {{ t('subgraphScript.unsupported_title') }}
      </p>
      <ul class="space-y-1.5">
        <li
          v-for="g in groupedUnsupported"
          :key="g.kind + g.reason"
          class="flex items-start gap-2 text-xs rounded-md bg-elevated/40 px-3 py-2"
        >
          <span class="font-mono text-toned shrink-0">
            {{ kindLabel(g.kind)
            }}<span v-if="g.count > 1" class="text-muted">&nbsp;×{{ g.count }}</span>
          </span>
          <span class="text-muted">{{ t(g.reason) }}</span>
        </li>
      </ul>
    </div>
    <div v-else ref="host" class="border border-default rounded-md overflow-hidden" />

    <template #footer>
      <UButton
        v-if="!failed"
        size="sm"
        variant="ghost"
        :color="copied ? 'success' : 'neutral'"
        :icon="copied ? 'i-tabler-check' : 'i-tabler-copy'"
        @click="onCopy"
      >
        {{ copied ? t('common.copied') : t('common.copy') }}
      </UButton>
      <UButton
        v-if="!failed"
        size="sm"
        variant="soft"
        color="primary"
        icon="i-tabler-square-rounded-plus"
        @click="emit('insert')"
      >
        {{ t('subgraphScript.insert') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorView } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { javascript } from '@codemirror/lang-javascript'
import { useToast } from '@nuxt/ui/composables'
import { baseEditorExtensions } from '@/lib/editorTheme'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import type { UnsupportedItem } from '@/lib/subgraphToScript'
import BaseModal from '@/components/common/BaseModal.vue'

const { t, te } = useI18n()
const toast = useToast()

const props = defineProps<{
  open: boolean
  sgLabel: string
  code: string
  unsupported: UnsupportedItem[]
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  insert: []
}>()

const failed = computed(() => props.unsupported.length > 0)

// 拒转清单按 (kind, 原因) 去重计数 — 同一类节点(如 6 个表达式)只列一行 + ×N, 不堆重复。
const groupedUnsupported = computed(() => {
  const m = new Map<string, { kind: string; reason: string; count: number }>()
  for (const u of props.unsupported) {
    const k = `${u.kind}|${u.reason}`
    const e = m.get(k)
    if (e) e.count++
    else m.set(k, { kind: u.kind, reason: u.reason, count: 1 })
  }
  return [...m.values()]
})

function kindLabel(kind: string): string {
  const key = getSpec(kind)?.labelZh ?? ''
  return key && te(key) ? t(key) : kind
}

// 只读 CodeMirror: modal 内容 lazy mount, open 后 nextTick 建 view。
const host = ref<HTMLElement | null>(null)
let view: EditorView | null = null

watch(
  () => [props.open, props.code] as const,
  async ([open]) => {
    view?.destroy()
    view = null
    if (!open || failed.value) return
    await nextTick()
    if (!host.value) return
    view = new EditorView({
      parent: host.value,
      doc: props.code,
      extensions: [
        baseEditorExtensions({ modal: true }),
        javascript(),
        EditorState.readOnly.of(true),
      ],
    })
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})

const copied = ref(false)
let copiedTimer = 0
async function onCopy() {
  try {
    await navigator.clipboard.writeText(props.code)
    copied.value = true
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
      copied.value = false
    }, 1500)
  } catch {
    toast.add({ title: t('toast.copy_failed'), color: 'error' })
  }
}
</script>
