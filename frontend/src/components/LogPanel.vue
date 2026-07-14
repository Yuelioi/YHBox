<template>
  <div
    class="flex shrink-0 flex-col border-t border-default bg-default"
    :style="{ height: collapsed ? '32px' : 'clamp(180px, 28vh, 320px)' }"
  >
    <!-- header -->
    <div class="flex h-8 shrink-0 select-none items-center border-b border-default px-2">
      <button
        type="button"
        class="flex h-full min-w-0 flex-1 items-center gap-2 rounded px-1 text-left hover:bg-elevated/40"
        :aria-expanded="!collapsed"
        aria-controls="app-log-panel-body"
        @click="togglePanel"
      >
        <UIcon
          :name="collapsed ? 'i-tabler-chevron-up' : 'i-tabler-chevron-down'"
          class="size-3.5 shrink-0 text-dimmed"
        />
        <span class="text-xs font-medium text-toned">{{ t('log.header_title') }}</span>
        <span v-if="filteredLines.length" class="text-xs tabular-nums text-dimmed">{{
          t('log.count', { n: filteredLines.length })
        }}</span>
        <span v-if="hasErrors" class="text-xs text-error">{{ t('log.has_errors') }}</span>
        <span v-if="logStore.dropped" class="text-[11px] tabular-nums text-warning">
          {{ t('log.dropped', { n: logStore.dropped }) }}
        </span>
      </button>

      <div v-if="!collapsed" class="ml-2 flex shrink-0 items-center gap-1">
        <UButton
          size="xs"
          variant="ghost"
          :color="enabled ? 'primary' : 'neutral'"
          :icon="enabled ? 'i-tabler-activity' : 'i-tabler-activity-off'"
          :title="enabled ? t('log.disable') : t('log.enable')"
          :aria-label="enabled ? t('log.disable') : t('log.enable')"
          :aria-pressed="enabled"
          :ui="{ base: 'size-7 p-0' }"
          @click="toggleField('enabled', !enabled)"
        />

        <!-- 实际落盘状态 -->
        <span
          role="status"
          :aria-label="
            enabled && writeFile
              ? t('log.write_file_tooltip_on', { dir: fileDir })
              : t('log.write_file_tooltip_off')
          "
          :title="
            enabled && writeFile
              ? t('log.write_file_tooltip_on', { dir: fileDir })
              : t('log.write_file_tooltip_off')
          "
          class="mx-1 size-2 shrink-0 rounded-full"
          :class="enabled && writeFile ? 'bg-success' : 'bg-accented'"
        />

        <!-- 双源 filter -->
        <div class="flex items-center gap-0.5" role="group" :aria-label="t('log.filter_label')">
          <button
            v-for="opt in ['ALL', 'SYS', 'CTR'] as const"
            :key="opt"
            type="button"
            class="rounded px-1.5 py-1 text-[11px] transition-colors"
            :class="filter === opt ? 'bg-primary/15 text-primary' : 'text-dimmed hover:text-toned'"
            :aria-pressed="filter === opt"
            :aria-label="t(`log.filter_${opt.toLowerCase()}`)"
            @click="filter = opt"
          >
            {{ opt }}
          </button>
        </div>

        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-route"
          :title="t('log.action_trace.open')"
          :aria-label="t('log.action_trace.open')"
          :ui="{ base: 'size-7 p-0' }"
          @click="actionTraceOpen = true"
        />

        <!-- 设置 popover (showTime/showTag/wrap/autoScroll/writeFile) -->
        <UPopover mode="click" :ui="{ content: 'w-60 p-2.5' }">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-settings"
            :title="t('log.settings')"
            :aria-label="t('log.settings')"
            :ui="{ base: 'size-7 p-0' }"
          />
          <template #content>
            <div class="text-[11px] leading-4 text-toned">
              <div
                class="flex items-center justify-between gap-3 rounded-md bg-elevated/60 px-2 py-1.5"
              >
                <div class="min-w-0">
                  <div class="font-medium text-highlighted">{{ t('log.popover.enabled') }}</div>
                  <div class="text-[10px] leading-3.5 text-dimmed">
                    {{ t('log.popover.enabled_hint') }}
                  </div>
                </div>
                <USwitch
                  size="xs"
                  :model-value="enabled"
                  :aria-label="t('log.popover.enabled')"
                  @update:model-value="toggleField('enabled', $event)"
                />
              </div>

              <div class="mt-1 space-y-0.5">
                <div
                  class="flex min-h-6 items-center justify-between gap-3 rounded px-1.5"
                  :class="{ 'opacity-50': !enabled }"
                >
                  <span>{{ t('log.popover.live_view') }}</span>
                  <USwitch
                    size="xs"
                    :model-value="liveView"
                    :disabled="!enabled"
                    :aria-label="t('log.popover.live_view')"
                    @update:model-value="toggleField('liveView', $event)"
                  />
                </div>
                <div
                  class="flex min-h-7 items-center justify-between gap-3 rounded px-1.5"
                  :class="{ 'opacity-50': !enabled }"
                >
                  <span>{{ t('log.popover.level') }}</span>
                  <USelect
                    size="xs"
                    class="w-22"
                    :ui="{ base: 'text-[11px]' }"
                    :model-value="level"
                    :items="logLevelItems"
                    :disabled="!enabled"
                    :aria-label="t('log.popover.level')"
                    @update:model-value="toggleField('level', $event)"
                  />
                </div>
              </div>

              <div class="my-1 border-t border-default/70" />

              <div class="space-y-0.5">
                <div class="flex min-h-6 items-center justify-between gap-3 rounded px-1.5">
                  <span>{{ t('log.popover.show_time') }}</span>
                  <USwitch
                    size="xs"
                    :model-value="showTime"
                    :aria-label="t('log.popover.show_time')"
                    @update:model-value="toggleField('showTime', $event)"
                  />
                </div>
                <div class="flex min-h-6 items-center justify-between gap-3 rounded px-1.5">
                  <span>{{ t('log.popover.show_tag') }}</span>
                  <USwitch
                    size="xs"
                    :model-value="showTag"
                    :aria-label="t('log.popover.show_tag')"
                    @update:model-value="toggleField('showTag', $event)"
                  />
                </div>
                <div class="flex min-h-6 items-center justify-between gap-3 rounded px-1.5">
                  <span>{{ t('log.popover.wrap_text') }}</span>
                  <USwitch
                    size="xs"
                    :model-value="wrapText"
                    :aria-label="t('log.popover.wrap_text')"
                    @update:model-value="toggleField('wrapText', $event)"
                  />
                </div>
                <div class="flex min-h-6 items-center justify-between gap-3 rounded px-1.5">
                  <span>{{ t('log.popover.auto_scroll') }}</span>
                  <USwitch
                    size="xs"
                    :model-value="autoScroll"
                    :aria-label="t('log.popover.auto_scroll')"
                    @update:model-value="toggleField('autoScroll', $event)"
                  />
                </div>
              </div>

              <div class="my-1 border-t border-default/70" />

              <div class="space-y-0.5">
                <div
                  class="flex min-h-6 items-center justify-between gap-3 rounded px-1.5"
                  :class="{ 'opacity-50': !enabled }"
                >
                  <span>{{ t('log.popover.write_file') }}</span>
                  <USwitch
                    size="xs"
                    :model-value="writeFile"
                    :disabled="!enabled"
                    :aria-label="t('log.popover.write_file')"
                    @update:model-value="toggleField('writeFile', $event)"
                  />
                </div>
                <div class="flex min-h-6 items-center justify-between gap-3 rounded px-1.5">
                  <span>{{ t('log.popover.show_node_enter') }}</span>
                  <USwitch
                    size="xs"
                    :model-value="showNodeEnter"
                    :aria-label="t('log.popover.show_node_enter')"
                    @update:model-value="showNodeEnter = $event"
                  />
                </div>
              </div>
            </div>
          </template>
        </UPopover>

        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-trash"
          :title="t('log.clear')"
          :aria-label="t('log.clear')"
          :ui="{ base: 'size-7 p-0' }"
          @click="logStore.clear()"
        />
      </div>
    </div>

    <!-- body -->
    <div
      v-show="!collapsed"
      id="app-log-panel-body"
      ref="bodyRef"
      class="flex-1 space-y-0.5 overflow-y-auto bg-sunken px-2 py-1 font-mono text-xs"
    >
      <div
        v-if="!enabled"
        class="flex items-center justify-center text-dimmed"
        :class="
          filteredLines.length
            ? 'sticky top-0 z-10 border-b border-default bg-sunken/95 py-2'
            : 'h-full'
        "
      >
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-activity-off" class="size-4" />
          <span>{{ t('log.disabled') }}</span>
        </div>
      </div>
      <div
        v-else-if="!liveView"
        class="flex items-center justify-center text-dimmed"
        :class="
          filteredLines.length
            ? 'sticky top-0 z-10 border-b border-default bg-sunken/95 py-2'
            : 'h-full'
        "
      >
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-player-pause" class="size-4" />
          <span>{{ t('log.live_paused') }}</span>
        </div>
      </div>
      <div v-else-if="filteredLines.length === 0" class="text-dimmed italic">
        {{ t('log.empty') }}
      </div>
      <div
        v-for="(l, idx) in filteredLines"
        :key="l.id ?? idx"
        class="flex gap-2 leading-tight"
        :class="wrapText ? 'whitespace-pre-wrap wrap-break-word' : 'whitespace-nowrap'"
      >
        <span v-if="showTime" class="text-dimmed shrink-0 tabular-nums">{{
          fmtShortTime(l.time)
        }}</span>
        <span
          v-if="showTag"
          class="shrink-0 uppercase tracking-wide w-8"
          :class="sourceClass(l.source)"
          >{{ l.source }}</span
        >
        <span
          v-if="showTag"
          class="shrink-0 uppercase tracking-wide w-12"
          :class="levelClass(l.level)"
          >{{ l.level }}</span
        >
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
const logLevelItems = [
  { label: 'DEBUG', value: 'debug' },
  { label: 'INFO', value: 'info' },
  { label: 'WARN', value: 'warn' },
  { label: 'ERROR', value: 'error' },
]

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
const enabled = computed(() => settingsStore.data?.ui.logger.enabled ?? true)
const liveView = computed(() => settingsStore.data?.ui.logger.liveView ?? true)
const level = computed(() => settingsStore.data?.ui.logger.level ?? 'info')
const writeFile = computed(() => settingsStore.data?.ui.logger.writeFile ?? true)
const fileDir = computed(() => settingsStore.data?.ui.logger.fileDir ?? 'logs')

const showNodeEnter = computed({
  get: () => settingsStore.data?.ui.logger.showNodeEnter ?? false,
  set: (v: boolean) => settingsStore.patch({ ui: { logger: { showNodeEnter: v } } }),
})

function toggleField(field: string, v: boolean | string) {
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
