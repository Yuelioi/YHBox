<template>
  <BaseModal
    :open="true"
    :title="t('recordingSave.title')"
    icon="i-tabler-device-floppy"
    size="lg"
    :show-close="false"
    :dismissible="false"
  >
    <form class="flex flex-col gap-5" @submit.prevent="submit">
      <div class="flex items-center gap-3 rounded-lg border border-default bg-elevated/40 px-4 py-3">
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
          <UIcon :name="modeIcon" class="size-5" aria-hidden="true" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-highlighted">{{ modeLabel }}</p>
          <p class="text-xs text-muted">
            {{ t('recordingSave.summary', { duration: durationLabel, count: pending.eventCount }) }}
          </p>
        </div>
        <UBadge color="warning" variant="soft" :label="t('recordingSave.pending')" />
      </div>

      <UFormField
        :label="t('recordingSave.name')"
        :description="t('recordingSave.name_hint')"
        :error="nameError"
        required
      >
        <UInput
          v-model="label"
          autofocus
          maxlength="80"
          :placeholder="t('recordingSave.name_placeholder')"
          :aria-invalid="!!nameError"
          @blur="nameTouched = true"
        />
      </UFormField>

      <div class="grid gap-4 sm:grid-cols-2">
        <UFormField :label="t('common.category')" :hint="t('common.optional')">
          <UInputMenu
            v-model="category"
            :items="categoryItems"
            :create-item="'always'"
            :placeholder="t('library.explorer.category_placeholder')"
            @create="onCreateCategory"
          />
        </UFormField>
        <UFormField :label="t('common.tags')" :hint="t('common.optional')">
          <UInputMenu
            v-model="tags"
            multiple
            :items="tagItems"
            :create-item="'always'"
            :placeholder="t('library.explorer.filter_tags')"
            @create="onCreateTag"
          />
        </UFormField>
      </div>

      <UFormField :label="t('common.description')" :hint="t('common.optional')">
        <UTextarea v-model="description" :rows="2" :placeholder="t('recordingSave.description_placeholder')" />
      </UFormField>
    </form>

    <template #footer>
      <p class="mr-auto max-w-56 text-xs text-dimmed" role="status" aria-live="polite">
        {{ discardArmed ? t('recordingSave.discard_confirm_hint') : t('recordingSave.pending_hint') }}
      </p>
      <UButton
        color="error"
        variant="ghost"
        icon="i-tabler-trash"
        :disabled="busy"
        @click="armOrDiscard"
      >
        {{ discardArmed ? t('recordingSave.discard_confirm') : t('recordingSave.discard') }}
      </UButton>
      <UButton
        color="primary"
        icon="i-tabler-check"
        :loading="busy"
        :disabled="!canSave"
        @click="submit"
      >
        {{ replaceMode ? t('recordingSave.save_replace') : t('recordingSave.save_add') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RecordingStopPayload } from '@/stores/recording'
import { backend } from '@/lib/backend'
import BaseModal from '@/components/common/BaseModal.vue'

const props = defineProps<{
  pending: RecordingStopPayload
  busy: boolean
  replaceMode: boolean
}>()
const emit = defineEmits<{
  save: [metadata: { label: string; description: string; category: string; tags: string[] }]
  discard: []
}>()

const { t } = useI18n()
const label = ref('')
const description = ref('')
const category = ref('')
const tags = ref<string[]>([])
const categories = ref<string[]>([])
const knownTags = ref<string[]>([])
const nameTouched = ref(false)
const discardArmed = ref(false)
let discardTimer: ReturnType<typeof setTimeout> | null = null

const modeLabel = computed(() => props.pending.filterMode === 'precise'
  ? t('recordingSave.mode_precise')
  : t('recordingSave.mode_simple'))
const modeIcon = computed(() => props.pending.filterMode === 'precise' ? 'i-tabler-movie' : 'i-tabler-route')
const durationLabel = computed(() => {
  const total = Math.max(0, Math.round(props.pending.durationUs / 1_000_000))
  const minutes = Math.floor(total / 60)
  const seconds = String(total % 60).padStart(2, '0')
  return `${minutes}:${seconds}`
})
const trimmedLabel = computed(() => label.value.trim())
const nameError = computed(() => nameTouched.value && !trimmedLabel.value ? t('recordingSave.name_required') : '')
const canSave = computed(() => !!trimmedLabel.value && !props.busy)
const categoryItems = computed(() => [...new Set([...categories.value, category.value].filter(Boolean))])
const tagItems = computed(() => [...new Set([...knownTags.value, ...tags.value].filter(Boolean))])

watch(() => props.pending.pendingID, () => {
  label.value = ''
  description.value = ''
  category.value = ''
  tags.value = []
  nameTouched.value = false
  void loadOptions()
}, { immediate: true })

async function loadOptions() {
  try {
    const [subgraphs, clips] = await Promise.all([
      backend.subgraphs.list(),
      backend.clipsContainer.list(),
    ])
    const assets = [...(subgraphs ?? []), ...(clips ?? [])] as Array<{ category?: string; tags?: string[] }>
    categories.value = [...new Set(assets.map((item) => item.category?.trim()).filter(Boolean) as string[])]
    knownTags.value = [...new Set(assets.flatMap((item) => item.tags ?? []).map((tag) => tag.trim()).filter(Boolean))]
  } catch {
    categories.value = []
    knownTags.value = []
  }
}

function onCreateCategory(value: string) {
  const next = value.trim()
  if (!next) return
  categories.value = [...new Set([...categories.value, next])]
  category.value = next
}

function onCreateTag(value: string) {
  const next = value.trim()
  if (!next) return
  knownTags.value = [...new Set([...knownTags.value, next])]
  tags.value = [...new Set([...tags.value, next])]
}

function submit() {
  nameTouched.value = true
  if (!canSave.value) return
  emit('save', {
    label: trimmedLabel.value,
    description: description.value.trim(),
    category: category.value.trim(),
    tags: [...new Set(tags.value.map((tag) => tag.trim()).filter(Boolean))],
  })
}

function armOrDiscard() {
  if (props.busy) return
  if (discardArmed.value) {
    clearDiscardTimer()
    emit('discard')
    return
  }
  discardArmed.value = true
  discardTimer = setTimeout(() => { discardArmed.value = false }, 4000)
}

function clearDiscardTimer() {
  if (discardTimer) clearTimeout(discardTimer)
  discardTimer = null
}

onBeforeUnmount(clearDiscardTimer)
</script>
