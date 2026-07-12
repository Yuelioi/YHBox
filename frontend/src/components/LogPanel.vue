<template>
  <div
    class="border-t border-default flex flex-col shrink-0 bg-default"
    :style="{ height: collapsed ? '28px' : '220px' }"
  >
    <!-- header -->
    <div
      class="h-7 px-2 flex items-center gap-2 border-b border-default shrink-0 select-none cursor-pointer hover:bg-elevated/40"
      @click="togglePanel"
    >
      <UIcon
        :name="collapsed ? 'i-tabler-chevron-up' : 'i-tabler-chevron-down'"
        class="size-3.5 text-dimmed"
      />
      <span class="text-[11px] text-toned font-medium">{{ t('log.header_title') }}</span>
      <span v-if="filteredLines.length" class="text-[10px] text-dimmed">{{
        t('log.count', { n: filteredLines.length })
      }}</span>
      <span v-if="hasErrors" class="text-[10px] text-error">{{ t('log.has_errors') }}</span>

      <div class="flex-1" />

      <!-- 写文件状态 icon -->
      <span
        v-if="!collapsed"
        :title="
          writeFile
            ? t('log.write_file_tooltip_on', { dir: fileDir })
            : t('log.write_file_tooltip_off')
        "
        class="size-2 rounded-full shrink-0"
        :class="writeFile ? 'bg-success' : 'bg-accented'"
        @click.stop
      />

      <!-- 双源 filter -->
      <div v-if="!collapsed" class="flex items-center gap-0.5 ml-1" @click.stop>
        <button
          v-for="opt in ['ALL', 'SYS', 'CTR'] as const"
          :key="opt"
          class="px-1.5 py-0.5 text-[10px] rounded transition-colors"
          :class="filter === opt ? 'bg-primary/15 text-primary' : 'text-dimmed hover:text-toned'"
          @click="filter = opt"
        >
          {{ opt }}
        </button>
      </div>

      <!-- 设置 popover (showTime/showTag/wrap/autoScroll/writeFile) -->
      <UButton
        v-if="!collapsed"
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-route"
        :title="t('log.action_trace.open')"
        :ui="{ base: 'h-5 px-1' }"
        @click.stop="actionTraceOpen = true"
      />

      <UPopover v-if="!collapsed" mode="click" :ui="{ content: 'p-2 w-48' }">
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-settings"
          :ui="{ base: 'h-5 px-1' }"
          @click.stop
        />
        <template #content>
          <div class="space-y-1 text-[11px]">
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="showTime"
                @change="toggleField('showTime', ($event.target as HTMLInputElement).checked)"
              />
              {{ t('log.popover.show_time') }}
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="showTag"
                @change="toggleField('showTag', ($event.target as HTMLInputElement).checked)"
              />
              {{ t('log.popover.show_tag') }}
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="wrapText"
                @change="toggleField('wrapText', ($event.target as HTMLInputElement).checked)"
              />
              {{ t('log.popover.wrap_text') }}
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="autoScroll"
                @change="toggleField('autoScroll', ($event.target as HTMLInputElement).checked)"
              />
              {{ t('log.popover.auto_scroll') }}
            </label>
            <hr class="border-default my-1" />
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="writeFile"
                @change="toggleField('writeFile', ($event.target as HTMLInputElement).checked)"
              />
              {{ t('log.popover.write_file') }}
            </label>
            <hr class="border-default my-1" />
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="showNodeEnter"
                @change="showNodeEnter = ($event.target as HTMLInputElement).checked"
              />
              {{ t('log.popover.show_node_enter') }}
            </label>
          </div>
        </template>
      </UPopover>

      <!-- clear -->
      <UButton
        v-if="!collapsed"
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-trash"
        :ui="{ base: 'h-5 px-1' }"
        @click.stop="logStore.clear()"
      />
    </div>

    <!-- body -->
    <div
      v-show="!collapsed"
      ref="bodyRef"
      class="flex-1 overflow-y-auto font-mono text-[11px] px-2 py-1 space-y-0.5 bg-sunken"
    >
      <div v-if="filteredLines.length === 0" class="text-dimmed italic">{{ t('log.empty') }}</div>
      <div
        v-for="(l, idx) in filteredLines"
        :key="idx"
        class="flex gap-2 leading-tight"
        :class="wrapText ? 'whitespace-pre-wrap wrap-break-word' : 'whitespace-nowrap'"
      >
        <span v-if="showTime" class="text-dimmed shrink-0 tabular-nums">{{
          fmtShortTime(l.time)
        }}</span>
        <span class="shrink-0 uppercase tracking-wide w-8" :class="sourceClass(l.source)">{{
          l.source
        }}</span>
        <span class="shrink-0 uppercase tracking-wide w-12" :class="levelClass(l.level)">{{
          l.level
        }}</span>
        <span class="text-default break-all"
          >{{ l.message
          }}<span v-if="(l.count ?? 1) > 1" class="text-dimmed"> ×{{ l.count }}</span></span
        >
      </div>
    </div>

    <ActionTraceDrawer v-model:open="actionTraceOpen" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ActionTraceDrawer from '@/components/ActionTraceDrawer.vue'
import { useLogStore } from '@/stores/log'
import { useSettingsStore } from '@/stores/settings'

const { t } = useI18n()
const logStore = useLogStore()
const settingsStore = useSettingsStore()

const filter = ref<'ALL' | 'SYS' | 'CTR'>('ALL')
const bodyRef = ref<HTMLDivElement | null>(null)
const actionTraceOpen = ref(false)

const collapsed = computed({
  get: () => !(settingsStore.data?.ui.logger.panelOpen ?? true),
  set: (v) => settingsStore.patch({ ui: { logger: { panelOpen: !v } } }),
})

function togglePanel() {
  collapsed.value = !collapsed.value
}

const showTime = computed(() => settingsStore.data?.ui.logger.showTime ?? true)
const showTag = computed(() => settingsStore.data?.ui.logger.showTag ?? true)
const wrapText = computed(() => settingsStore.data?.ui.logger.wrapText ?? false)
const autoScroll = computed(() => settingsStore.data?.ui.logger.autoScroll ?? true)
const writeFile = computed(() => settingsStore.data?.ui.logger.writeFile ?? true)
const fileDir = computed(() => settingsStore.data?.ui.logger.fileDir ?? 'logs')

const showNodeEnter = computed({
  get: () => settingsStore.data?.ui.logger.showNodeEnter ?? false,
  set: (v: boolean) => settingsStore.patch({ ui: { logger: { showNodeEnter: v } } }),
})

function toggleField(field: string, v: boolean) {
  settingsStore.patch({ ui: { logger: { [field]: v } } })
}

const filteredLines = computed(() => {
  let ls = logStore.lines
  if (!showNodeEnter.value) ls = ls.filter((l) => l.level !== 'node')
  if (filter.value !== 'ALL') ls = ls.filter((l) => l.source === filter.value)
  return ls
})

const hasErrors = computed(() =>
  filteredLines.value.some((l) => l.level === 'error' || l.level === 'warn'),
)

function sourceClass(s: string) {
  return s === 'SYS' ? 'text-cyan-400' : 'text-violet-400'
}

function levelClass(level: string) {
  switch (level) {
    case 'error':
      return 'text-error'
    case 'warn':
      return 'text-warning'
    case 'debug':
      return 'text-dimmed'
    // node/dump/log 是日志流身份色 (区分流, 非状态语义), 不走 semantic
    case 'node':
      return 'text-violet-300'
    case 'dump':
      return 'text-emerald-300'
    case 'log':
      return 'text-emerald-400'
    case 'action':
      return 'text-sky-300'
    default:
      return 'text-info'
  }
}

function fmtShortTime(iso: string) {
  const d = new Date(iso)
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return `${hh}:${mm}:${ss}.${ms}`
}

// auto-scroll to bottom on new lines
watch(
  () => logStore.lines.length,
  () => {
    if (!autoScroll.value) return
    void nextTick(() => {
      if (bodyRef.value) bodyRef.value.scrollTop = bodyRef.value.scrollHeight
    })
  },
)
</script>
