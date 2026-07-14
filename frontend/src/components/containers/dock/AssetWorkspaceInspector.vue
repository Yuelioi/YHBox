<template>
  <aside
    ref="rootRef"
    class="asset-workspace-inspector min-h-0 flex-col bg-default"
    :data-open="open"
    :aria-label="title"
    @keydown.esc.stop="emit('close')"
    @keydown.tab="trapFocus"
  >
    <header class="compact-header shrink-0 items-center gap-2 border-b border-default px-3 py-2">
      <UButton
        ref="closeRef"
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-arrow-left"
        @click="emit('close')"
      >
        {{ t('common.back') }}
      </UButton>
      <span class="min-w-0 flex-1 truncate text-sm font-medium text-highlighted">{{ title }}</span>
    </header>
    <div class="min-h-0 flex-1 overflow-y-auto">
      <slot />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { open } = defineProps<{ title: string; open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const rootRef = useTemplateRef<HTMLElement>('rootRef')
const closeRef = useTemplateRef<{ $el?: HTMLElement }>('closeRef')
const compact = ref(false)
let resizeObserver: ResizeObserver | undefined
let previousFocus: HTMLElement | null = null

function setBackgroundInert(value: boolean) {
  const root = rootRef.value
  if (!root?.parentElement) return
  for (const sibling of root.parentElement.children) {
    if (sibling !== root && sibling instanceof HTMLElement) sibling.inert = value
  }
}

function syncCompactMode() {
  const panel = rootRef.value?.closest<HTMLElement>('.asset-panel')
  compact.value = (panel?.clientWidth ?? 1040) < 1040
}

watch(
  () => open && compact.value,
  async (active) => {
    setBackgroundInert(active)
    if (active) {
      previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
      await nextTick()
      closeRef.value?.$el?.querySelector<HTMLElement>('button')?.focus()
      closeRef.value?.$el?.focus()
    } else if (previousFocus) {
      previousFocus.focus()
      previousFocus = null
    }
  },
)

function trapFocus(event: KeyboardEvent) {
  if (!open || !compact.value || !rootRef.value) return
  const focusable = [
    ...rootRef.value.querySelectorAll<HTMLElement>(
      'button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  ].filter((element) => !element.hidden)
  if (!focusable.length) return
  const first = focusable[0]!
  const last = focusable.at(-1)!
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

onMounted(() => {
  const panel = rootRef.value?.closest<HTMLElement>('.asset-panel')
  syncCompactMode()
  if (panel) {
    resizeObserver = new ResizeObserver(syncCompactMode)
    resizeObserver.observe(panel)
  }
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  setBackgroundInert(false)
})
</script>

<style scoped>
.asset-workspace-inspector {
  display: flex;
  width: 360px;
  flex-shrink: 0;
  overflow: hidden;
  border-left: 1px solid var(--ui-border);
}

.compact-header {
  display: none;
}

@container (width < 1040px) {
  .asset-workspace-inspector {
    display: none;
  }

  .asset-workspace-inspector[data-open='true'] {
    position: absolute;
    inset: 0;
    z-index: 20;
    display: flex;
    width: auto;
    border-left: 0;
  }

  .compact-header {
    display: flex;
  }
}
</style>
