<template>
  <div v-if="!subgraph" class="text-sm text-dimmed">{{ t('subgraphProps.no_selection') }}</div>
  <div v-else class="space-y-4">
    <header class="flex items-center gap-2 pb-3 border-b border-default">
      <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
      <h3 class="text-sm font-medium text-highlighted">{{ subgraph.label }}</h3>
      <UBadge size="xs" color="neutral" variant="soft">{{
        t('subgraphProps.outputs_count', { n: subgraph.outputPins?.length ?? 0 })
      }}</UBadge>
    </header>

    <section class="space-y-1.5">
      <label class="block text-xs font-medium text-muted">ID</label>
      <button
        type="button"
        class="w-full text-left text-[11px] font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate flex items-center gap-1.5"
        :class="copied ? 'text-success' : 'text-dimmed'"
        :title="t('subgraphProps.click_to_copy') + subgraph.id"
        @click="onCopyID"
      >
        <UIcon v-if="copied" name="i-tabler-check" class="size-3 shrink-0" />
        <span class="truncate">{{ copied ? t('common.copied') : subgraph.id }}</span>
      </button>
    </section>

    <UButton
      size="xs"
      variant="soft"
      color="neutral"
      icon="i-tabler-code"
      block
      @click="$emit('to-script')"
    >
      {{ t('subgraphProps.to_script') }}
    </UButton>

    <section class="space-y-2">
      <label class="text-xs text-toned">{{ t('subgraphProps.name') }}</label>
      <UInput
        :model-value="subgraph.label"
        size="sm"
        @update:model-value="(v: string) => $emit('update', { label: v })"
      />
    </section>

    <section class="space-y-2">
      <label class="text-xs text-toned">{{ t('subgraphProps.description') }}</label>
      <UTextarea
        :model-value="subgraph.description ?? ''"
        :rows="2"
        size="sm"
        @update:model-value="(v: string) => $emit('update', { description: v })"
      />
    </section>

    <section class="space-y-2">
      <label class="text-xs text-toned">{{ t('common.category') }}</label>
      <UInputMenu
        :model-value="subgraph.category ?? ''"
        :create-item="'always'"
        :items="allCategoriesList"
        size="sm"
        :placeholder="t('library.explorer.category_placeholder')"
        @update:model-value="(v: string) => $emit('update', { category: v ?? '' })"
        @create="(v: string) => $emit('update', { category: v })"
      />
    </section>

    <!-- 标签 tags -->
    <section class="space-y-2">
      <label class="text-xs text-toned">tags</label>
      <UInputMenu
        :model-value="subgraph.tags ?? []"
        multiple
        :create-item="'always'"
        :items="allTagsList"
        size="sm"
        @update:model-value="(v: string[]) => $emit('update', { tags: v })"
        @create="(v: string) => $emit('update', { tags: [...(subgraph?.tags ?? []), v] })"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'

const { t } = useI18n()
const toast = useToast()

interface SubgraphLike {
  id: string
  label: string
  description?: string
  outputPins?: { id: string; name: string }[]
  tags?: string[]
  category?: string
}

const props = defineProps<{
  subgraph: SubgraphLike | null
  allTags?: string[]
  allCategories?: string[]
}>()
defineEmits<{ update: [patch: Record<string, any>]; 'to-script': [] }>()

const allTagsList = computed(() => props.allTags ?? [])
const allCategoriesList = computed(() => props.allCategories ?? [])

const copied = ref(false)
let copiedTimer = 0
async function onCopyID() {
  if (!props.subgraph) return
  try {
    await navigator.clipboard.writeText(props.subgraph.id)
    copied.value = true
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
      copied.value = false
    }, 1500)
  } catch {
    toast.add({ title: t('toast.copy_failed'), color: 'error' })
  }
}
</script>
