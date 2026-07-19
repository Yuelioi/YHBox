<template>
  <BaseModal
    :open="open"
    :title="t('workflow.quick_add.title')"
    icon="i-tabler-apps"
    size="3xl"
    @update:open="emit('update:open', $event)"
  >
    <div
      data-testid="workflow-quick-add"
      class="grid h-[30rem] min-h-0 grid-cols-[11rem_minmax(0,1fr)] overflow-hidden rounded-lg border border-default"
      @keydown.down.prevent="move(1)"
      @keydown.up.prevent="move(-1)"
      @keydown.enter.prevent="selectActive"
      @keydown.esc.stop.prevent="emit('update:open', false)"
    >
      <aside class="overflow-y-auto border-r border-default bg-elevated/20 p-2">
        <p class="px-2 pb-2 text-[10px] font-semibold uppercase tracking-wider text-dimmed">
          {{ t('workflow.quick_add.categories') }}
        </p>
        <button
          v-for="entry in categories"
          :key="entry.value"
          type="button"
          class="flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-xs transition-colors hover:bg-elevated focus-visible:outline-2 focus-visible:outline-primary"
          :class="category === entry.value && !query ? 'bg-primary/10 text-primary' : 'text-toned'"
          @click="selectCategory(entry.value)"
        >
          <span class="min-w-0 flex-1 truncate">{{ entry.label }}</span>
          <span class="font-mono text-[9px] text-dimmed">{{ entry.count }}</span>
        </button>
      </aside>
      <section class="flex min-h-0 min-w-0 flex-col bg-default">
        <div class="border-b border-default p-3">
          <UInput
            ref="searchInput"
            v-model="query"
            data-testid="workflow-quick-add-search"
            icon="i-tabler-search"
            autofocus
            :placeholder="t('workflow.quick_add.search')"
          />
        </div>
        <div ref="resultsElement" class="min-h-0 flex-1 overflow-y-auto p-2">
          <button
            v-for="(item, index) in visibleItems"
            :key="`${item.kind}:${item.id}`"
            type="button"
            data-testid="workflow-quick-add-item"
            class="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors focus-visible:outline-2 focus-visible:outline-primary"
            :class="index === activeIndex ? 'bg-primary/10' : 'hover:bg-elevated'"
            @mouseenter="activeIndex = index"
            @click="choose(item)"
          >
            <span
              class="flex size-8 shrink-0 items-center justify-center rounded-md bg-elevated text-primary"
            >
              <UIcon :name="item.icon" class="size-4" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="block truncate text-xs font-medium text-highlighted">{{
                item.title
              }}</span>
              <span class="mt-0.5 block truncate text-[10px] text-muted">{{
                item.description
              }}</span>
            </span>
            <UKbd v-if="item.shortcut" size="xs">{{ item.shortcut }}</UKbd>
            <UBadge color="neutral" variant="soft" size="xs">{{ item.categoryLabel }}</UBadge>
          </button>
          <div v-if="!visibleItems.length" class="px-4 py-16 text-center">
            <UIcon name="i-tabler-search-off" class="mx-auto size-5 text-dimmed" />
            <p class="mt-2 text-xs text-muted">{{ t('workflow.quick_add.empty') }}</p>
          </div>
        </div>
      </section>
    </div>
    <template #footer>
      <span class="mr-auto text-[10px] text-muted">{{ t('workflow.quick_add.hint') }}</span>
      <span class="text-[10px] text-dimmed">{{
        t('workflow.quick_add.count', { n: visibleItems.length })
      }}</span>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseModal from '@/components/common/BaseModal.vue'
import {
  filterWorkflowQuickAddItems,
  moveWorkflowQuickAddIndex,
  type WorkflowQuickAddItem,
} from '@/app/editor/workflowQuickAdd'

const props = defineProps<{ open: boolean; items: WorkflowQuickAddItem[] }>()
const emit = defineEmits<{
  'update:open': [open: boolean]
  choose: [item: WorkflowQuickAddItem]
}>()
const { t } = useI18n()
const query = ref('')
const category = ref('all')
const activeIndex = ref(0)
const searchInput = ref<{ $el?: HTMLElement } | null>(null)
const resultsElement = ref<HTMLElement | null>(null)

const categories = computed(() => {
  const counts = new Map<string, { label: string; count: number }>()
  for (const item of props.items) {
    const current = counts.get(item.category) ?? { label: item.categoryLabel, count: 0 }
    current.count += 1
    counts.set(item.category, current)
  }
  return [
    { value: 'all', label: t('workflow.quick_add.all'), count: props.items.length },
    ...[...counts.entries()]
      .map(([value, item]) => ({ value, ...item }))
      .sort((left, right) => left.label.localeCompare(right.label)),
  ]
})
const visibleItems = computed(() =>
  filterWorkflowQuickAddItems(props.items, query.value, category.value),
)

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    query.value = ''
    category.value = 'all'
    activeIndex.value = 0
    await nextTick()
    searchInput.value?.$el?.querySelector('input')?.focus()
  },
)
watch([query, category], () => (activeIndex.value = 0))

function selectCategory(value: string): void {
  query.value = ''
  category.value = value
  activeIndex.value = 0
  searchInput.value?.$el?.querySelector('input')?.focus()
}

function move(delta: number): void {
  activeIndex.value = moveWorkflowQuickAddIndex(activeIndex.value, delta, visibleItems.value.length)
  nextTick(() => {
    resultsElement.value
      ?.querySelectorAll<HTMLElement>('[data-testid="workflow-quick-add-item"]')
      [activeIndex.value]?.scrollIntoView({ block: 'nearest' })
  })
}

function selectActive(): void {
  const item = visibleItems.value[activeIndex.value]
  if (item) choose(item)
}

function choose(item: WorkflowQuickAddItem): void {
  emit('choose', item)
  emit('update:open', false)
}
</script>
