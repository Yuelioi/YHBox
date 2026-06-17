<!-- 表达式/代码放大编辑 modal — 语言无关的壳: BaseModal + 放大/还原 + 草稿确认 + 共享 <CodeEditor> 主体。
     调用方经 extensions() 注入语言 (Expr→exprEditorExtensions, Script→scriptEditorExtensions, modal:true 档)。
     编辑的是 draft, 确认 (按钮或 Ctrl+Enter) 才回写 update:modelValue; 取消/关闭丢弃。
     编辑器主体 (工具栏/参考面板/状态栏/补全) 全在 <CodeEditor>, 跟控制台共用同一份。 -->
<template>
  <BaseModal
    :open="open"
    :title="title"
    :icon="icon ?? 'i-tabler-code'"
    size="7xl"
    :tall="maximized"
    :content-class="maximized ? 'sm:max-w-[96vw]' : undefined"
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <template #header-extra>
      <UButton
        :icon="maximized ? 'i-tabler-minimize' : 'i-tabler-maximize'"
        variant="ghost"
        color="neutral"
        size="xs"
        :title="maximized ? t('inspector.editor_restore') : t('inspector.editor_maximize')"
        @click="maximized = !maximized"
      />
    </template>

    <div class="min-h-0" :class="maximized ? 'h-full' : 'h-[70vh]'">
      <CodeEditor
        ref="bodyRef"
        v-model:model-value="draftDoc"
        :extensions="extensions"
        :reference="reference"
        :lint-first="lintFirst"
        :commentable="commentable"
        :foldable="foldable"
        :snippet-lang="snippetLang"
        :lang-label="langLabel"
        @submit="confirm"
      >
        <template #toolbar-extra>
          <slot name="toolbar-extra" />
        </template>
      </CodeEditor>
    </div>

    <template #footer>
      <span class="text-[10px] text-muted mr-auto">{{ t('inspector.editor_confirm_hint') }}</span>
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
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Extension } from '@codemirror/state'
import BaseModal from '@/components/common/BaseModal.vue'
import CodeEditor from './CodeEditor.vue'
import type { CodeSnippetLang } from '@/stores/codeSnippets'
import type { InsertItem } from '@/lib/scriptCompletions'
import type { RefItem } from './CodeEditor.vue'

// RefItem 从 CodeEditor 转出 — 老调用方 (CodeInput / ExprInput) 仍 from './EditorModal.vue' 导入它, 不动它们。
export type { RefItem }

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  modelValue: string
  title: string
  icon?: string
  /** 每次打开时调用, 构建语言扩展 (高亮/补全/lint, modal 档); 不要带 onChange — draft 由本组件管。 */
  extensions: () => Extension[]
  reference?: RefItem[]
  /** 实时状态栏: 首条问题 (文案+位置, 点击跳转), null = 无问题。 */
  lintFirst?: (doc: string) => { message: string; from: number } | null
  /** 工具栏「注释」按钮 (Script JS 开, Expr 关)。 */
  commentable?: boolean
  /** 工具栏「折叠」按钮组 (Script 开, Expr 关)。 */
  foldable?: boolean
  /** 传了就有「片段」入口。 */
  snippetLang?: CodeSnippetLang
  /** 状态栏右侧语言标签。 */
  langLabel?: string
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  'update:modelValue': [v: string]
}>()

const maximized = ref(true)
const draftDoc = ref('')
const bodyRef = ref<InstanceType<typeof CodeEditor> | null>(null)

// 打开时灌当前值当 draft (CodeEditor 跟 modal 内容懒挂/卸, mount 时读 draftDoc 当初值)。
watch(() => props.open, (open) => {
  if (open) draftDoc.value = props.modelValue ?? ''
})

function confirm() {
  emit('update:modelValue', draftDoc.value)
  emit('update:open', false)
}

// CodeInput 经 ref 调 insert (新建变量后插 $引用) — 转给 <CodeEditor>。
defineExpose({ insert: (it: InsertItem) => bodyRef.value?.insert(it) })
</script>
