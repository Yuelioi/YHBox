<template>
  <BaseModal
    :open="open"
    :title="copy('title')"
    :icon="resourceIcon"
    size="2xl"
    @update:open="emit('update:open', $event)"
  >
    <div v-if="loading" class="flex flex-col gap-3" aria-busy="true">
      <USkeleton class="h-16 w-full" />
      <USkeleton class="h-28 w-full" />
      <p class="text-xs text-muted" role="status">{{ copy('scanning') }}</p>
    </div>

    <UAlert
      v-else-if="error"
      color="error"
      variant="soft"
      icon="i-tabler-alert-circle"
      :title="copy('scan_failed')"
      :description="error"
    />

    <UEmpty
      v-else-if="preview.unused.length === 0"
      icon="i-tabler-sparkles"
      :title="copy('empty_title')"
      :description="copy('empty_desc', { n: preview.referenced.length })"
    />

    <div v-else class="flex flex-col gap-5">
      <div class="grid grid-cols-2 gap-3">
        <div class="rounded-lg border border-default px-4 py-3">
          <p class="text-xs text-muted">{{ copy('can_delete') }}</p>
          <p class="mt-1 text-xl font-semibold tabular-nums text-highlighted">
            {{ preview.unused.length }}
          </p>
        </div>
        <div class="rounded-lg border border-default px-4 py-3">
          <p class="text-xs text-muted">{{ copy('in_use') }}</p>
          <p class="mt-1 text-xl font-semibold tabular-nums text-highlighted">
            {{ preview.referenced.length }}
          </p>
        </div>
      </div>

      <section class="flex flex-col gap-2" :aria-labelledby="selectionTitleId">
        <div class="flex items-center justify-between gap-4">
          <h4 :id="selectionTitleId" class="text-sm font-medium text-highlighted">
            {{ copy('selected_title') }}
          </h4>
          <UButton size="xs" color="neutral" variant="ghost" @click="toggleAll">
            {{ allSelected ? copy('clear_selection') : copy('select_all') }}
          </UButton>
        </div>
        <ul
          class="max-h-64 overflow-y-auto rounded-lg border border-default divide-y divide-default/60"
        >
          <li
            v-for="item in preview.unused"
            :key="item.id"
            class="flex items-center gap-3 px-3 py-2.5"
          >
            <UCheckbox
              :model-value="selected.has(item.id)"
              :aria-label="copy('select_item', { name: item.label })"
              @update:model-value="toggle(item.id, !!$event)"
            />
            <UIcon
              :name="kindIcon(item.kind)"
              class="size-4 shrink-0 text-dimmed"
              aria-hidden="true"
            />
            <span class="min-w-0 flex-1 truncate text-sm text-default">{{ item.label }}</span>
            <UBadge color="neutral" variant="soft" size="xs" :label="kindLabel(item.kind)" />
          </li>
        </ul>
      </section>

      <UAlert
        v-if="preview.referenced.length > 0"
        color="warning"
        variant="soft"
        icon="i-tabler-link"
        :title="copy('skipped_title', { n: preview.referenced.length })"
        :description="copy('skipped_desc')"
      />
    </div>

    <template #footer>
      <UButton color="neutral" variant="ghost" :disabled="busy" @click="emit('update:open', false)">
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        color="error"
        icon="i-tabler-trash"
        :loading="busy"
        :disabled="selected.size === 0 || loading || !!error"
        @click="emit('confirm', [...selected])"
      >
        {{ copy('delete_count', { n: selected.size }) }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseModal from '@/components/common/BaseModal.vue'

type CleanupResource = 'recording' | 'subgraph' | 'template'

interface CleanupItem {
  id: string
  label: string
  kind: string
  references: number
}

interface CleanupPreview {
  unused: CleanupItem[]
  referenced: CleanupItem[]
}

const props = defineProps<{
  resource: CleanupResource
  open: boolean
  preview: CleanupPreview
  loading: boolean
  busy: boolean
  error: string
}>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  confirm: [ids: string[]]
}>()
const { t } = useI18n()
const selected = ref(new Set<string>())
const allSelected = computed(
  () => props.preview.unused.length > 0 && selected.value.size === props.preview.unused.length,
)
const resourceIcon = computed(() => {
  if (props.resource === 'subgraph') return 'i-tabler-hierarchy'
  if (props.resource === 'template') return 'i-tabler-photo-search'
  return 'i-tabler-database-search'
})
const selectionTitleId = computed(() => `${props.resource}-cleanup-selection-title`)

watch(
  () => props.preview.unused,
  (items) => {
    selected.value = new Set(items.map((item) => item.id))
  },
  { immediate: true },
)

function copy(key: string, params?: Record<string, unknown>) {
  return t(`${props.resource}Cleanup.${key}`, params ?? {})
}

function toggle(id: string, checked: boolean) {
  const next = new Set(selected.value)
  if (checked) next.add(id)
  else next.delete(id)
  selected.value = next
}

function toggleAll() {
  selected.value = allSelected.value
    ? new Set()
    : new Set(props.preview.unused.map((item) => item.id))
}

function kindIcon(kind: string) {
  if (props.resource === 'subgraph') return 'i-tabler-hierarchy'
  if (props.resource === 'template') return 'i-tabler-photo'
  return kind === 'precise' ? 'i-tabler-movie' : 'i-tabler-route'
}

function kindLabel(kind: string) {
  if (props.resource === 'subgraph') return t('subgraphCleanup.kind')
  if (props.resource === 'template') return t('templateCleanup.kind')
  return kind === 'precise' ? t('recordingSave.mode_precise') : t('recordingSave.mode_simple')
}
</script>
