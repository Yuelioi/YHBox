<template>
  <USelectMenu
    v-model="model"
    v-bind="$attrs"
    :items="items"
    :label-key="labelKey"
    :value-key="valueKey"
    :search-input="{ placeholder: t('common.search_options') }"
    :virtualize="virtualize"
  />
</template>

<script setup lang="ts">
import type { SelectItem, SelectValue } from '@nuxt/ui'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'SearchableSelect', inheritAttrs: false })

defineProps<{
  items: SelectItem[]
  labelKey: string
  valueKey: string
  virtualize: boolean
}>()

const model = defineModel<SelectValue>({ required: true })
const { t } = useI18n()
</script>
