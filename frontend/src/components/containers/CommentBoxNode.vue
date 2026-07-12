<template>
  <!-- CommentBox — visual-only 注释节点, 仿普通节点卡片 (彩色标题栏 + 半透明 body), 无 handle.
       颜色/图标从视觉注册中心取 (config.literal.Color = palette key, Icon = tabler 名).
       双击标题/正文内联编辑; 颜色/图标在 NodeInspector 选. body 加 .nodrag → 只标题栏可拖. -->
  <div
    class="cb-card"
    :class="{ 'is-selected': selected }"
    :style="cardStyle"
    @dblclick.stop="enterEdit"
  >
    <!-- 标题栏 = 拖拽手柄 (非 .nodrag) -->
    <div class="cb-header" :style="headerStyle">
      <span class="cb-accent" :style="{ background: hex }" />
      <UIcon :name="icon" class="cb-icon shrink-0" :style="{ color: hex }" />
      <input
        v-if="editing"
        ref="titleEl"
        v-model="draftTitle"
        class="nodrag cb-title-input"
        :placeholder="t('commentBox.title_placeholder')"
        @mousedown.stop
        @keydown.esc.prevent="save"
        @keydown.enter.prevent="save"
      />
      <span v-else class="cb-title truncate" :class="{ 'opacity-50': !title }">
        {{ title || t('commentBox.title_placeholder') }}
      </span>
    </div>

    <!-- 正文 body (.nodrag 不参与拖拽) -->
    <div class="cb-body nodrag nowheel">
      <textarea
        v-if="editing"
        v-model="draftContent"
        class="cb-textarea"
        :rows="contentRows"
        :placeholder="t('commentBox.content_placeholder')"
        @mousedown.stop
        @keydown.esc.prevent="save"
        @keydown.enter.ctrl.prevent="save"
        @blur="save"
      />
      <!-- eslint-disable-next-line vue/no-v-html — markdown-it html:false 输出无原始 HTML, 安全 -->
      <div v-else class="comment-md" @click="onContentClick" v-html="renderedContent" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Browser } from '@wailsio/runtime'
import { ContainerCanvasApiKey } from '@/composables/containerEditor/pinLiterals'
import { renderMarkdown } from '@/lib/markdown'
import { resolvePalette, DEFAULT_ICON, DEFAULT_PALETTE_KEY } from './visualRegistry'

const { t } = useI18n()

const props = defineProps<{
  id: string
  data: { kind: string; config?: Record<string, any> }
  selected?: boolean
}>()

const canvasApi = inject(ContainerCanvasApiKey, null)

const literal = computed<Record<string, any>>(() => props.data?.config?.literal ?? {})
const title = computed<string>(() => String(literal.value.Title ?? ''))
const content = computed<string>(() => String(literal.value.Content ?? ''))
const colorKey = computed<string>(() => String(literal.value.Color ?? DEFAULT_PALETTE_KEY))
const icon = computed<string>(() => String(literal.value.Icon ?? DEFAULT_ICON))
const hex = computed(() => resolvePalette(colorKey.value).hex)

const renderedContent = computed(() => renderMarkdown(content.value))

const width = computed<number>(() => {
  const w = Number(literal.value.Width)
  return Number.isFinite(w) && w > 0 ? Math.max(160, w) : 280
})

const cardStyle = computed(() => ({
  width: width.value + 'px',
  background: hex.value + '1f', // ~12% 暗底色调, 仿普通节点
  borderColor: hex.value + '66',
}))
const headerStyle = computed(() => ({
  background: hex.value + '33', // ~20% 标题栏更浓
}))

// ── 编辑态 (标题 + 正文; 颜色/图标走 inspector) ─────────────
const editing = ref(false)
const draftTitle = ref('')
const draftContent = ref('')
const titleEl = ref<HTMLInputElement | null>(null)

const contentRows = computed(() =>
  Math.min(20, Math.max(3, (draftContent.value.match(/\n/g)?.length ?? 0) + 2)),
)

async function enterEdit() {
  draftTitle.value = title.value
  draftContent.value = content.value
  editing.value = true
  await nextTick()
  titleEl.value?.focus()
}

function save() {
  if (!editing.value) return
  editing.value = false
  canvasApi?.patchNodeLiteral(props.id, {
    Title: draftTitle.value,
    Content: draftContent.value,
  })
}

// 渲染态: 拦截 markdown 链接 → 外部浏览器, 别在 webview 内导航跳走.
function onContentClick(e: MouseEvent) {
  const a = (e.target as HTMLElement).closest('a')
  if (a && a.getAttribute('href')) {
    e.preventDefault()
    Browser.OpenURL(a.href)
  }
}
</script>

<style scoped>
.cb-card {
  position: relative;
  border-radius: 12px;
  border: 1px solid;
  overflow: hidden;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.4);
  font-size: 12px;
  color: #e4e4e7;
  backdrop-filter: blur(2px);
}
.cb-card.is-selected {
  box-shadow:
    0 0 0 2px rgba(255, 255, 255, 0.7),
    0 2px 12px rgba(0, 0, 0, 0.5);
}

/* 标题栏 = 拖拽手柄 */
.cb-header {
  position: relative;
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 7px 11px 7px 13px;
  cursor: move;
  font-weight: 700;
  font-size: 13px;
}
.cb-accent {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
}
.cb-icon {
  width: 16px;
  height: 16px;
  filter: drop-shadow(0 0 4px currentColor);
}
.cb-title {
  flex: 1;
  min-width: 0;
  color: #f4f4f5;
}
.cb-title-input {
  flex: 1;
  min-width: 0;
  background: transparent;
  border: none;
  outline: none;
  font-weight: 700;
  font-size: 13px;
  color: #f4f4f5;
}
.cb-title-input::placeholder {
  opacity: 0.5;
}

/* 正文 body */
.cb-body {
  padding: 8px 11px 10px;
  cursor: text;
}
.cb-textarea {
  width: 100%;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 5px;
  outline: none;
  resize: none;
  padding: 5px 7px;
  font-size: 12px;
  line-height: 1.45;
  font-family: ui-monospace, SFMono-Regular, Menlo, 'Microsoft YaHei', 'PingFang SC', monospace;
  color: #e4e4e7;
}

/* markdown 排版 (scoped, 暗底) */
.comment-md {
  line-height: 1.5;
  word-break: break-word;
  color: #d4d4d8;
}
.comment-md :deep(p) {
  margin: 0 0 0.4em;
}
.comment-md :deep(p:last-child) {
  margin-bottom: 0;
}
.comment-md :deep(ul),
.comment-md :deep(ol) {
  margin: 0 0 0.4em;
  padding-left: 1.2em;
}
.comment-md :deep(h1),
.comment-md :deep(h2),
.comment-md :deep(h3) {
  font-weight: 700;
  margin: 0.3em 0 0.2em;
  line-height: 1.3;
  color: #f4f4f5;
}
.comment-md :deep(h1) {
  font-size: 1.25em;
}
.comment-md :deep(h2) {
  font-size: 1.12em;
}
.comment-md :deep(code) {
  background: rgba(255, 255, 255, 0.12);
  padding: 0 0.25em;
  border-radius: 3px;
  font-size: 0.9em;
}
.comment-md :deep(pre) {
  background: rgba(0, 0, 0, 0.3);
  padding: 0.5em;
  border-radius: 5px;
  overflow-x: auto;
  margin: 0 0 0.4em;
}
.comment-md :deep(pre code) {
  background: none;
  padding: 0;
}
.comment-md :deep(a) {
  color: #93c5fd;
  text-decoration: underline;
  cursor: pointer;
  font-weight: 600;
}
.comment-md :deep(blockquote) {
  border-left: 3px solid rgba(255, 255, 255, 0.3);
  padding-left: 0.6em;
  opacity: 0.85;
  margin: 0 0 0.4em;
}
.comment-md :deep(hr) {
  border: none;
  border-top: 1px solid rgba(255, 255, 255, 0.15);
  margin: 0.5em 0;
}
</style>
