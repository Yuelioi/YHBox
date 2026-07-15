<template>
  <section data-authoring-panel class="mb-5 space-y-4">
    <div class="rounded-md border border-default/60 bg-elevated/20 p-3">
      <div class="flex flex-wrap gap-1.5">
        <span
          class="rounded border border-default/70 bg-muted/70 px-1.5 py-0.5 text-[10px] text-toned"
          >{{ t(`inspector.authoring.execution.${projection.execution.class}`) }}</span
        >
        <span
          class="rounded border border-default/70 bg-muted/70 px-1.5 py-0.5 text-[10px] text-toned"
          >{{ t(`inspector.authoring.availability.${projection.availability}`) }}</span
        >
        <span
          v-if="projection.editorAdapter"
          class="rounded border border-default/70 bg-muted/70 px-1.5 py-0.5 font-mono text-[10px] text-toned"
        >
          {{ projection.editorAdapter }}
        </span>
      </div>

      <dl v-if="ports.length" class="mt-3 space-y-2">
        <div
          v-for="port in ports"
          :key="`${port.direction}:${port.id}`"
          class="grid grid-cols-[1fr_auto] gap-x-3 text-xs"
        >
          <dt class="min-w-0 text-toned">
            <span class="font-mono text-highlighted">{{ port.id }}</span>
            <span class="ml-1.5">{{ translate(port.type.titleKey, port.type.label) }}</span>
          </dt>
          <dd class="text-right text-dimmed">
            {{ t(`inspector.authoring.direction.${port.direction}`) }} ·
            {{ t(`inspector.authoring.binding.${port.binding}`) }}
          </dd>
          <dd class="col-span-2 mt-0.5 text-[11px] text-dimmed">
            {{ t(`inspector.authoring.carrier.${port.carrier}`) }} ·
            {{ t(`inspector.authoring.lifecycle.${port.type.lifecycle}`) }}
            <template v-if="port.hasDefault">
              ·
              {{
                t('inspector.authoring.port_default_hint', { value: displayValue(port.default) })
              }}
            </template>
          </dd>
          <dd
            v-if="port.type.descriptionKey && te(port.type.descriptionKey)"
            class="col-span-2 mt-0.5 text-[11px] leading-relaxed text-dimmed"
          >
            {{ t(port.type.descriptionKey) }}
          </dd>
        </div>
      </dl>
    </div>

    <div v-if="projection.configFields.length" class="space-y-3">
      <h4 class="text-xs font-semibold text-toned">{{ t('inspector.authoring.config') }}</h4>
      <div v-for="field in projection.configFields" :key="field.id" class="space-y-1.5">
        <label
          :for="controlID(field.id)"
          class="flex items-center gap-1.5 text-xs font-medium text-toned"
        >
          <span>{{ translate(field.titleKey, field.title || field.id) }}</span>
          <span :class="field.required ? 'text-error' : 'text-dimmed'">
            {{
              field.required ? t('inspector.authoring.required') : t('inspector.authoring.optional')
            }}
          </span>
          <span v-if="field.deprecated" class="text-warning">{{
            t('inspector.authoring.deprecated')
          }}</span>
        </label>

        <select
          v-if="field.control === 'select'"
          :id="controlID(field.id)"
          :value="serializedValue(field)"
          :disabled="field.readOnly"
          :required="field.required"
          :aria-describedby="descriptionID(field.id)"
          class="w-full rounded-md border border-default bg-default px-2.5 py-1.5 text-xs text-highlighted outline-none transition-colors focus:border-primary disabled:cursor-not-allowed disabled:opacity-50"
          @change="onSelect(field, $event)"
        >
          <option value="">{{ t('inspector.authoring.unset') }}</option>
          <option
            v-for="option in field.constraints.enum"
            :key="JSON.stringify(option)"
            :value="JSON.stringify(option)"
          >
            {{ String(option) }}
          </option>
        </select>
        <input
          v-else-if="field.control === 'toggle'"
          :id="controlID(field.id)"
          type="checkbox"
          :checked="valueFor(field.id) === true"
          :disabled="field.readOnly"
          :required="field.required"
          :aria-describedby="descriptionID(field.id)"
          class="size-4 accent-primary"
          @change="onToggle(field, $event)"
        />
        <input
          v-else-if="field.control === 'number' || field.control === 'integer'"
          :id="controlID(field.id)"
          type="number"
          :step="field.control === 'integer' ? 1 : 'any'"
          :min="numericConstraint(field.constraints.minimum)"
          :max="numericConstraint(field.constraints.maximum)"
          :value="valueFor(field.id) ?? ''"
          :disabled="field.readOnly"
          :required="field.required"
          :aria-describedby="descriptionID(field.id)"
          class="w-full rounded-md border border-default bg-default px-2.5 py-1.5 text-xs text-highlighted outline-none transition-colors focus:border-primary disabled:cursor-not-allowed disabled:opacity-50"
          @input="onNumber(field, $event)"
        />
        <textarea
          v-else-if="['object', 'list', 'json'].includes(field.control)"
          :id="controlID(field.id)"
          :value="jsonValue(field.id)"
          :disabled="field.readOnly"
          :required="field.required"
          :aria-describedby="descriptionID(field.id)"
          rows="4"
          spellcheck="false"
          class="w-full resize-y rounded-md border border-default bg-default px-2.5 py-1.5 font-mono text-xs text-highlighted outline-none transition-colors focus:border-primary disabled:cursor-not-allowed disabled:opacity-50"
          @change="onJSON(field, $event)"
        />
        <input
          v-else
          :id="controlID(field.id)"
          type="text"
          :value="valueFor(field.id) ?? ''"
          :readonly="field.readOnly"
          :required="field.required"
          :minlength="field.constraints.minLength"
          :maxlength="field.constraints.maxLength"
          :pattern="field.constraints.pattern"
          :aria-describedby="descriptionID(field.id)"
          class="w-full rounded-md border border-default bg-default px-2.5 py-1.5 text-xs text-highlighted outline-none transition-colors focus:border-primary disabled:cursor-not-allowed disabled:opacity-50"
          @input="onText(field, $event)"
        />

        <div
          :id="descriptionID(field.id)"
          class="space-y-0.5 text-[11px] leading-relaxed text-dimmed"
        >
          <p v-if="field.descriptionKey && te(field.descriptionKey)">
            {{ t(field.descriptionKey) }}
          </p>
          <p v-else-if="field.description">{{ field.description }}</p>
          <p v-if="field.hasDefault">
            {{ t('inspector.authoring.default_hint', { value: displayValue(field.default) }) }}
          </p>
          <p v-if="constraintTokens(field).length" class="font-mono break-all">
            {{ constraintTokens(field).join(' · ') }}
          </p>
          <p v-if="['object', 'list', 'json'].includes(field.control)">
            {{ t('inspector.authoring.json_editor') }}
          </p>
          <p v-if="jsonErrors[field.id]" role="alert" class="text-error">
            {{ jsonErrors[field.id] }}
          </p>
        </div>
      </div>
    </div>

    <div v-if="projection.capabilities.length" class="space-y-2">
      <h4 class="text-xs font-semibold text-toned">{{ t('inspector.authoring.capabilities') }}</h4>
      <div
        v-for="requirement in projection.capabilities"
        :key="requirement.requirementId"
        class="rounded-md border border-warning/25 bg-warning/5 px-3 py-2 text-[11px] leading-relaxed"
      >
        <div class="flex items-center gap-1.5 text-warning">
          <span class="font-mono">{{ requirement.requirementId }}</span>
          <span>· {{ t(`inspector.authoring.risk.${requirement.risk}`) }}</span>
          <span>· {{ t(`inspector.authoring.consent.${requirement.consent}`) }}</span>
        </div>
        <p class="mt-0.5 text-dimmed">
          {{ t('inspector.authoring.target', { slot: requirement.targetSlot }) }} ·
          {{ requirement.operations.join(', ') }}
        </p>
        <p class="text-dimmed">
          {{ t('inspector.authoring.target_kinds', { kinds: requirement.targetKinds.join(', ') }) }}
          ·
          {{ t('inspector.authoring.scope', { scope: displayValue(requirement.scope) }) }}
        </p>
        <p v-if="requirement.credential === 'required'" class="text-dimmed">
          {{ t('inspector.authoring.credential_required', { slot: requirement.credentialSlot }) }}
        </p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FieldProjection31 as FieldProjection, NodeProjection31 } from '@/contracts/node31'
import { patchProjectedConfig, projectedConstraintTokens } from '@/contracts/nodeAuthoringUi'

const props = defineProps<{
  nodeId: string
  projection: NodeProjection31
  modelValue?: Record<string, unknown>
}>()

const emit = defineEmits<{ 'update:modelValue': [config: Record<string, unknown>] }>()
const { t, te } = useI18n()
const jsonErrors = reactive<Record<string, string>>({})

const ports = computed(() => [
  ...props.projection.dataInputs.map((port) => ({ ...port, direction: 'input' as const })),
  ...props.projection.dataOutputs.map((port) => ({ ...port, direction: 'output' as const })),
])

function translate(key: string | undefined, fallback: string): string {
  return key && te(key) ? t(key) : fallback
}

function stableID(fieldID: string): string {
  return `${props.nodeId}-${fieldID}`.replace(/[^a-zA-Z0-9_-]/g, '-')
}

function controlID(fieldID: string): string {
  return `authoring-control-${stableID(fieldID)}`
}

function descriptionID(fieldID: string): string {
  return `authoring-description-${stableID(fieldID)}`
}

function valueFor(fieldID: string): unknown {
  return props.modelValue?.[fieldID]
}

function update(field: FieldProjection, value: unknown): void {
  emit('update:modelValue', patchProjectedConfig(props.modelValue, field.id, value))
}

function onText(field: FieldProjection, event: Event): void {
  update(field, (event.target as HTMLInputElement).value)
}

function onNumber(field: FieldProjection, event: Event): void {
  const raw = (event.target as HTMLInputElement).value
  update(field, raw === '' ? undefined : Number(raw))
}

function onToggle(field: FieldProjection, event: Event): void {
  update(field, (event.target as HTMLInputElement).checked)
}

function serializedValue(field: FieldProjection): string {
  const value = valueFor(field.id)
  return value === undefined ? '' : JSON.stringify(value)
}

function onSelect(field: FieldProjection, event: Event): void {
  const raw = (event.target as HTMLSelectElement).value
  update(field, raw === '' ? undefined : JSON.parse(raw))
}

function jsonValue(fieldID: string): string {
  const value = valueFor(fieldID)
  return value === undefined ? '' : JSON.stringify(value, null, 2)
}

function onJSON(field: FieldProjection, event: Event): void {
  const raw = (event.target as HTMLTextAreaElement).value.trim()
  if (raw === '') {
    delete jsonErrors[field.id]
    update(field, undefined)
    return
  }
  try {
    const parsed = JSON.parse(raw)
    delete jsonErrors[field.id]
    update(field, parsed)
  } catch {
    jsonErrors[field.id] = t('inspector.authoring.invalid_json')
  }
}

function constraintTokens(field: FieldProjection): string[] {
  return projectedConstraintTokens(field)
}

function displayValue(value: unknown): string {
  return typeof value === 'string' ? value : JSON.stringify(value)
}

function numericConstraint(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}
</script>
