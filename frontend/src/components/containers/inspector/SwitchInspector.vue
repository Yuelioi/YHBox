<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphNode, GraphEdge } from '@/lib/backend'
import { useConfirm } from '@/composables/useConfirm'
import { useNodeRegistryStore } from '@/stores/nodeRegistry'

const { t } = useI18n()

// 已知错误码作下拉建议 (自由文本仍可输入任意 case 值 — creatable).
const nodeRegistry = useNodeRegistryStore()
const caseSuggestions = computed(() => nodeRegistry.errorCodes)

const props = defineProps<{
  node: GraphNode
  edges: GraphEdge[]
}>()

const emit = defineEmits<{
  (e: 'update', config: Record<string, any>): void
}>()

const { confirm: confirmDialog } = useConfirm()

interface CaseRow {
  /** stable UUID — decoupled from the case value so edits don't cause key flicker */
  id: string
  value: string
}

// Initialise rows from node.config.cases
function buildRows(): CaseRow[] {
  const rawCases = Array.isArray(props.node.config?.cases) ? props.node.config.cases : []
  return (rawCases as unknown[]).map((v) => ({
    id: crypto.randomUUID(),
    value: typeof v === 'string' ? v : '',
  }))
}

const rows = ref<CaseRow[]>(buildRows())

// Sync rows → config.cases whenever they change (deep watch)
watch(
  rows,
  () => {
    emit('update', {
      ...(props.node.config ?? {}),
      cases: rows.value.map((r) => r.value),
    })
  },
  { deep: true },
)

// Re-initialise rows when the selected node changes (user clicks a different Switch node)
watch(
  () => props.node.id,
  () => {
    rows.value = buildRows()
  },
)

// Dangling edges: out-pin `nodeId.caseValue` still referenced but case no longer exists
const danglingEdges = computed(() => {
  const currentPins = new Set(rows.value.map((r) => r.value).filter(Boolean))
  currentPins.add('default')
  return props.edges.filter((e) => {
    const dot = e.from.indexOf('.')
    if (dot < 0) return false
    const fromNode = e.from.slice(0, dot)
    const fromPin = e.from.slice(dot + 1)
    return fromNode === props.node.id && !currentPins.has(fromPin)
  })
})

function addCase() {
  rows.value.push({ id: crypto.randomUUID(), value: '' })
}

async function removeCase(i: number) {
  const caseValue = rows.value[i].value
  const pinKey = `${props.node.id}.${caseValue}`
  const affected = props.edges.filter((e) => e.from === pinKey)

  if (affected.length > 0) {
    const yes = await confirmDialog({
      title: t('node.Switch.inspector.delete_confirm_title', { name: caseValue }),
      description: t('node.Switch.inspector.delete_confirm_desc', { count: affected.length }),
      color: 'error',
      confirmText: t('common.delete'),
      cancelText: t('common.cancel'),
    })
    if (yes !== true) return
    // Remove affected edges by mutating the prop array in place (same pattern as WindowTarget / PlayClip)
    for (const edge of affected) {
      const idx = props.edges.indexOf(edge)
      if (idx >= 0) props.edges.splice(idx, 1)
    }
  }

  rows.value.splice(i, 1)
}

// Value 是 Spec.Input → 正源 config.literal.Value (input-editing-unification guardrail)。
function updateValue(v: string) {
  const cfg = props.node.config ?? {}
  emit('update', {
    ...cfg,
    literal: { ...((cfg.literal as Record<string, any>) ?? {}), Value: v },
  })
}

function currentValue(): string {
  const cfg = props.node.config as any
  // 正源 literal.Value; fallback 顶层 Value/value (旧数据), 镜像后端 PinValue。
  const v = cfg?.literal?.Value ?? cfg?.Value ?? cfg?.value
  return typeof v === 'string' ? v : ''
}
</script>

<template>
  <div class="space-y-4">
    <!-- Value expression -->
    <div class="space-y-1">
      <label class="block text-xs text-toned">{{ t('node.Switch.inspector.value_label') }}</label>
      <UInput
        :model-value="currentValue()"
        placeholder="$vars.state"
        @update:model-value="updateValue"
      />
      <p class="text-[11px] text-dimmed leading-snug">
        {{ t('node.Switch.inspector.value_hint') }}
      </p>
    </div>

    <!-- Cases list -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <label class="text-xs text-toned">{{ t('node.Switch.inspector.cases_label') }}</label>
        <UBadge size="xs" color="neutral" variant="soft">{{ t('node.Switch.inspector.cases_count', { n: rows.length }) }}</UBadge>
      </div>

      <!-- Dangling edge warning -->
      <div
        v-if="danglingEdges.length > 0"
        class="rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2 flex items-start gap-2"
      >
        <UIcon name="i-tabler-alert-triangle" class="size-3.5 text-amber-300 shrink-0 mt-0.5" />
        <p class="text-[11px] text-amber-300 leading-snug">
          {{ t('node.Switch.inspector.dangling_warn', { n: danglingEdges.length }) }}
        </p>
      </div>

      <!-- Case rows -->
      <div v-if="rows.length === 0" class="text-[11px] text-dimmed italic">
        {{ t('node.Switch.inspector.empty') }}
      </div>
      <div v-else class="space-y-1.5">
        <div
          v-for="(row, i) in rows"
          :key="row.id"
          class="flex gap-2 items-center"
        >
          <UInputMenu
            v-model="row.value"
            :create-item="'always'"
            :items="caseSuggestions"
            size="sm"
            :placeholder="t('node.Switch.inspector.case_placeholder')"
            class="flex-1"
            @create="(v: string) => (row.value = v)"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="error"
            icon="i-tabler-trash"
            :title="t('node.Switch.inspector.delete_case_title')"
            @click="removeCase(i)"
          />
        </div>
      </div>

      <!-- Add case button -->
      <UButton
        size="xs"
        variant="soft"
        color="primary"
        icon="i-tabler-plus"
        @click="addCase"
      >
        {{ t('node.Switch.inspector.add_case') }}
      </UButton>

      <p class="text-[10px] text-dimmed leading-snug">
        {{ t('node.Switch.inspector.footer_pre') }}<code class="font-mono">default</code> {{ t('node.Switch.inspector.footer_post') }}
      </p>
    </div>
  </div>
</template>
