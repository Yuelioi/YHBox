<template>
  <header
    data-testid="workflow-editor-toolbar"
    class="flex h-13 shrink-0 items-center gap-2 overflow-hidden whitespace-nowrap border-b border-default bg-default px-3"
  >
    <UButton
      data-testid="workflow-editor-back"
      class="shrink-0"
      icon="i-tabler-arrow-left"
      color="neutral"
      variant="ghost"
      size="xs"
      :aria-label="t('workflow.editor.back')"
      @click="emit('back')"
    />

    <div
      data-testid="workflow-editor-context"
      class="flex min-w-0 flex-1 items-center gap-1 overflow-hidden"
    >
      <span class="min-w-0 max-w-56 truncate text-sm font-medium text-highlighted">
        {{ name }}
      </span>
      <slot name="breadcrumbs" />
      <span class="hidden shrink-0 font-mono text-[10px] text-dimmed min-[1180px]:inline">
        {{ t('workflow.editor.revision', { n: revision }) }}
      </span>
      <span
        v-if="dirty"
        data-testid="workflow-unsaved"
        class="shrink-0 text-[11px] font-medium text-warning"
      >
        {{ t('workflow.editor.unsaved') }}
      </span>
    </div>

    <div data-testid="workflow-editor-actions" class="flex shrink-0 items-center gap-1">
      <UBadge
        v-if="model.recordingStatusKey"
        class="shrink-0"
        color="primary"
        variant="soft"
        size="sm"
      >
        {{ t(model.recordingStatusKey) }}
      </UBadge>
      <UButton
        v-for="item in model.contextual"
        :key="item.command"
        :data-testid="item.testId"
        class="shrink-0"
        :label="t(item.labelKey, item.labelParams ?? {})"
        :icon="item.icon"
        :color="item.color ?? 'neutral'"
        :variant="item.variant ?? 'ghost'"
        size="xs"
        :disabled="item.disabled"
        :loading="item.loading"
        :aria-pressed="item.active"
        @click="emit('command', item.command)"
      />

      <div class="mx-1 h-5 w-px shrink-0 bg-default" />
      <UButton
        v-for="item in model.editing"
        :key="item.command"
        :data-testid="item.testId"
        class="shrink-0"
        :icon="item.icon"
        color="neutral"
        variant="ghost"
        size="xs"
        :disabled="item.disabled"
        :aria-label="t(item.labelKey)"
        :title="
          item.command === 'find-node' ? t('workflow.node_search.shortcut') : t(item.labelKey)
        "
        @click="emit('command', item.command)"
      />

      <div class="flex shrink-0 items-center">
        <slot name="target" />
      </div>

      <UButton
        v-for="item in model.primary"
        :key="item.command"
        :data-testid="item.testId"
        class="shrink-0"
        :label="t(item.labelKey, item.labelParams ?? {})"
        :icon="item.icon"
        :color="item.color ?? 'neutral'"
        :variant="item.variant ?? 'soft'"
        size="xs"
        :disabled="item.disabled"
        :loading="item.loading"
        @click="emit('command', item.command)"
      />

      <UDropdownMenu :items="toolItems">
        <UButton
          data-testid="workflow-editor-tools"
          class="shrink-0"
          :label="t('workflow.editor.tools')"
          icon="i-tabler-tool"
          :color="model.toolsNeedAttention ? 'warning' : 'neutral'"
          :variant="model.toolsNeedAttention ? 'soft' : 'ghost'"
          size="xs"
          :aria-label="t('workflow.editor.tools')"
        />
        <template #item="{ item }">
          <span
            :data-testid="item.testId"
            class="flex min-w-0 flex-1 items-center justify-start gap-2 text-left"
            :aria-pressed="item.checked || undefined"
          >
            <UIcon :name="item.icon" class="size-4 shrink-0" />
            <span class="min-w-0 flex-1 truncate text-left">{{ item.label }}</span>
          </span>
        </template>
      </UDropdownMenu>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  buildEditorToolbarModel,
  type EditorToolbarCommand,
  type EditorToolbarContext,
} from './editorToolbarModel'

const props = defineProps<{
  name: string
  revision: number
  dirty: boolean
  context: Omit<EditorToolbarContext, 'dirty'>
}>()
const emit = defineEmits<{
  back: []
  command: [command: EditorToolbarCommand]
}>()
const { t } = useI18n()
const model = computed(() =>
  buildEditorToolbarModel({
    ...props.context,
    dirty: props.dirty,
  }),
)
const toolItems = computed(() =>
  model.value.tools.map((group) =>
    group.map((item) => ({
      label: t(item.labelKey, item.labelParams ?? {}),
      icon: item.icon,
      color: item.color,
      disabled: item.disabled,
      type: item.active ? ('checkbox' as const) : ('link' as const),
      checked: item.active,
      testId: item.testId,
      onSelect: () => emit('command', item.command),
    })),
  ),
)
</script>
