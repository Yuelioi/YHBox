<template>
  <!-- VSCode 风格底部日志面板: 折叠态 28px 只显示 header; 展开 200px 内嵌滚动列表.
       header 用 bg-default 跟编辑器其他面板对齐, body 黑底保留 (日志可读性高). -->
  <div
    class="border-t border-default flex flex-col shrink-0"
    :style="{ height: collapsed ? '28px' : '200px' }"
  >
    <!-- header -->
    <div
      class="h-7 px-2 flex items-center gap-2 border-b border-default shrink-0 select-none cursor-pointer bg-default hover:bg-elevated/40"
      @click="collapsed = !collapsed"
    >
      <UIcon
        :name="collapsed ? 'i-tabler-chevron-up' : 'i-tabler-chevron-down'"
        class="size-3.5 text-dimmed"
      />
      <span class="text-[11px] text-toned font-medium">运行日志</span>
      <span v-if="lines.length" class="text-[10px] text-dimmed">{{ lines.length }} 条</span>
      <span v-if="hasErrors" class="text-[10px] text-error">含错误</span>
      <div class="flex-1" />
      <UButton
        v-if="!collapsed"
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-trash"
        :ui="{ base: 'h-5 px-1.5 text-[10px]' }"
        @click.stop="clearLines"
      >
        清空
      </UButton>
    </div>

    <!-- body: 滚动列表. 折叠时 hidden 不占高 (但 height 已收 28 不需要 v-if). -->
    <div
      v-show="!collapsed"
      ref="bodyRef"
      class="flex-1 overflow-y-auto font-mono text-[11px] px-2 py-1 space-y-0.5 bg-zinc-950"
    >
      <div v-if="lines.length === 0" class="text-dimmed italic">无日志. ▶ 试运行后这里会显示节点执行 / PlayClip / Log / Toast.</div>
      <div
        v-for="(l, idx) in lines"
        :key="idx"
        class="flex gap-2 leading-tight"
      >
        <span class="text-dimmed shrink-0">{{ l.ts }}</span>
        <span
          class="shrink-0 uppercase tracking-wide w-12"
          :class="levelClass(l.level)"
        >{{ l.level }}</span>
        <span class="text-default break-all">{{ l.message }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'

interface LogLine {
  ts: string
  level: string
  message: string
}

const collapsed = ref(false)
const lines = ref<LogLine[]>([])
const bodyRef = ref<HTMLDivElement | null>(null)

const MAX_LINES = 500

const hasErrors = computed(() => lines.value.some((l) => l.level === 'error' || l.level === 'warn'))

function levelClass(level: string): string {
  switch (level) {
    case 'error':
      return 'text-error'
    case 'warn':
      return 'text-amber-400'
    case 'debug':
      return 'text-dimmed'
    case 'node':
      return 'text-violet-400'
    default:
      return 'text-blue-400'
  }
}

function formatTs(): string {
  const d = new Date()
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return `${hh}:${mm}:${ss}.${ms}`
}

function appendLine(level: string, message: string) {
  lines.value.push({ ts: formatTs(), level, message })
  if (lines.value.length > MAX_LINES) {
    lines.value.splice(0, lines.value.length - MAX_LINES)
  }
  // 自动滚到底
  void nextTick(() => {
    if (bodyRef.value) bodyRef.value.scrollTop = bodyRef.value.scrollHeight
  })
}

function clearLines() {
  lines.value = []
}

let unsubscribeLog: (() => void) | null = null
let unsubscribeNode: (() => void) | null = null

onMounted(() => {
  // container:log payload: {level, message}
  unsubscribeLog = Events.On('container:log', (e: any) => {
    const payload = e?.data?.[0] ?? e?.data ?? e
    appendLine(String(payload?.level ?? 'info'), String(payload?.message ?? ''))
  }) as unknown as () => void

  // container:node-enter-batch payload: {entries: [{nodeId, nodeKind, count}, ...]}
  // 后端 200ms 累积一批, 连续同 nodeId 折叠成 count (console.log × N 风格).
  // 兼容单条 container:node-enter (不被 batch 的低频路径).
  unsubscribeNode = Events.On('container:node-enter-batch', (e: any) => {
    const payload = e?.data?.[0] ?? e?.data ?? e
    const entries = payload?.entries
    if (!Array.isArray(entries)) return
    for (const ent of entries) {
      const kind = ent?.nodeKind ?? '?'
      const id = ent?.nodeId ?? '?'
      const cnt = Number(ent?.count ?? 1)
      const suffix = cnt > 1 ? ` × ${cnt}` : ''
      appendLine('node', `→ ${kind} (${id})${suffix}`)
    }
  }) as unknown as () => void
})

onUnmounted(() => {
  unsubscribeLog?.()
  unsubscribeNode?.()
})
</script>
