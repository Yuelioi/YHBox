<template>
  <div class="flex flex-col h-full bg-default border-t border-default">
    <!-- Header bar -->
    <div
      class="h-8 shrink-0 flex items-center justify-between px-3 border-b border-default/60 bg-default"
    >
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-terminal" class="size-3.5 text-dimmed" />
        <span class="text-xs font-medium text-muted">日志</span>
        <span
          class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-elevated text-muted"
        >
          {{ logStore.lines.length }} / 500
        </span>
      </div>
      <div class="flex items-center gap-0.5">
        <!-- Show time toggle -->
        <button
          class="flex items-center justify-center size-6 rounded text-xs transition-colors duration-150"
          :class="
            showTime
              ? 'text-indigo-400 bg-indigo-500/10'
              : 'text-dimmed hover:text-toned hover:bg-elevated'
          "
          title="显示时间戳"
          @click="toggleShowTime"
        >
          <UIcon name="i-tabler-clock" class="size-3.5" />
        </button>
        <!-- Show tag toggle -->
        <button
          class="flex items-center justify-center size-6 rounded text-xs transition-colors duration-150"
          :class="
            showTag
              ? 'text-indigo-400 bg-indigo-500/10'
              : 'text-dimmed hover:text-toned hover:bg-elevated'
          "
          title="显示标签"
          @click="toggleShowTag"
        >
          <UIcon name="i-tabler-tag" class="size-3.5" />
        </button>
        <!-- Wrap toggle -->
        <button
          class="flex items-center justify-center size-6 rounded text-xs transition-colors duration-150"
          :class="
            wrapText
              ? 'text-indigo-400 bg-indigo-500/10'
              : 'text-dimmed hover:text-toned hover:bg-elevated'
          "
          title="切换折行"
          @click="toggleWrap"
        >
          <UIcon name="i-tabler-text-wrap" class="size-3.5" />
        </button>
        <!-- Auto-scroll toggle -->
        <button
          class="flex items-center justify-center size-6 rounded text-xs transition-colors duration-150"
          :class="
            autoScroll
              ? 'text-indigo-400 bg-indigo-500/10'
              : 'text-dimmed hover:text-toned hover:bg-elevated'
          "
          title="切换自动滚动"
          @click="toggleAutoScroll"
        >
          <UIcon name="i-tabler-arrow-bar-to-down" class="size-3.5" />
        </button>
        <!-- Clear -->
        <button
          class="flex items-center justify-center size-6 rounded text-dimmed hover:text-error hover:bg-error/10 transition-colors duration-150"
          title="清空日志"
          @click="logStore.clear()"
        >
          <UIcon name="i-tabler-trash" class="size-3.5" />
        </button>
      </div>
    </div>

    <!-- Log body -->
    <div
      v-if="logStore.lines.length > 0"
      v-bind="containerProps"
      class="flex-1 overflow-auto"
      @scroll.passive="onScroll"
    >
      <div v-bind="wrapperProps">
        <div
          v-for="item in list"
          :key="item.index"
          class="flex items-start px-3 py-0.5 leading-5 hover:bg-elevated/60 group"
          :class="wrapText ? 'whitespace-pre-wrap wrap-break-word' : 'whitespace-nowrap truncate'"
        >
          <span
            v-if="showTime"
            class="shrink-0 mr-2 text-muted tabular-nums font-mono text-[11px] select-none"
            >{{ fmtTs(item.data.time) }}</span
          >
          <span
            v-if="showTag && item.data.tag"
            class="shrink-0 mr-2 inline-flex items-center px-1.5 rounded text-[10px] font-semibold uppercase select-none tracking-wide h-4 mt-0.5"
            :class="tagClass(item.data.tag)"
            >{{ item.data.tag }}</span
          >
          <span class="min-w-0 font-mono text-[12px]" :class="levelClass(item.data.level)">{{
            item.data.message
          }}</span>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="flex-1 flex flex-col items-center justify-center gap-2 text-dimmed">
      <UIcon name="i-tabler-terminal" class="size-6" />
      <span class="text-xs">暂无日志</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, nextTick } from 'vue'
import { useVirtualList } from '@vueuse/core'
import { useLogStore } from '@/stores/log'
import { useSettingsStore } from '@/stores/settings'
import { fmtTs } from '@/lib/format'
import { tagClass, levelClass } from '@/lib/logFormat'

const STICKY_BOTTOM_THRESHOLD = 24

const logStore = useLogStore()
const settingsStore = useSettingsStore()

const showTime = computed(() => settingsStore.data?.ui.logger.showTime ?? true)
const showTag = computed(() => settingsStore.data?.ui.logger.showTag ?? true)
const wrapText = computed(() => settingsStore.data?.ui.logger.wrapText ?? false)
const autoScroll = computed(() => settingsStore.data?.ui.logger.autoScroll ?? true)

const { list, containerProps, wrapperProps, scrollTo } = useVirtualList(
  computed(() => logStore.lines),
  { itemHeight: 18, overscan: 10 },
)

let userScrolledUp = false

function onScroll() {
  const el = containerProps.ref.value
  if (!el) return
  const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
  userScrolledUp = distFromBottom > STICKY_BOTTOM_THRESHOLD
}

watch(
  () => logStore.lines.length,
  () => {
    if (!autoScroll.value) return
    if (userScrolledUp) return
    nextTick(() => {
      scrollTo(logStore.lines.length - 1)
    })
  },
)

// 用户勾选"自动滚动"那一刻立刻滚到底（之前要等下一行日志到来才滚 — 不直观）
// 同时重置 userScrolledUp，让自动滚动恢复
watch(autoScroll, (v) => {
  if (!v) return
  userScrolledUp = false
  nextTick(() => {
    if (logStore.lines.length > 0) {
      scrollTo(logStore.lines.length - 1)
    }
  })
})

function toggleWrap() {
  settingsStore.patch({ ui: { logger: { wrapText: !wrapText.value } } })
}

function toggleAutoScroll() {
  settingsStore.patch({ ui: { logger: { autoScroll: !autoScroll.value } } })
}

function toggleShowTime() {
  settingsStore.patch({ ui: { logger: { showTime: !showTime.value } } })
}

function toggleShowTag() {
  settingsStore.patch({ ui: { logger: { showTag: !showTag.value } } })
}
</script>
