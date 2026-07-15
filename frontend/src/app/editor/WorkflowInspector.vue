<template>
  <aside class="flex h-full w-[340px] shrink-0 flex-col border-l border-default bg-default">
    <div class="flex items-center justify-between border-b border-default px-4 py-3">
      <div class="min-w-0">
        <h2 class="truncate text-sm font-semibold text-highlighted">
          {{ t('workflow31.inspector.title') }}
        </h2>
        <p class="truncate font-mono text-[10px] text-dimmed">
          {{ node?.id || t('workflow31.inspector.no_selection') }}
        </p>
      </div>
      <UButton
        v-if="node"
        icon="i-tabler-trash"
        color="error"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow31.inspector.remove_node')"
        @click="emit('command', { kind: 'remove-node', nodeId: node.id })"
      />
    </div>

    <div
      v-if="!node || !projection"
      class="flex flex-1 items-center justify-center px-8 text-center"
    >
      <div>
        <UIcon name="i-tabler-pointer" class="mx-auto mb-3 size-6 text-dimmed" />
        <p class="text-xs text-muted">{{ t('workflow31.inspector.select_hint') }}</p>
      </div>
    </div>

    <div v-else class="flex-1 space-y-6 overflow-y-auto p-4">
      <section class="space-y-3">
        <label class="block text-xs font-medium text-toned" for="workflow-node-label">
          {{ t('workflow31.inspector.label') }}
        </label>
        <UInput
          id="workflow-node-label"
          :model-value="node.label || ''"
          :placeholder="t('workflow31.inspector.label_placeholder')"
          class="w-full"
          @change="setLabel"
        />
      </section>

      <section v-if="projection.configFields.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow31.inspector.configuration') }}
        </h3>
        <GeneratedFieldEditor
          v-for="field in projection.configFields"
          :key="field.id"
          :field="field"
          :model-value="node.config[field.id]"
          @update:model-value="
            emit('command', {
              kind: 'set-config',
              nodeId: node.id,
              fieldId: field.id,
              value: $event,
            })
          "
        />
      </section>

      <section v-if="projection.dataInputs.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow31.inspector.inputs') }}
        </h3>
        <div
          v-for="port in projection.dataInputs"
          :key="port.id"
          class="space-y-2 rounded-lg bg-elevated/55 p-3"
        >
          <div class="flex items-center gap-2">
            <span
              class="size-2 rounded-full"
              :style="{ backgroundColor: port.type.color || '#a1a1aa' }"
              aria-hidden="true"
            />
            <span class="text-xs font-medium text-toned">{{ port.id }}</span>
            <span class="ml-auto font-mono text-[10px] text-dimmed">{{ port.binding }}</span>
          </div>
          <UInput
            v-if="acceptsInline(port)"
            :model-value="literalText(node.bindings[port.id])"
            :placeholder="literalPlaceholder(port)"
            class="w-full"
            @change="setLiteral(port.id, $event)"
          />
          <p v-else class="text-[11px] leading-5 text-muted">
            {{ t('workflow31.inspector.reference_only', { carrier: port.carrier }) }}
          </p>
          <div class="flex items-center gap-2">
            <UButton
              v-if="port.hasDefault"
              :label="t('workflow31.inspector.use_default')"
              size="xs"
              color="neutral"
              variant="soft"
              @click="emit('command', { kind: 'bind-default', nodeId: node.id, portId: port.id })"
            />
            <UButton
              v-if="node.bindings[port.id]"
              :label="t('workflow31.inspector.clear')"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="emit('command', { kind: 'clear-binding', nodeId: node.id, portId: port.id })"
            />
          </div>
        </div>
      </section>

      <section v-if="projection.capabilities.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow31.inspector.capabilities') }}
        </h3>
        <div
          v-for="capability in projection.capabilities"
          :key="capability.requirementId"
          class="rounded-lg border border-default px-3 py-2.5"
        >
          <p class="text-xs font-medium text-toned">{{ capability.requirementId }}</p>
          <p class="mt-1 text-[11px] text-muted">
            {{ capability.operations.join(', ') }}
          </p>
          <p class="mt-1 font-mono text-[10px] text-dimmed">
            {{ capability.risk }} / {{ capability.consent }}
          </p>
        </div>
      </section>

      <section v-if="projection.statusEvents.length" class="space-y-2">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow31.inspector.observed_status') }}
        </h3>
        <p class="text-[11px] leading-5 text-muted">
          {{ t('workflow31.inspector.status_hint') }}
        </p>
        <code class="block text-[10px] text-toned">{{
          projection.statusEvents.map((event) => event.code).join('\n')
        }}</code>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { InputBinding } from '../../../../contracts/workflow/3.1/workflow-source'
import type { PortProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorCommand, Node, NodeProjection } from '@/app/editor/EditorSession'
import GeneratedFieldEditor from '@/app/editor/GeneratedFieldEditor.vue'

const props = defineProps<{ node: Node | null; projection: NodeProjection | null }>()
const emit = defineEmits<{ command: [command: EditorCommand] }>()
const { t } = useI18n()

function setLabel(event: Event): void {
  if (!props.node) return
  emit('command', {
    kind: 'set-node-label',
    nodeId: props.node.id,
    label: (event.target as HTMLInputElement).value,
  })
}

function setLiteral(portId: string, event: Event): void {
  if (!props.node) return
  emit('command', {
    kind: 'bind-value',
    nodeId: props.node.id,
    portId,
    value: (event.target as HTMLInputElement).value,
  })
}

function acceptsInline(port: PortProjection): boolean {
  return port.type.representations.some((representation) => representation.kind === 'inline-json')
}

function literalText(binding: InputBinding | undefined): string {
  return binding?.kind === 'value' && typeof binding.value === 'string' ? binding.value : ''
}

function literalPlaceholder(port: PortProjection): string {
  if (port.hasDefault && typeof port.default === 'string') return port.default
  return t(
    port.binding === 'required'
      ? 'workflow31.inspector.required_value'
      : 'workflow31.inspector.optional_value',
  )
}
</script>
