<template>
  <div class="space-y-2">
    <div
      v-for="(outputField, index) in fields"
      :key="index"
      class="space-y-2 rounded-lg border border-default p-2.5"
    >
      <div class="flex items-center gap-2">
        <span class="text-[11px] font-medium text-toned">
          {{ t('node.ai.extract.config.fields.field', { n: index + 1 }) }}
        </span>
        <UButton
          :aria-label="t('node.ai.extract.config.fields.remove', { n: index + 1 })"
          icon="i-tabler-trash"
          color="error"
          variant="ghost"
          size="xs"
          class="ml-auto"
          @click="removeField(index)"
        />
      </div>

      <div class="grid grid-cols-[minmax(0,1fr)_112px] gap-2">
        <UFormField :label="t('node.ai.extract.config.fields.name')" :ui="compactFieldUI">
          <UInput
            :model-value="outputField.name"
            :maxlength="64"
            class="w-full"
            @update:model-value="updateField(index, 'name', String($event))"
          />
        </UFormField>
        <UFormField :label="t('node.ai.extract.config.fields.type')" :ui="compactFieldUI">
          <USelect
            :model-value="outputField.type"
            :items="typeItems"
            value-key="value"
            label-key="label"
            class="w-full"
            @update:model-value="updateType(index, $event)"
          />
        </UFormField>
      </div>

      <UFormField
        :label="t('node.ai.extract.config.fields.field_description')"
        :ui="compactFieldUI"
      >
        <UInput
          :model-value="outputField.description"
          :maxlength="256"
          class="w-full"
          @update:model-value="updateField(index, 'description', String($event))"
        />
      </UFormField>

      <label class="flex min-h-8 items-center gap-2 text-[11px] text-muted">
        <USwitch
          :model-value="outputField.nullable"
          size="xs"
          @update:model-value="updateField(index, 'nullable', Boolean($event))"
        />
        {{ t('node.ai.extract.config.fields.nullable') }}
      </label>
    </div>

    <p v-if="fields.length === 0" class="text-[11px] leading-4 text-warning">
      {{ t('node.ai.extract.config.fields.empty') }}
    </p>
    <UButton
      :label="t('node.ai.extract.config.fields.add')"
      icon="i-tabler-plus"
      color="neutral"
      variant="soft"
      size="sm"
      block
      @click="addField"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

type OutputFieldType = 'string' | 'number' | 'integer' | 'boolean'

interface OutputField {
  name: string
  type: OutputFieldType
  description: string
  nullable: boolean
}

const props = defineProps<{ modelValue: unknown }>()
const emit = defineEmits<{ 'update:modelValue': [value: unknown] }>()
const { t } = useI18n()

const compactFieldUI = {
  label: 'text-[11px] font-medium text-muted',
  container: 'mt-1',
}
const typeItems = computed(() =>
  (['string', 'number', 'integer', 'boolean'] as const).map((value) => ({
    value,
    label: t(`node.ai.extract.config.fields.types.${value}`),
  })),
)
const fields = computed(() => normalizeFields(props.modelValue))

function addField(): void {
  const names = new Set(fields.value.map((field) => field.name))
  let suffix = fields.value.length + 1
  let name = `field${suffix}`
  while (names.has(name)) {
    suffix += 1
    name = `field${suffix}`
  }
  emit('update:modelValue', [
    ...fields.value,
    { name, type: 'string', description: '', nullable: false },
  ])
}

function removeField(index: number): void {
  emit(
    'update:modelValue',
    fields.value.filter((_, current) => current !== index),
  )
}

function updateType(index: number, value: unknown): void {
  if (isOutputFieldType(value)) updateField(index, 'type', value)
}

function updateField<Key extends keyof OutputField>(
  index: number,
  key: Key,
  value: OutputField[Key],
): void {
  emit(
    'update:modelValue',
    fields.value.map((field, current) => (current === index ? { ...field, [key]: value } : field)),
  )
}

function normalizeFields(value: unknown): OutputField[] {
  if (!Array.isArray(value)) return []
  return value.flatMap((candidate) => {
    if (!candidate || typeof candidate !== 'object') return []
    const record = candidate as Record<string, unknown>
    const type = isOutputFieldType(record.type) ? record.type : 'string'
    return [
      {
        name: typeof record.name === 'string' ? record.name : '',
        type,
        description: typeof record.description === 'string' ? record.description : '',
        nullable: record.nullable === true,
      },
    ]
  })
}

function isOutputFieldType(value: unknown): value is OutputFieldType {
  return ['string', 'number', 'integer', 'boolean'].includes(String(value))
}
</script>
