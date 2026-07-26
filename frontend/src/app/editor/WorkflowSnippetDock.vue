<template>
  <section class="flex h-full min-h-0 flex-col bg-default" data-testid="workflow-snippet-dock">
    <header class="shrink-0 space-y-2 border-b border-default px-3 py-3">
      <div class="flex items-start gap-2">
        <div class="min-w-0 flex-1">
          <h2 class="text-xs font-semibold text-highlighted">{{ t('workflow.snippets.title') }}</h2>
          <p class="mt-1 text-[10px] leading-4 text-muted">{{ t('workflow.snippets.hint') }}</p>
        </div>
        <UButton
          icon="i-tabler-refresh"
          color="neutral"
          variant="ghost"
          size="xs"
          :loading="store.loading"
          :aria-label="t('common.refresh')"
          @click="reload"
        />
      </div>
      <UInput
        v-model="query"
        data-testid="workflow-snippet-search"
        icon="i-tabler-search"
        size="sm"
        :placeholder="t('workflow.snippets.search')"
      />
      <div class="grid grid-cols-2 gap-2">
        <AdaptiveSelect v-model="category" :items="categoryItems" width-mode="fill" size="sm" />
        <AdaptiveSelect v-model="tag" :items="tagItems" width-mode="fill" size="sm" />
      </div>
    </header>

    <div class="min-h-0 flex-1 overflow-y-auto p-2">
      <div
        v-if="store.warnings.length"
        class="mb-2 rounded-lg border border-warning/30 bg-warning/10 px-3 py-2 text-[10px] leading-4 text-warning"
        role="status"
      >
        <p>{{ t('workflow.snippets.corrupt', { count: store.warnings.length }) }}</p>
        <ul class="mt-1 space-y-0.5 font-mono text-[9px]">
          <li v-for="warning in store.warnings.slice(0, 3)" :key="warning.file" class="truncate">
            {{ warning.file }} · {{ warning.error }}
          </li>
        </ul>
        <p class="mt-1 text-muted">{{ t('workflow.snippets.corrupt_hint') }}</p>
      </div>
      <div v-if="store.loading && !store.loaded" class="space-y-2">
        <USkeleton v-for="index in 8" :key="index" class="h-16 rounded-lg" />
      </div>
      <div v-else-if="filtered.length" class="space-y-1.5">
        <article
          v-for="snippet in filtered"
          :key="snippet.id"
          draggable="true"
          data-testid="workflow-snippet-item"
          class="group flex cursor-grab items-center gap-2 rounded-lg border border-default bg-elevated/20 p-2 transition-colors hover:border-primary/40 hover:bg-elevated/50 active:cursor-grabbing"
          tabindex="0"
          @click="emit('use', snippet.id)"
          @keydown.enter.prevent="emit('use', snippet.id)"
          @dragstart="startDrag($event, snippet.id)"
        >
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-md bg-elevated text-primary"
          >
            <UIcon name="i-tabler-bookmark" class="size-4" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-xs font-medium text-highlighted">{{
              snippet.name
            }}</span>
            <span class="mt-0.5 block truncate font-mono text-[9px] text-dimmed">
              {{ nodeLabel(snippet.nodeTypeId) }}
            </span>
            <span v-if="snippet.category || snippet.tags.length" class="mt-1 flex min-w-0 gap-1">
              <UBadge v-if="snippet.category" color="neutral" variant="soft" size="xs">
                {{ snippet.category }}
              </UBadge>
              <span class="truncate text-[9px] text-dimmed">{{ snippet.tags.join(' · ') }}</span>
            </span>
            <UKbd v-if="snippet.shortcut" size="xs" class="mt-1">{{ snippet.shortcut }}</UKbd>
          </span>
          <UButton
            data-testid="workflow-snippet-edit"
            icon="i-tabler-pencil"
            color="neutral"
            variant="ghost"
            size="xs"
            class="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100"
            :aria-label="t('workflow.snippets.edit')"
            @click.stop="emit('edit', snippet.id)"
          />
          <UButton
            data-testid="workflow-snippet-delete"
            icon="i-tabler-trash"
            color="error"
            variant="ghost"
            size="xs"
            class="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100"
            :aria-label="t('workflow.snippets.delete')"
            @click.stop="emit('delete', snippet.id)"
          />
        </article>
      </div>
      <EmptyState
        v-else
        inset
        icon="i-tabler-bookmarks"
        :title="t('workflow.snippets.empty')"
        :description="t('workflow.snippets.empty_hint')"
      />
    </div>
    <footer
      class="flex h-10 shrink-0 items-center border-t border-default px-3 text-[10px] text-dimmed"
    >
      {{ t('workflow.snippets.count', { count: filtered.length }) }}
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { errorMessage } from '@/lib/invoke'
import { useSnippetsStore } from '@/stores/snippets'

const props = defineProps<{ dragFormat: string }>()
const emit = defineEmits<{ use: [id: string]; edit: [id: string]; delete: [id: string] }>()
const { t } = useI18n()
const toast = useToast()
const store = useSnippetsStore()
const all = '__all__'
const query = ref('')
const category = ref(all)
const tag = ref(all)

const categoryItems = computed(() => [
  { label: t('workflow.snippets.all_categories'), value: all },
  ...store.categories.map((value) => ({ label: value, value })),
])
const tagItems = computed(() => [
  { label: t('workflow.snippets.all_tags'), value: all },
  ...store.tags.map((value) => ({ label: value, value })),
])
const filtered = computed(() => {
  const search = query.value.trim().toLocaleLowerCase()
  return store.items.filter((item) => {
    if (category.value !== all && item.category !== category.value) return false
    if (tag.value !== all && !item.tags.includes(tag.value)) return false
    if (!search) return true
    return [item.name, item.description, item.category, item.tags.join(' '), item.nodeTypeId]
      .join(' ')
      .toLocaleLowerCase()
      .includes(search)
  })
})

onMounted(() => void reload())

async function reload(): Promise<void> {
  try {
    await store.load(true)
  } catch (error) {
    toast.add({
      title: t('workflow.snippets.load_failed'),
      description: errorMessage(error),
      color: 'error',
    })
  }
}

function startDrag(event: DragEvent, id: string): void {
  if (!event.dataTransfer) return
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData(props.dragFormat, id)
}

function nodeLabel(nodeTypeID: string): string {
  const parts = nodeTypeID.split('/')
  return parts.at(-1) || nodeTypeID
}
</script>
