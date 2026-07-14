<template>
  <div
    class="hud-shell"
    :class="{ 'hud-shell--dense': dense }"
    :data-accent="accent"
    :aria-label="title || undefined"
  >
    <header class="hud-shell__header" style="--wails-draggable: drag">
      <span v-if="icon" class="hud-shell__mark" aria-hidden="true">
        <UIcon :name="icon" class="size-4" />
      </span>

      <div v-if="title || subtitle" class="hud-shell__identity">
        <div class="flex min-w-0 items-center gap-2">
          <h1 v-if="title" class="hud-shell__title">{{ title }}</h1>
          <span v-if="status" class="hud-shell__status">
            <span v-if="statusActive" class="hud-shell__status-dot" />
            {{ status }}
          </span>
        </div>
        <p v-if="subtitle" class="hud-shell__subtitle">{{ subtitle }}</p>
      </div>

      <UIcon
        v-if="!icon && !title"
        name="i-tabler-grip-horizontal"
        class="size-3.5 shrink-0 text-dimmed"
      />

      <div class="hud-shell__actions" style="--wails-draggable: no-drag">
        <slot name="actions" />
        <UButton
          v-if="!noClose"
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-x"
          class="size-7 p-0"
          :title="resolvedCloseTitle"
          :aria-label="resolvedCloseTitle"
          @click="emit('close')"
        />
      </div>
    </header>

    <main class="hud-shell__body">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const {
  title = '',
  subtitle = '',
  icon = '',
  accent = 'primary',
  status = '',
  statusActive = false,
  dense = false,
  noClose = false,
  closeTitle = '',
} = defineProps<{
  title?: string
  subtitle?: string
  icon?: string
  accent?: 'primary' | 'error' | 'success' | 'warning' | 'neutral'
  status?: string
  statusActive?: boolean
  dense?: boolean
  noClose?: boolean
  closeTitle?: string
}>()
const emit = defineEmits<{ close: [] }>()
const resolvedCloseTitle = computed(() => closeTitle || t('hudShell.close'))
</script>

<style scoped>
.hud-shell {
  --hud-accent: var(--ui-primary);
  container-type: inline-size;
  display: flex;
  width: 100vw;
  height: 100vh;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  color: var(--ui-text);
  background:
    linear-gradient(
      180deg,
      color-mix(in oklab, var(--ui-bg-elevated) 30%, transparent),
      transparent 88px
    ),
    var(--ui-bg);
  box-shadow: inset 0 0 0 1px var(--ui-border);
  user-select: none;
}

.hud-shell[data-accent='error'] {
  --hud-accent: var(--ui-error);
}

.hud-shell[data-accent='success'] {
  --hud-accent: var(--ui-success);
}

.hud-shell[data-accent='warning'] {
  --hud-accent: var(--ui-warning);
}

.hud-shell[data-accent='neutral'] {
  --hud-accent: var(--ui-text-muted);
}

.hud-shell__header {
  display: flex;
  min-height: 42px;
  flex: none;
  align-items: center;
  gap: 10px;
  padding: 7px 8px 7px 10px;
  border-bottom: 1px solid var(--ui-border);
}

.hud-shell--dense .hud-shell__header {
  min-height: 34px;
  gap: 8px;
  padding: 4px 6px 4px 8px;
}

.hud-shell__mark {
  display: inline-flex;
  width: 28px;
  height: 28px;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in oklab, var(--hud-accent) 30%, var(--ui-border));
  border-radius: 8px;
  color: var(--hud-accent);
  background: color-mix(in oklab, var(--hud-accent) 10%, transparent);
}

.hud-shell--dense .hud-shell__mark {
  width: 24px;
  height: 24px;
  border-radius: 7px;
}

.hud-shell__identity {
  min-width: 0;
  flex: 1;
}

.hud-shell__title {
  overflow: hidden;
  font-size: 12px;
  line-height: 16px;
  font-weight: 650;
  color: var(--ui-text-highlighted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hud-shell__subtitle {
  overflow: hidden;
  margin-top: 1px;
  font-size: 10px;
  line-height: 13px;
  color: var(--ui-text-dimmed);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hud-shell__status {
  display: inline-flex;
  min-width: 0;
  flex: none;
  align-items: center;
  gap: 5px;
  border-radius: 999px;
  padding: 2px 7px;
  font-size: 10px;
  line-height: 13px;
  color: var(--hud-accent);
  background: color-mix(in oklab, var(--hud-accent) 10%, transparent);
}

.hud-shell__status-dot {
  width: 5px;
  height: 5px;
  flex: none;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 0 3px color-mix(in oklab, currentColor 12%, transparent);
}

.hud-shell__actions {
  display: flex;
  flex: none;
  align-items: center;
  gap: 2px;
}

.hud-shell__body {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
  flex-direction: column;
}

@container (max-width: 340px) {
  .hud-shell__subtitle {
    display: none;
  }
}
</style>
