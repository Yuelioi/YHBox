<!-- SaveSnippetDrawer — 右侧抽屉式 panel, 保存 / 编辑 snippet.
     单字段 ref + setup time prefill — 跟 NuxtUI v-model 100% 兼容, 不依赖
     watch / mounted hook 时序 (父用 v-if + :key force remount 每次新打开). -->
<template>
  <transition name="drawer">
    <div v-if="open" class="drawer-mask" @click="close">
      <div class="drawer-panel bg-default" @click.stop>
        <div class="drawer-header">
          <UIcon name="i-tabler-bookmark-plus" class="size-5 text-primary" />
          <div class="flex-1">
            <div class="text-[13px] font-semibold text-default">
              {{ editingId ? t('editor.snippet.drawer.title_edit') : t('editor.snippet.drawer.title_new') }}
            </div>
            <div class="text-[10px] text-dimmed">
              {{ editingId ? `ID: ${editingId.slice(0, 8)}…` : t('editor.snippet.drawer.subtitle_source_kind', { kind: sourceKind || '?' }) }}
            </div>
          </div>
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-x"
            @click="close"
            :title="t('editor.snippet.drawer.close_tip')"
          />
        </div>

        <div class="drawer-body">
          <div class="field">
            <label>{{ t('editor.snippet.drawer.name_label') }} <span class="text-error">*</span></label>
            <UInput v-model="formName" size="sm" :placeholder="t('editor.snippet.drawer.name_placeholder')" class="w-full" />
          </div>

          <div class="field">
            <label>{{ t('editor.snippet.drawer.desc_label') }}</label>
            <UTextarea v-model="formDesc" size="sm" :rows="2" :placeholder="t('editor.snippet.drawer.desc_placeholder')" class="w-full" />
          </div>

          <div class="field">
            <label>{{ t('editor.snippet.drawer.cat_label') }}</label>
            <UInput
              v-model="formCategory"
              size="sm"
              :placeholder="t('editor.snippet.drawer.cat_placeholder')"
              class="w-full"
            />
            <div v-if="existingCategories.length > 0" class="flex flex-wrap gap-1 mt-1">
              <button
                v-for="c in existingCategories"
                :key="c"
                type="button"
                class="suggestion-chip"
                :class="formCategory === c ? 'is-active' : ''"
                @click="formCategory = formCategory === c ? '' : c"
              >{{ c }}</button>
            </div>
          </div>

          <div class="field">
            <label>{{ t('editor.snippet.drawer.tags_label') }}</label>
            <UInputTags
              v-model="formTags"
              size="sm"
              :placeholder="t('editor.snippet.drawer.tags_placeholder')"
              add-on-paste
              add-on-tab
              add-on-blur
              class="w-full"
            />
            <div v-if="allTags.length > 0" class="flex flex-wrap gap-1 mt-1">
              <span class="text-[10px] text-dimmed mr-1">{{ t('editor.snippet.drawer.tags_suggest') }}</span>
              <button
                v-for="tag in allTags"
                :key="tag"
                type="button"
                class="suggestion-chip"
                :class="formTags.includes(tag) ? 'is-active' : ''"
                @click="toggleTag(tag)"
              >#{{ tag }}</button>
            </div>
          </div>

          <div class="field">
            <label>{{ t('editor.snippet.drawer.color_label') }}</label>
            <div class="flex flex-wrap gap-1.5">
              <button
                v-for="c in colorPalette"
                :key="c.value"
                type="button"
                class="color-chip"
                :style="{ background: c.value }"
                :class="formColor === c.value ? 'is-selected' : ''"
                :title="t(c.labelKey)"
                @click="formColor = formColor === c.value ? undefined : c.value"
              />
            </div>
          </div>

          <div class="field">
            <label>{{ t('editor.snippet.drawer.icon_label') }}</label>
            <div class="flex flex-wrap gap-1">
              <button
                v-for="ic in iconPalette"
                :key="ic"
                type="button"
                class="icon-chip"
                :class="formIcon === ic ? 'is-selected' : ''"
                :title="ic"
                @click="formIcon = formIcon === ic ? undefined : ic"
              >
                <UIcon :name="ic" class="size-4" />
              </button>
            </div>
          </div>

          <div class="field">
            <label>{{ t('editor.snippet.drawer.shortcut_label') }}</label>
            <div class="flex gap-2">
              <UInput
                v-model="formShortcut"
                size="sm"
                :placeholder="t('editor.snippet.drawer.shortcut_placeholder')"
                class="flex-1"
                @keydown.escape.stop="formShortcut = ''"
              />
              <UButton
                :variant="capturing ? 'solid' : 'soft'"
                :color="capturing ? 'primary' : 'neutral'"
                size="sm"
                :icon="capturing ? 'i-tabler-keyboard' : 'i-tabler-target'"
                @click="toggleCapture"
                :title="capturing ? t('editor.snippet.drawer.shortcut_record') : t('editor.snippet.drawer.shortcut_idle')"
              />
            </div>
            <div v-if="shortcutError" class="text-[10px] text-error mt-1">⚠ {{ shortcutError }}</div>
            <div v-else-if="formShortcut" class="text-[10px] text-dimmed mt-1">
              normalize: <code class="text-primary">{{ normalizeShortcut(formShortcut) }}</code>
            </div>
          </div>
        </div>

        <div class="drawer-footer">
          <UButton
            v-if="editingId"
            size="sm"
            color="error"
            variant="ghost"
            icon="i-tabler-trash"
            @click="onDelete"
          >{{ t('editor.snippet.drawer.delete') }}</UButton>
          <div class="flex-1" />
          <UButton size="sm" variant="ghost" color="neutral" @click="close">{{ t('editor.snippet.drawer.cancel') }}</UButton>
          <UButton size="sm" color="primary" :disabled="!canSave" @click="onSave">{{ t('editor.snippet.drawer.save') }}</UButton>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  useSnippetsStore,
  normalizeShortcut,
  isReservedShortcut,
  eventToShortcutKey,
  type Snippet,
} from '@/stores/snippets'
import { useConfirm } from '@/composables/useConfirm'
import { PALETTE, PALETTE_KEYS } from '@/components/containers/visualRegistry'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  sourceKind?: string
  sourceConfig?: Record<string, unknown>
  editingId?: string
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  saved: [s: Snippet]
}>()

const store = useSnippetsStore()
store.load()

const { confirm: confirmDialog } = useConfirm()

// ===== DEBUG: 让用户 console 看 prefill 流程 =====
console.log('[SnippetDrawer] setup', {
  open: props.open,
  editingId: props.editingId,
  sourceKind: props.sourceKind,
  storeLoaded: store.loaded,
  storeSize: store.snippets.length,
  storeIDs: store.snippets.map((s) => s.id),
})

// 每字段单独 ref — 跟 NuxtUI v-model 100% 兼容. setup time prefill 一次,
// 之后用户输入直接改各 ref. 不需要 watch / Object.assign / reactive 体操.
const formName = ref('')
const formDesc = ref('')
const formCategory = ref('')
const formTags = ref<string[]>([])
const formColor = ref<string | undefined>(undefined)
const formIcon = ref<string | undefined>(undefined)
const formShortcut = ref('')

// setup time prefill — 父 v-if + :key 保证每次新打开 setup 重跑, props 已最终.
if (props.editingId) {
  const s = store.getById(props.editingId)
  console.log('[SnippetDrawer] prefill getById', props.editingId, '→', s)
  if (s) {
    formName.value = s.name
    formDesc.value = s.description ?? ''
    formCategory.value = s.category ?? ''
    formTags.value = [...s.tags]
    formColor.value = s.color
    formIcon.value = s.icon
    formShortcut.value = s.shortcut ?? ''
    console.log('[SnippetDrawer] filled', {
      name: formName.value,
      category: formCategory.value,
      tags: formTags.value,
      shortcut: formShortcut.value,
    })
  } else {
    console.warn('[SnippetDrawer] snippet not found in store, prefill skipped')
  }
} else {
  formName.value = props.sourceKind ?? ''
}

function toggleTag(t: string) {
  if (formTags.value.includes(t)) {
    formTags.value = formTags.value.filter((x) => x !== t)
  } else {
    formTags.value = [...formTags.value, t]
  }
}

const existingCategories = computed(() => store.allCategories)
const allTags = computed(() => store.allTags)

// 色板从视觉注册中心取 (单一真源); snippet 仍存 hex, value=hex.
const colorPalette = PALETTE_KEYS.map((k) => ({ labelKey: PALETTE[k].labelKey, value: PALETTE[k].hex }))

const iconPalette = [
  'i-tabler-bookmark', 'i-tabler-target', 'i-tabler-eye', 'i-tabler-mouse',
  'i-tabler-keyboard', 'i-tabler-clock', 'i-tabler-flag', 'i-tabler-bolt',
  'i-tabler-fish', 'i-tabler-sword', 'i-tabler-heart', 'i-tabler-coin',
  'i-tabler-package', 'i-tabler-settings', 'i-tabler-puzzle', 'i-tabler-wand',
]

const shortcutError = computed(() => {
  const s = formShortcut.value.trim()
  if (!s) return ''
  const norm = normalizeShortcut(s)
  if (isReservedShortcut(s)) return t('editor.snippet.drawer.err_reserved', { key: norm })
  const existing = store.byShortcut.get(norm)
  if (existing && existing.id !== props.editingId) {
    return t('editor.snippet.drawer.err_taken', { key: norm, name: existing.name })
  }
  return ''
})

const canSave = computed(() => formName.value.trim().length > 0 && !shortcutError.value)

const capturing = ref(false)
function toggleCapture() {
  capturing.value = !capturing.value
  if (capturing.value) window.addEventListener('keydown', onCaptureKey, true)
  else window.removeEventListener('keydown', onCaptureKey, true)
}
function onCaptureKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    capturing.value = false
    window.removeEventListener('keydown', onCaptureKey, true)
    return
  }
  if (['Control', 'Shift', 'Alt', 'Meta'].includes(e.key)) return
  e.preventDefault()
  e.stopPropagation()
  formShortcut.value = eventToShortcutKey(e)
  capturing.value = false
  window.removeEventListener('keydown', onCaptureKey, true)
}

function onSave() {
  if (!canSave.value) return
  const payload = props.editingId
    ? store.getById(props.editingId)?.payload
    : ({ type: 'node' as const, kind: props.sourceKind!, config: props.sourceConfig ?? {} })
  if (!payload) return

  const data = {
    name: formName.value.trim(),
    description: formDesc.value.trim() || undefined,
    category: formCategory.value.trim() || undefined,
    tags: formTags.value.map((t) => t.trim()).filter(Boolean),
    color: formColor.value,
    icon: formIcon.value,
    shortcut: formShortcut.value.trim() ? normalizeShortcut(formShortcut.value) : undefined,
    payload,
  }

  let saved: Snippet | null = null
  if (props.editingId) {
    saved = store.update(props.editingId, data)
  } else {
    saved = store.create({ ...data, lastUsedAt: undefined })
  }
  if (saved) emit('saved', saved)
  close()
}

async function onDelete() {
  if (!props.editingId) return
  const s = store.getById(props.editingId)
  const name = s?.name ?? t('editor.snippet.drawer.delete_dialog_fallback_name')
  const ok = await confirmDialog({
    title: t('editor.snippet.drawer.delete_dialog_title'),
    description: t('editor.snippet.drawer.delete_dialog_desc', { name }),
    confirmText: t('editor.snippet.drawer.delete'),
    cancelText: t('editor.snippet.drawer.cancel'),
    color: 'error',
  })
  if (!ok) return
  store.remove(props.editingId)
  close()
}

function close() {
  emit('update:open', false)
}
</script>

<style scoped>
.drawer-mask {
  position: fixed;
  inset: 0;
  z-index: 60;
  background: rgba(8, 8, 12, 0.65);
  backdrop-filter: blur(8px) saturate(120%);
  display: flex;
  justify-content: flex-end;
}
.drawer-panel {
  width: 400px;
  height: 100%;
  border-left: 1px solid var(--ui-border);
  display: flex;
  flex-direction: column;
  box-shadow: -20px 0 50px -10px rgba(0, 0, 0, 0.7);
  font-family:
    system-ui, -apple-system, 'Segoe UI Variable Text', 'PingFang SC', sans-serif;
}
.drawer-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--ui-border);
  background-image: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.04) 0%,
    transparent 60%
  );
}
.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field label {
  font-size: 11px;
  color: var(--ui-text-toned);
  font-weight: 500;
}
.drawer-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--ui-border);
  background: rgba(0, 0, 0, 0.2);
}

.suggestion-chip {
  font-size: 10.5px;
  padding: 2px 8px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--ui-text-dimmed);
  cursor: pointer;
  transition: all 120ms ease;
}
.suggestion-chip:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--ui-text-default);
}
.suggestion-chip.is-active {
  background: var(--ui-primary);
  border-color: var(--ui-primary);
  color: white;
}

.color-chip {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 120ms ease;
}
.color-chip:hover {
  transform: scale(1.1);
}
.color-chip.is-selected {
  border-color: rgba(255, 255, 255, 0.8);
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.2);
}

.icon-chip {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  color: var(--ui-text-toned);
  cursor: pointer;
  transition: all 120ms ease;
}
.icon-chip:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--ui-text-default);
}
.icon-chip.is-selected {
  background: var(--ui-primary);
  border-color: var(--ui-primary);
  color: white;
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 180ms ease;
}
.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}
.drawer-enter-active .drawer-panel,
.drawer-leave-active .drawer-panel {
  transition: transform 220ms cubic-bezier(0.4, 0, 0.2, 1);
}
.drawer-enter-from .drawer-panel,
.drawer-leave-to .drawer-panel {
  transform: translateX(100%);
}
</style>
