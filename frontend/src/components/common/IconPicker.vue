<template>
  <div class="w-full space-y-2">
    <UInput
      v-model="query"
      size="xs"
      class="w-full"
      :placeholder="t('iconPicker.search_placeholder')"
      icon="i-tabler-search"
      :loading="searching"
    />
    <div class="flex max-h-48 flex-wrap gap-1.5 overflow-auto">
      <button
        v-for="icon in shown"
        :key="icon"
        type="button"
        class="icon-chip"
        :class="{ 'is-selected': modelValue === icon }"
        :title="icon"
        @click="emit('update:modelValue', icon)"
      >
        <UIcon :name="icon" class="size-4" />
      </button>
      <div
        v-if="query.trim() && shown.length === 0 && !searching"
        class="px-1 py-2 text-[11px] text-dimmed"
      >
        {{ t('iconPicker.no_match') }}
      </div>
    </div>
    <p v-if="query.trim()" class="text-[10px] text-dimmed">{{ t('iconPicker.search_hint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

defineProps<{ modelValue: string | undefined | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { t } = useI18n()

const curated = [
  'i-tabler-note',
  'i-tabler-info-circle',
  'i-tabler-alert-triangle',
  'i-tabler-bulb',
  'i-tabler-star',
  'i-tabler-flag',
  'i-tabler-pin',
  'i-tabler-bookmark',
  'i-tabler-help-circle',
  'i-tabler-circle-check',
  'i-tabler-circle-x',
  'i-tabler-list-check',
  'i-tabler-book',
  'i-tabler-bolt',
  'i-tabler-settings',
  'i-tabler-target',
]
const query = ref('')
const allNames = ref<string[]>([])
const searching = ref(false)
let loaded = false

async function ensureLoaded() {
  if (loaded) return
  loaded = true
  searching.value = true
  try {
    const { default: names } = await import('virtual:tabler-icon-names')
    allNames.value = names.map((name) => `i-tabler-${name}`)
  } finally {
    searching.value = false
  }
}

watch(query, (value) => {
  if (value.trim()) void ensureLoaded()
})

const shown = computed(() => {
  const value = query.value.trim().toLowerCase()
  if (!value) return curated
  const candidates = allNames.value.length ? allNames.value : curated
  return candidates.filter((name) => name.includes(value)).slice(0, 120)
})
</script>

<style scoped>
.icon-chip {
  display: flex;
  width: 26px;
  height: 26px;
  cursor: pointer;
  align-items: center;
  justify-content: center;
  border: 1px solid rgb(255 255 255 / 8%);
  border-radius: 6px;
  background: rgb(255 255 255 / 5%);
  color: var(--ui-text-toned);
  transition: all 120ms ease;
}
.icon-chip:hover {
  background: rgb(255 255 255 / 12%);
  color: var(--ui-text-default);
}
.icon-chip.is-selected {
  border-color: var(--ui-primary);
  background: var(--ui-primary);
  color: white;
}
</style>
