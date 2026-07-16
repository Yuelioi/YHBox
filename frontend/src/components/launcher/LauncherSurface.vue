<template>
  <div
    class="launcher-surface"
    :class="[`launcher-surface--${display}`, { 'launcher-surface--preview': preview }]"
  >
    <template v-if="groups.length">
      <section v-for="group in groups" :key="group.id" class="launcher-surface__group">
        <div v-if="group.label" class="launcher-surface__heading">
          <span>{{ group.label }}</span>
          <span>{{ group.items.length }}</span>
        </div>
        <div class="launcher-surface__items">
          <UButton
            v-for="item in group.items"
            :key="item.id"
            color="neutral"
            variant="ghost"
            class="launcher-command"
            :class="[
              `launcher-command--${statusFor(item.workflowId)}`,
              {
                'launcher-command--selected': selectedId === item.workflowId,
                'launcher-command--stale': item.stale,
                'launcher-command--separator-before': item.separatorBefore === 'vertical',
              },
            ]"
            :title="item.label"
            :aria-label="item.stale ? `${item.label}: ${staleLabel}` : runLabel(item.label)"
            :aria-current="selectedId === item.workflowId ? 'true' : undefined"
            :aria-disabled="item.stale ? 'true' : undefined"
            :tabindex="preview ? -1 : 0"
            @mouseenter="!item.stale && emit('select', item.workflowId)"
            @focus="!item.stale && emit('select', item.workflowId)"
            @click="!preview && !item.stale && emit('run', item.workflowId)"
          >
            <span v-if="display !== 'text'" class="launcher-command__icon">
              <UIcon
                :name="statusIcon(item.workflowId, item.icon)"
                class="size-4"
                :class="{ 'animate-spin': statusFor(item.workflowId) === 'running' }"
              />
            </span>
            <span v-if="display !== 'icon'" class="launcher-command__copy">
              <span class="launcher-command__label">{{ item.label }}</span>
              <span v-if="statusText(item)" class="launcher-command__status">
                {{ statusText(item) }}
              </span>
            </span>
            <span v-if="display !== 'icon'" class="launcher-command__meta">
              <UKbd v-if="item.shortcut" :value="item.shortcut" />
              <UKbd
                v-else-if="item.ordinal > 0 && item.ordinal <= 9"
                :value="String(item.ordinal)"
              />
              <UIcon v-else name="i-tabler-chevron-right" class="size-3 text-dimmed" />
            </span>
            <UKbd
              v-else-if="item.ordinal > 0 && item.ordinal <= 9"
              class="launcher-command__ordinal"
              :value="String(item.ordinal)"
            />
          </UButton>
        </div>
      </section>
    </template>

    <div v-else class="launcher-surface__empty">
      <UIcon name="i-tabler-search-off" class="size-5" />
      <span>{{ emptyLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { LauncherDisplay, ResolvedLauncherGroup, ResolvedLauncherItem } from './launcherModel'

export type LauncherCommandStatus = 'idle' | 'running' | 'success' | 'error'

const {
  groups,
  display = 'both',
  selectedId = '',
  statuses = {},
  preview = false,
  emptyLabel,
  runLabel,
  statusLabels,
  staleLabel,
} = defineProps<{
  groups: ResolvedLauncherGroup[]
  display?: LauncherDisplay
  selectedId?: string
  statuses?: Record<string, LauncherCommandStatus>
  preview?: boolean
  emptyLabel: string
  runLabel: (name: string) => string
  statusLabels: Pick<Record<LauncherCommandStatus, string>, 'running' | 'success' | 'error'>
  staleLabel: string
}>()

const emit = defineEmits<{
  run: [workflowId: string]
  select: [workflowId: string]
}>()

function statusFor(workflowId: string): LauncherCommandStatus {
  return statuses[workflowId] ?? 'idle'
}

function statusText(item: ResolvedLauncherItem) {
  if (item.stale) return staleLabel
  const status = statusFor(item.workflowId)
  return status === 'idle' ? '' : statusLabels[status]
}

function statusIcon(workflowId: string, fallback: string) {
  switch (statusFor(workflowId)) {
    case 'running':
      return 'i-tabler-loader-2'
    case 'success':
      return 'i-tabler-check'
    case 'error':
      return 'i-tabler-alert-circle'
    default:
      return fallback
  }
}
</script>

<style scoped>
.launcher-surface {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 8px;
}

.launcher-surface__group {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.launcher-surface__group + .launcher-surface__group {
  padding-top: 8px;
  border-top: 1px solid color-mix(in oklab, var(--ui-border) 72%, transparent);
}

.launcher-surface__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 2px 6px;
  color: var(--ui-text-dimmed);
  font-size: 10px;
  font-weight: 650;
  line-height: 16px;
  letter-spacing: 0.06em;
}

.launcher-surface__items {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.launcher-command {
  position: relative;
  display: grid;
  width: 100%;
  min-width: 0;
  min-height: 42px;
  grid-template-columns: 30px minmax(0, 1fr) auto;
  align-items: center;
  gap: 9px;
  justify-content: initial;
  padding: 5px 7px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--ui-text);
  text-align: left;
  transition:
    border-color 120ms ease,
    background-color 120ms ease,
    color 120ms ease;
}

.launcher-command:hover,
.launcher-command--selected {
  border-color: color-mix(in oklab, var(--ui-primary) 28%, var(--ui-border));
  background: color-mix(in oklab, var(--ui-primary) 7%, var(--ui-bg-elevated));
  color: var(--ui-text-highlighted);
}

.launcher-command:focus-visible {
  outline: 2px solid color-mix(in oklab, var(--ui-primary) 70%, transparent);
  outline-offset: -2px;
}

.launcher-command__icon {
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in oklab, var(--ui-border) 76%, transparent);
  border-radius: 7px;
  color: var(--ui-text-toned);
  background: color-mix(in oklab, var(--ui-bg-elevated) 72%, transparent);
}

.launcher-command__copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 1px;
}

.launcher-command__label {
  overflow: hidden;
  font-size: 12px;
  font-weight: 600;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.launcher-command__status {
  color: var(--ui-text-dimmed);
  font-size: 10px;
  line-height: 12px;
}

.launcher-command__meta {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
}

.launcher-command__meta :deep(kbd),
.launcher-command__ordinal {
  min-width: 18px;
  height: 18px;
  padding-inline: 4px;
  font-size: 9px;
  opacity: 0.72;
}

.launcher-command--running .launcher-command__icon {
  color: var(--ui-primary);
  border-color: color-mix(in oklab, var(--ui-primary) 36%, var(--ui-border));
}

.launcher-command--success .launcher-command__icon {
  color: var(--ui-success);
}

.launcher-command--error .launcher-command__icon,
.launcher-command--error .launcher-command__status {
  color: var(--ui-error);
}

.launcher-command--stale {
  cursor: not-allowed;
  color: var(--ui-text-muted);
  opacity: 0.72;
}

.launcher-command--stale .launcher-command__icon,
.launcher-command--stale .launcher-command__status {
  color: var(--ui-warning);
}

.launcher-surface:not(.launcher-surface--icon) .launcher-command--separator-before {
  margin-top: 5px;
  border-top-color: var(--ui-border);
}

.launcher-surface--text .launcher-command {
  grid-template-columns: minmax(0, 1fr) auto;
}

.launcher-surface--icon .launcher-surface__items {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(46px, 1fr));
  gap: 5px;
}

.launcher-surface--icon .launcher-command {
  display: flex;
  min-height: 46px;
  align-items: center;
  justify-content: center;
  padding: 6px;
}

.launcher-surface--icon .launcher-command--separator-before {
  border-left-color: var(--ui-border-accented);
}

.launcher-surface--icon .launcher-command__ordinal {
  position: absolute;
  top: 3px;
  right: 3px;
}

.launcher-surface__empty {
  display: flex;
  min-height: 92px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px;
  color: var(--ui-text-dimmed);
  font-size: 11px;
  text-align: center;
}

.launcher-surface--preview {
  pointer-events: none;
}
</style>
