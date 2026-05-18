<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { GraphNode, GraphEdge } from '@/lib/backend'
import { useConfirm } from '@/composables/useConfirm'

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
      title: `删除 case "${caseValue}"？`,
      description: `该 case 的出口边 (${affected.length} 条) 将断开，需手动重连。`,
      color: 'error',
      confirmText: '删除',
      cancelText: '取消',
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

function updateValue(v: string) {
  emit('update', {
    ...(props.node.config ?? {}),
    value: v,
  })
}

function currentValue(): string {
  const v = props.node.config?.value
  return typeof v === 'string' ? v : ''
}
</script>

<template>
  <div class="space-y-4">
    <!-- Value expression -->
    <div class="space-y-1">
      <label class="block text-xs text-toned">Value 表达式</label>
      <UInput
        :model-value="currentValue()"
        placeholder="$vars.state"
        @update:model-value="updateValue"
      />
      <p class="text-[11px] text-dimmed leading-snug">
        运行时将此表达式与 cases 逐一比对，匹配则走对应出口，无匹配走 default。
      </p>
    </div>

    <!-- Cases list -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <label class="text-xs text-toned">Cases</label>
        <UBadge size="xs" color="neutral" variant="soft">{{ rows.length }} 项</UBadge>
      </div>

      <!-- Dangling edge warning -->
      <div
        v-if="danglingEdges.length > 0"
        class="rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2 flex items-start gap-2"
      >
        <UIcon name="i-tabler-alert-triangle" class="size-3.5 text-amber-300 shrink-0 mt-0.5" />
        <p class="text-[11px] text-amber-300 leading-snug">
          {{ danglingEdges.length }} 条边引用了已删除/改名的 case pin，需手动重连或断开。
        </p>
      </div>

      <!-- Case rows -->
      <div v-if="rows.length === 0" class="text-[11px] text-dimmed italic">
        暂无 case — 点下方按钮添加
      </div>
      <div v-else class="space-y-1.5">
        <div
          v-for="(row, i) in rows"
          :key="row.id"
          class="flex gap-2 items-center"
        >
          <UInput
            v-model="row.value"
            size="sm"
            placeholder="case 字符串 (如 IDLE)"
            class="flex-1"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="error"
            icon="i-tabler-trash"
            title="删除此 case"
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
        添加 case
      </UButton>

      <p class="text-[10px] text-dimmed leading-snug">
        case 名即出口 pin 名。改名后已连接的边需手动重连。<code class="font-mono">default</code> 出口始终存在。
      </p>
    </div>
  </div>
</template>
