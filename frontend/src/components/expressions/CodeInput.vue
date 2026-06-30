<!-- Script 代码编辑器 (widget kind 'code', PinInput 分发) — CodeMirror 6 + JS 语法:
     高亮 + 补全 (节点函数/糖函数/动态输入名/pin 值位枚举值·变量名) + 悬停函数文档
     + 快速反馈 lint (语法错/未声明 $变量; 权威仍是后端 SCRIPT_PARSE_ERROR)。
     右上放大按钮弹 EditorModal 大编辑器: 工具栏/片段/折叠/参考面板/新建变量。 -->
<template>
  <div class="relative">
    <div
      ref="host"
      class="border border-default rounded-md overflow-hidden focus-within:border-primary/60"
    />
    <UButton
      icon="i-tabler-maximize"
      variant="ghost"
      color="neutral"
      size="xs"
      class="absolute top-1 right-1 opacity-60 hover:opacity-100"
      :title="t('inspector.code_expand')"
      @click="modalOpen = true"
    />
    <EditorModal
      ref="editorModalRef"
      v-model:open="modalOpen"
      :model-value="modelValue"
      :title="t('inspector.code_editor_title')"
      :extensions="modalExtensions"
      :reference="referenceItems"
      snippet-lang="script"
      :lint-first="lintFirst"
      lang-label="JavaScript"
      commentable
      foldable
      formattable
      @update:model-value="(v: string) => emit('update:modelValue', v)"
    >
      <template #toolbar-extra>
        <UButton
          v-if="declaredVars"
          icon="i-tabler-variable-plus"
          variant="ghost"
          color="neutral"
          size="xs"
          :title="t('inspector.editor_new_var')"
          @click="newVarOpen = true"
        >
          {{ t('inspector.editor_new_var') }}
        </UButton>
        <UDropdownMenu :items="scriptInsertMenuItems">
          <UButton
            icon="i-tabler-wand"
            trailing-icon="i-tabler-chevron-down"
            variant="ghost"
            color="neutral"
            size="xs"
            :loading="scriptInsertBusy"
            :title="t('inspector.editor_insert_tip')"
          >
            {{ t('inspector.editor_insert') }}
          </UButton>
        </UDropdownMenu>
        <UButton
          v-if="captureKeyBusy"
          :icon="captureKeyBusy ? 'i-tabler-x' : 'i-tabler-keyboard'"
          :variant="captureKeyBusy ? 'solid' : 'ghost'"
          :color="captureKeyBusy ? 'primary' : 'neutral'"
          size="xs"
          :title="
            captureKeyBusy
              ? t('inspector.editor_capture_key_cancel_tip')
              : t('inspector.editor_capture_key_tip')
          "
          @click="toggleKeyCaptureForScript"
        >
          {{
            captureKeyBusy
              ? t('inspector.editor_capture_key_waiting')
              : t('inspector.editor_capture_key')
          }}
        </UButton>
      </template>
    </EditorModal>
    <NewVarModal
      v-model:open="newVarOpen"
      initial-name=""
      :existing-var-names="(declaredVars ?? []).map((v) => v.name)"
      @confirm="onNewVar"
    />
    <BaseModal
      v-model:open="asyncCandidateOpen"
      :title="t('inspector.editor_async_candidates_title')"
      icon="i-tabler-list-search"
      size="lg"
    >
      <div class="space-y-3">
        <p class="text-xs text-muted">
          {{ asyncCandidateHint }}
        </p>
        <div v-if="asyncCandidateLoading" class="flex items-center gap-2 text-xs text-muted">
          <UIcon name="i-tabler-loader-2" class="size-4 animate-spin" />
          <span>{{ t('inspector.editor_async_loading') }}</span>
        </div>
        <div v-else-if="asyncCandidateItems.length" class="max-h-[48vh] overflow-y-auto space-y-1">
          <button
            v-for="item in asyncCandidateItems"
            :key="item.value"
            type="button"
            class="w-full text-left rounded-md border border-default bg-elevated/40 hover:bg-elevated px-3 py-2 transition-colors"
            @click="insertAsyncCandidate(item)"
          >
            <span class="block text-sm text-highlighted truncate">{{ item.label }}</span>
            <span class="block text-[11px] text-muted truncate">{{
              item.detail || item.value
            }}</span>
          </button>
        </div>
        <p v-else class="text-xs text-muted">{{ t('inspector.editor_async_empty') }}</p>
      </div>
    </BaseModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { EditorView } from '@codemirror/view'
import type { Completion } from '@codemirror/autocomplete'
import { useNodeRegistryStore } from '@/stores/nodeRegistry'
import {
  nodeFnCompletions,
  nodeFnItems,
  scriptAIConnectionItemsForPin,
  scriptAsyncDropdownTargetForPin,
  scriptAssetItemsForPin,
  scriptColorInsertText,
  scriptColorPickTargetForPin,
  scriptCurrentCallInputSnapshot,
  scriptDollarRefs,
  scriptEditorExtensions,
  scriptExitItemsForKind,
  scriptGeometryInsertText,
  scriptStringInsertText,
  scriptTemplateInsertMode,
  scriptTemplateInsertText,
  scriptTemplateItemsForPin,
  scriptPointInsertText,
  scriptSyntaxErrors,
  SUGAR_COMPLETIONS,
  SUGAR_ITEMS,
} from '@/lib/scriptCompletions'
import { scriptPinValueContext } from '@/lib/editorSignature'
import { renderDoc, type HoverDoc } from '@/lib/editorHover'
import { snippetCompletions } from '@/lib/snippetCompletion'
import { useCodeSnippetsStore } from '@/stores/codeSnippets'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useSettingsStore } from '@/stores/settings'
import { useTemplatesStore } from '@/stores/templates'
import type { VarType } from '@/lib/variableRef'
import { backend } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { errorMessage } from '@/lib/invoke'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { ALL_NODE_GROUPS, groupLabelColor } from '@/composables/editor/useNodeGroupColor'
import NewVarModal from '@/components/containers/NewVarModal.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import EditorModal, { type RefItem } from './EditorModal.vue'
import type { AssetSummary } from '@/lib/backend'
import { keyEventToVK } from '@/components/containers/keyCapture'
import { NodeService } from '@bindings/yotta/internal/node'
import type { AsyncOption } from '@/components/containers/inline/asyncDropdown'

const { t, te } = useI18n()
const toast = useToast()

const props = defineProps<{
  modelValue: string
  placeholder?: string
  /** 动态输入名 (config.Inputs[] 声明) — 进补全, 让脚本里直接引用。 */
  inputNames?: string[]
  /** 容器变量 (名+类型) — varname pin 值位补全 + $引用 lint + 参考面板「变量」组 + 新建变量。 */
  declaredVars?: { name: string; type: VarType }[]
}>()

const emit = defineEmits<{
  'update:modelValue': [v: string]
  'declare-var': [args: { name: string; type: VarType; default: unknown }]
}>()

const host = ref<HTMLElement | null>(null)
const modalOpen = ref(false)
const newVarOpen = ref(false)
const editorModalRef = ref<InstanceType<typeof EditorModal> | null>(null)
let view: EditorView | null = null

const registry = useNodeRegistryStore()
const containerEditorStore = useContainerEditorStore()
const codeSnippetsStore = useCodeSnippetsStore()
const settingsStore = useSettingsStore()
const tplStore = useTemplatesStore()
const assetSummaries = ref<AssetSummary[]>([])
void codeSnippetsStore.ensureLoaded() // prefix 补全要用, 小框编辑器一挂载就拉

function kindLabel(kind: string): string {
  const key = `node.${kind}.label`
  return te(key) ? t(key) : ''
}

const varNames = computed<string[]>(() => (props.declaredVars ?? []).map((v) => v.name))

// 节点 + 糖函数补全项各带 info: 复用 hoverDoc 同一份文档数据 (单一来源)。
const withInfo = (c: Completion): Completion => ({
  ...c,
  info: () => {
    const d = hoverDoc(c.label)
    return d ? renderDoc(d) : null
  },
})

const completionOptions = computed<Completion[]>(() => [
  ...nodeFnCompletions(registry.scriptBindableKinds, registry.specs, kindLabel).map(withInfo),
  ...SUGAR_COMPLETIONS.map(withInfo),
  ...(props.inputNames ?? []).map((n) => ({ label: n, type: 'variable' as const })),
  // $hp live getter (读容器变量 hp 的实时值)
  ...(props.declaredVars ?? []).map((v) => ({
    label: `$${v.name}`,
    type: 'variable' as const,
    detail: v.type,
  })),
  // 用户片段: 打触发词 → 整段 body 上屏
  ...snippetCompletions(codeSnippetsStore.byLang('script')),
])

// 糖函数展开说明 (i18n key 用 _ 替代 ".": vue-i18n 把 "." 当层级分隔)。
function sugarDocs(label: string): string {
  const key = `script.fn.${label.replace(/\./g, '_')}`
  return te(key) ? t(key) : ''
}

// 枚举 pin 候选值 (dropdown widget 的 options): (kind, pin) → [{value, i18n label}]。
// 通用: 任何带 dropdown options 的 pin 自动覆盖 (Scope / Random dist …), 不写死 kind。
function pinEnumOptions(kind: string, pin: string): { value: string; label?: string }[] {
  const input = registry.specs.get(kind)?.inputs?.find((i) => i.name === pin)
  const raw = (input?.widget?.props as Record<string, unknown> | undefined)?.options
  if (!Array.isArray(raw)) return []
  return raw.map((o) => {
    const value = String((o as { value: unknown }).value)
    const k = `node.${kind}.input.${pin}.option.${value}`
    const label = te(k) ? t(k) : ''
    return { value, label: label && label !== value ? label : undefined }
  })
}

// pin 值位置补全候选: 枚举 pin → 候选值 (type enum); varname pin (Semantic "varname",
// 如 GetVar/SetVar 的 VarName) → 容器变量名; SubgraphID pin (脚本 Subgraph() 调子图)
// → 容器内可见子图 ID (显示子图名)。其余 pin 无候选。
function pinValues(
  kind: string,
  pin: string,
): {
  value: string
  label?: string
  detail?: string
  type?: 'enum' | 'variable' | 'asset'
  insertMode?: 'string' | 'template'
}[] {
  const enums = pinEnumOptions(kind, pin)
  if (enums.length) return enums.map((e) => ({ ...e, type: 'enum' as const }))
  const input = registry.specs.get(kind)?.inputs?.find((i) => i.name === pin)
  const templateItems = scriptTemplateItemsForPin(
    kind,
    pin,
    registry.specs,
    Object.values(tplStore.map),
  )
  if (templateItems.length) return templateItems
  const assetItems = scriptAssetItemsForPin(kind, pin, registry.specs, assetSummaries.value)
  if (assetItems.length) return assetItems
  const aiConnectionItems = scriptAIConnectionItemsForPin(
    kind,
    pin,
    registry.specs,
    settingsStore.data?.ai.connections ?? [],
  )
  if (aiConnectionItems.length) return aiConnectionItems
  if (input?.semantic === 'varname') {
    return varNames.value.map((n) => ({ value: n, type: 'variable' as const }))
  }
  if (input?.semantic === 'SubgraphID') {
    return containerEditorStore.visibleSubgraphs.map((s) => ({
      value: s.id,
      label: s.label,
      type: 'enum' as const,
    }))
  }
  return []
}

// 节点 pin 参数表 (展开详情): 名 / i18n 人话 label / 类型 / 必填 / 枚举候选。
function nodeParams(kind: string): RefItem['params'] {
  const spec = registry.specs.get(kind)
  return (spec?.inputs ?? [])
    .filter((i) => i.type !== 'Exec')
    .map((i) => {
      const labelKey = `node.${kind}.input.${i.name}.label`
      const opts = pinEnumOptions(kind, i.name)
      return {
        name: i.name,
        label: te(labelKey) ? t(labelKey) : '',
        type: i.type,
        required: !!i.required,
        options: opts.length ? opts.map((o) => o.value) : undefined,
      }
    })
}

function nodeText(kind: string, field: 'description' | 'example'): string {
  const key = `node.${kind}.${field}`
  return te(key) ? t(key) : ''
}

// 悬停文档: $变量 → 类型; 糖函数 → 签名+说明; 节点函数 → 签名+中文名/说明+参数表。
function hoverDoc(word: string): HoverDoc | null {
  if (word.startsWith('$')) {
    const v = (props.declaredVars ?? []).find((x) => x.name === word.slice(1))
    return v ? { sig: word, desc: v.type } : null
  }
  const sugar = SUGAR_ITEMS.find((s) => s.label === word)
  if (sugar) return { sig: sugar.detail ?? word, desc: sugarDocs(word) || undefined }
  if (registry.scriptBindableKinds.includes(word)) {
    const spec = registry.specs.get(word)
    if (!spec) return null
    const pins = (spec.inputs ?? [])
      .filter((i) => i.type !== 'Exec')
      .map((i) => i.name)
      .join(', ')
    const desc = [kindLabel(word), nodeText(word, 'description')].filter(Boolean).join(' — ')
    return { sig: `${word}({${pins}})`, desc: desc || undefined, params: nodeParams(word) }
  }
  return null
}

const lintMessages = {
  syntaxError: (line: number) => t('inspector.editor_syntax_error_line', { line }),
  unknownVar: (name: string) => t('inspector.editor_unknown_var', { name }),
}

// modal 状态栏首条问题: 语法错优先, 其次未声明 $变量。
function lintFirst(doc: string): { message: string; from: number } | null {
  const syn = scriptSyntaxErrors(doc)[0]
  if (syn) return { message: lintMessages.syntaxError(syn.line), from: syn.from }
  const { refs, defined } = scriptDollarRefs(doc)
  const known = new Set(varNames.value)
  const bad = refs.find((r) => !known.has(r.name) && !defined.has(r.name))
  return bad ? { message: lintMessages.unknownVar(bad.name), from: bad.from } : null
}

// 节点按画布同款分类分组 (nodeGroup.* 标签 + 分类配色), 组内按 kind 排; 分类顺序对齐面板。
function groupedNodeItems(): RefItem[] {
  const items = nodeFnItems(registry.scriptBindableKinds, registry.specs, kindLabel).map((it) => {
    const g = getSpec(it.label)?.group ?? 'system'
    const gKey = `nodeGroup.${g}`
    return {
      ...it,
      group: te(gKey) ? t(gKey) : g,
      groupClass: groupLabelColor(g),
      docs: nodeText(it.label, 'description') || undefined,
      example: nodeText(it.label, 'example') || undefined,
      params: nodeParams(it.label),
      _g: g,
    }
  })
  const order = new Map(ALL_NODE_GROUPS.map((g, i) => [g as string, i]))
  items.sort(
    (a, b) => (order.get(a._g) ?? 99) - (order.get(b._g) ?? 99) || a.label.localeCompare(b.label),
  )
  return items
}

// 放大编辑的参考面板: 变量 / 糖函数 / 本节点动态输入 / 节点 (按分类, 可展开用法+参数)。
const referenceItems = computed<RefItem[]>(() => [
  ...(props.declaredVars ?? []).map((v) => ({
    label: `$${v.name}`,
    desc: v.type,
    insert: `$${v.name}`,
    caretBack: 0,
    group: t('inspector.editor_ref_group_vars'),
  })),
  ...SUGAR_ITEMS.map((it) => ({
    ...it,
    group: t('inspector.editor_ref_group_fns'),
    docs: sugarDocs(it.label) || undefined,
  })),
  ...(props.inputNames ?? []).map((n) => ({
    label: n,
    insert: n,
    caretBack: 0,
    group: t('inspector.editor_ref_group_inputs'),
  })),
  ...groupedNodeItems(),
])

// 新建变量 (工具栏直接建, 免去切侧栏的割裂): 声明上抛 + 顺手插一个 $引用。
function onNewVar(a: { name: string; type: VarType; default: unknown }) {
  emit('declare-var', a)
  editorModalRef.value?.insert({ label: a.name, insert: `$${a.name}`, caretBack: 0 })
}

const captureTemplateBusy = ref(false)
const capturePointBusy = ref(false)
const captureRectBusy = ref(false)
const captureColorBusy = ref(false)
const captureKeyBusy = ref(false)
const asyncCandidateOpen = ref(false)
const asyncCandidateLoading = ref(false)
const asyncCandidateItems = ref<AsyncCandidateItem[]>([])
const asyncCandidateContext = ref<{ kind: string; pin: string } | null>(null)

type AsyncCandidateItem = {
  value: string
  label: string
  detail?: string
}

const scriptInsertBusy = computed(
  () =>
    captureTemplateBusy.value ||
    capturePointBusy.value ||
    captureRectBusy.value ||
    captureColorBusy.value ||
    asyncCandidateLoading.value,
)

const scriptInsertMenuItems = computed(() => [
  [
    {
      label: t('inspector.editor_insert_candidate'),
      icon: 'i-tabler-list-search',
      disabled: asyncCandidateLoading.value,
      onSelect: () => void loadAsyncCandidatesForScript(),
    },
  ],
  [
    {
      label: t('inspector.editor_capture_template'),
      icon: 'i-tabler-camera-plus',
      disabled: captureTemplateBusy.value,
      onSelect: () => void captureTemplateForScript(),
    },
    {
      label: t('inspector.editor_capture_point'),
      icon: 'i-tabler-crosshair',
      disabled: capturePointBusy.value,
      onSelect: () => void capturePointForScript(),
    },
    {
      label: t('inspector.editor_capture_rect'),
      icon: 'i-tabler-select',
      disabled: captureRectBusy.value,
      onSelect: () => void captureRectForScript(),
    },
    {
      label: t('inspector.editor_capture_color'),
      icon: 'i-tabler-color-picker',
      disabled: captureColorBusy.value,
      onSelect: () => void captureColorForScript(),
    },
    {
      label: t('inspector.editor_capture_key'),
      icon: 'i-tabler-keyboard',
      disabled: captureKeyBusy.value,
      onSelect: toggleKeyCaptureForScript,
    },
  ],
])

const asyncCandidateHint = computed(() => {
  const ctx = asyncCandidateContext.value
  if (!ctx) return t('inspector.editor_async_candidates_hint')
  return t('inspector.editor_async_candidates_context', { kind: ctx.kind, pin: ctx.pin })
})
function genTemplatePickID(): string {
  return 'script-tpl-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
}

function genScriptPickID(prefix: string): string {
  return prefix + '-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
}

async function reloadAssets() {
  const summaries = await backend.assets.list()
  assetSummaries.value = (summaries as AssetSummary[] | undefined) ?? []
}

async function captureTemplateForScript() {
  if (captureTemplateBusy.value) return
  const id = genTemplatePickID()
  captureTemplateBusy.value = true
  try {
    const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
      'tools:picker-result',
      (p) => p?.id === id,
    )
    await backend.tools.openScreenPicker('template_save', id, tplStore.containerId)
    const result = await waiter
    const guid = result.payload?.guid
    if (result.payload?.cancelled || typeof guid !== 'string' || !guid) return
    await tplStore.reload()
    await reloadAssets()
    const doc = editorModalRef.value?.currentDoc() ?? props.modelValue ?? ''
    const pos = editorModalRef.value?.currentCursor() ?? doc.length
    const mode = scriptTemplateInsertMode(doc, pos)
    editorModalRef.value?.insertText(scriptTemplateInsertText(guid, mode))
  } finally {
    captureTemplateBusy.value = false
  }
}

type PointPayload = { xRatio: number; yRatio: number; cancelled?: boolean }
type RectPayload = { region: [number, number, number, number]; cancelled?: boolean }
type ColorPayload = { range: number[]; hueWrap?: boolean; cancelled?: boolean }

async function capturePointForScript() {
  if (capturePointBusy.value) return
  const id = genScriptPickID('script-pt')
  capturePointBusy.value = true
  try {
    const waiter = awaitWailsEvent<{ id: string; mode: string; payload: PointPayload }>(
      'tools:picker-result',
      (p) => p?.id === id,
    )
    const opened = await backend.tools.openScreenPicker('point', id, tplStore.containerId)
    if (opened === undefined) return
    const result = await waiter
    const p = result.payload
    if (!p || p.cancelled) return
    editorModalRef.value?.insertText(scriptPointInsertText(p))
  } finally {
    capturePointBusy.value = false
  }
}

async function captureRectForScript() {
  if (captureRectBusy.value) return
  const id = genScriptPickID('script-rect')
  captureRectBusy.value = true
  try {
    const waiter = awaitWailsEvent<{ id: string; mode: string; payload: RectPayload }>(
      'tools:picker-result',
      (p) => p?.id === id,
    )
    const opened = await backend.tools.openScreenPicker('rect', id, tplStore.containerId)
    if (opened === undefined) return
    const result = await waiter
    const p = result.payload
    if (!p || p.cancelled) return
    editorModalRef.value?.insertText(scriptGeometryInsertText(p))
  } finally {
    captureRectBusy.value = false
  }
}

function currentScriptDocAndPos(): { doc: string; pos: number } {
  const doc = editorModalRef.value?.currentDoc() ?? props.modelValue ?? ''
  return { doc, pos: editorModalRef.value?.currentCursor() ?? doc.length }
}

async function loadAsyncCandidatesForScript() {
  if (asyncCandidateLoading.value) return
  const { doc, pos } = currentScriptDocAndPos()
  const pinCtx = scriptPinValueContext(doc, pos)
  const target = pinCtx
    ? scriptAsyncDropdownTargetForPin(pinCtx.kind, pinCtx.pin, registry.specs)
    : null
  if (!pinCtx || !target) {
    toast.add({
      title: t('inspector.editor_async_no_context'),
      color: 'warning',
      icon: 'i-tabler-info-circle',
    })
    return
  }

  asyncCandidateContext.value = pinCtx
  asyncCandidateItems.value = []
  asyncCandidateOpen.value = true
  asyncCandidateLoading.value = true
  try {
    const options = await NodeService.AsyncOptions(
      '',
      pinCtx.kind,
      target.asyncSource,
      scriptCurrentCallInputSnapshot(doc, pos),
    )
    asyncCandidateItems.value = ((options ?? []) as AsyncOption[]).map((o) => {
      const value = String(o.value ?? '')
      return {
        value,
        label: o.label || value,
        detail: formatAsyncCandidateDetail(o),
      }
    })
    if (!asyncCandidateItems.value.length) {
      asyncCandidateOpen.value = false
      toast.add({
        title: t('inspector.editor_async_empty'),
        color: 'neutral',
        icon: 'i-tabler-list-search',
      })
    }
  } catch (e) {
    asyncCandidateOpen.value = false
    toast.add({
      title: t('inspector.editor_async_load_failed'),
      description: errorMessage(e),
      color: 'error',
      icon: 'i-tabler-alert-circle',
    })
  } finally {
    asyncCandidateLoading.value = false
  }
}

function formatAsyncCandidateDetail(option: AsyncOption): string | undefined {
  const parts = Object.entries(option.meta ?? {})
    .filter(([, v]) => v != null && v !== '')
    .map(([k, v]) => `${k}: ${String(v)}`)
  return parts.length ? parts.join(' · ') : undefined
}

function insertAsyncCandidate(item: AsyncCandidateItem) {
  const { doc, pos } = currentScriptDocAndPos()
  editorModalRef.value?.insertText(scriptStringInsertText(item.value, doc, pos))
  asyncCandidateOpen.value = false
}

async function captureColorForScript() {
  if (captureColorBusy.value) return
  const { doc, pos } = currentScriptDocAndPos()
  const pinCtx = scriptPinValueContext(doc, pos)
  const target = pinCtx
    ? scriptColorPickTargetForPin(pinCtx.kind, pinCtx.pin, registry.specs, doc, pos)
    : null
  const colorTarget = target ?? { colorSpace: 'hsv' as const, shape: 'object' as const }
  const id = genScriptPickID('script-color')
  captureColorBusy.value = true
  try {
    const waiter = awaitWailsEvent<{ id: string; mode: string; payload: ColorPayload }>(
      'tools:picker-result',
      (p) => p?.id === id,
    )
    const opened = await backend.tools.openScreenPicker(
      'color',
      id,
      tplStore.containerId,
      '',
      colorTarget.colorSpace,
    )
    if (opened === undefined) return
    const result = await waiter
    const p = result.payload
    if (!p || p.cancelled) return
    editorModalRef.value?.insertText(scriptColorInsertText(p, colorTarget.shape))
  } finally {
    captureColorBusy.value = false
  }
}

function toggleKeyCaptureForScript() {
  if (captureKeyBusy.value) {
    stopKeyCaptureForScript()
    return
  }
  captureKeyBusy.value = true
  window.addEventListener('keydown', onScriptKeyDown, { capture: true })
}

function stopKeyCaptureForScript() {
  captureKeyBusy.value = false
  window.removeEventListener('keydown', onScriptKeyDown, { capture: true })
}

function onScriptKeyDown(e: KeyboardEvent) {
  if (!captureKeyBusy.value) return
  if (e.key === 'Meta') return
  e.preventDefault()
  e.stopPropagation()
  const { doc, pos } = currentScriptDocAndPos()
  editorModalRef.value?.insertText(scriptStringInsertText(keyEventToVK(e), doc, pos))
  stopKeyCaptureForScript()
}

function buildExtensions(opts: { modal?: boolean; onChange?: (doc: string) => void }) {
  return scriptEditorExtensions({
    completions: () => completionOptions.value,
    varNames: () => varNames.value,
    pinValues,
    exitValues: (kind: string) => scriptExitItemsForKind(kind, registry.specs),
    hoverDoc,
    signatureLookup: (name: string) => {
      const d = hoverDoc(name)
      return d ? { sig: d.sig } : null
    },
    lintMessages,
    placeholder: props.placeholder,
    minHeight: '10em',
    ...opts,
  })
}

// modal 的扩展工厂 — 不带 onChange (draft 由 modal 自管, 确认才回写)。
function modalExtensions() {
  return buildExtensions({ modal: true })
}

onMounted(() => {
  if (!settingsStore.loaded) void settingsStore.load()
  if (Object.keys(tplStore.map).length === 0) void tplStore.reload()
  void reloadAssets()
  view = new EditorView({
    parent: host.value!,
    doc: props.modelValue ?? '',
    extensions: buildExtensions({ onChange: (doc) => emit('update:modelValue', doc) }),
  })
})

watch(
  () => props.modelValue,
  (v) => {
    if (!view) return
    const cur = view.state.doc.toString()
    if (v !== cur) {
      view.dispatch({ changes: { from: 0, to: cur.length, insert: v ?? '' } })
    }
  },
)

onBeforeUnmount(() => {
  stopKeyCaptureForScript()
  view?.destroy()
  view = null
})
</script>
