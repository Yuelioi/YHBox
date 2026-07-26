<template>
  <section class="hud-state-panel" :data-tone="tone" :data-size="size">
    <div class="hud-state-panel__heading">
      <span v-if="active" class="hud-state-panel__pulse" aria-hidden="true" />
      <UIcon v-else-if="icon" :name="icon" class="size-4" aria-hidden="true" />
      <span class="hud-state-panel__eyebrow">{{ eyebrow }}</span>
      <slot name="meta" />
    </div>

    <div v-if="value || $slots.value" class="hud-state-panel__value font-mono">
      <slot name="value">{{ value }}</slot>
    </div>

    <p v-if="hint" class="hud-state-panel__hint">{{ hint }}</p>
    <div v-if="$slots.default" class="hud-state-panel__details">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
const {
  tone = 'primary',
  icon = '',
  eyebrow = '',
  value = '',
  hint = '',
  active = false,
  size = 'md',
} = defineProps<{
  tone?: 'primary' | 'error' | 'success' | 'warning' | 'neutral'
  icon?: string
  eyebrow?: string
  value?: string | number
  hint?: string
  active?: boolean
  size?: 'sm' | 'md' | 'lg'
}>()
</script>

<style scoped>
.hud-state-panel {
  --state-tone: var(--ui-primary);
  display: flex;
  width: 100%;
  min-width: 0;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid color-mix(in oklab, var(--state-tone) 34%, var(--ui-border));
  border-radius: 12px;
  padding: 14px 16px;
  color: var(--state-tone);
  background: color-mix(in oklab, var(--state-tone) 9%, var(--ui-bg));
  text-align: center;
}

.hud-state-panel[data-tone='error'] {
  --state-tone: var(--ui-error);
}

.hud-state-panel[data-tone='success'] {
  --state-tone: var(--ui-success);
}

.hud-state-panel[data-tone='warning'] {
  --state-tone: var(--ui-warning);
}

.hud-state-panel[data-tone='neutral'] {
  --state-tone: var(--ui-text-muted);
  border-style: dashed;
  background: color-mix(in oklab, var(--ui-bg-elevated) 38%, transparent);
}

.hud-state-panel[data-size='sm'] {
  gap: 4px;
  padding: 10px 12px;
}

.hud-state-panel[data-size='lg'] {
  gap: 8px;
  padding: 18px 20px;
}

.hud-state-panel__heading {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: center;
  gap: 7px;
}

.hud-state-panel__eyebrow {
  overflow: hidden;
  font-size: 11px;
  line-height: 14px;
  font-weight: 650;
  letter-spacing: 0.04em;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hud-state-panel__pulse {
  width: 8px;
  height: 8px;
  flex: none;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 0 4px color-mix(in oklab, currentColor 12%, transparent);
  animation: hud-state-pulse 1.8s ease-in-out infinite;
}

.hud-state-panel__value {
  font-size: 32px;
  line-height: 1;
  font-weight: 620;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.04em;
  color: var(--ui-text-highlighted);
}

[data-size='sm'] .hud-state-panel__value {
  font-size: 26px;
}

[data-size='lg'] .hud-state-panel__value {
  font-size: 42px;
}

.hud-state-panel__hint,
.hud-state-panel__details {
  max-width: 36ch;
  font-size: 10px;
  line-height: 14px;
  color: var(--ui-text-dimmed);
}

@keyframes hud-state-pulse {
  50% {
    opacity: 0.5;
    transform: scale(0.88);
  }
}

@media (prefers-reduced-motion: reduce) {
  .hud-state-panel__pulse {
    animation: none;
  }
}
</style>
