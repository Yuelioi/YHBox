<script setup lang="ts">
// 动态输出声明编辑区 — spec.dynamicDataFields 节点 (AI) 通用.
// 编辑 config.Outputs[] (PascalCase Name/Type, 镜像后端 BindableFieldsForNode) —
// 每个声明字段成 Done 出口一个可绑 Data 字段, 在「输出」组逐个绑变量。
// 保留名 Text(模型原文恒有)不可声明。模式照 DynamicInputsEditor。
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphNode } from '@/lib/backend'

const { t } = useI18n()

const props = defineProps<{
  node: GraphNode
}>()

const emit = defineEmits<{
  (e: 'update', config: Record<string, any>): void
}>()

const NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/
const TYPE_ITEMS = ['String', 'Number', 'Integer', 'Bool', 'JSON', 'List']
const RESERVED = new Set(['Text'])

interface OutputRow {
  id: string
  name: string
  type: string
}

function buildRows(): OutputRow[] {
  const raw = Array.isArray(props.node.config?.Outputs) ? props.node.config.Outputs : []
  return (raw as Array<Record<string, unknown>>).map((d) => ({
    id: crypto.randomUUID(),
    name: typeof d?.Name === 'string' ? d.Name : '',
    type: typeof d?.Type === 'string' && d.Type !== '' ? (d.Type as string) : 'String',
  }))
}

const rows = ref<OutputRow[]>(buildRows())

// rows → config.Outputs 即时同步。空/非法/保留名的行不写进 config(镜像后端跳过)。
watch(
  rows,
  () => {
    emit('update', {
      ...props.node.config,
      Outputs: rows.value
        .filter((r) => NAME_RE.test(r.name) && !RESERVED.has(r.name))
        .map((r) => ({ Name: r.name, Type: r.type })),
    })
  },
  { deep: true },
)

watch(
  () => props.node.id,
  () => {
    rows.value = buildRows()
  },
)

const dupNames = computed<Set<string>>(() => {
  const seen = new Set<string>()
  const dup = new Set<string>()
  for (const r of rows.value) {
    if (r.name === '') continue
    if (seen.has(r.name)) dup.add(r.name)
    seen.add(r.name)
  }
  return dup
})

// 非法标识符 / 重名 / 保留名 → 标红。空行不红。
function rowInvalid(r: OutputRow): boolean {
  return (r.name !== '' && !NAME_RE.test(r.name)) || dupNames.value.has(r.name) || RESERVED.has(r.name)
}

function addRow() {
  rows.value.push({ id: crypto.randomUUID(), name: '', type: 'String' })
}

function removeRow(i: number) {
  rows.value.splice(i, 1)
}
</script>

<template>
  <div class="space-y-2">
    <div class="space-y-1.5">
      <div v-for="(row, i) in rows" :key="row.id" class="flex gap-2 items-center">
        <UInput
          v-model="row.name"
          size="sm"
          placeholder="Count"
          class="flex-1 font-mono"
          :color="rowInvalid(row) ? 'error' : undefined"
          :highlight="rowInvalid(row)"
        />
        <USelect v-model="row.type" :items="TYPE_ITEMS" size="sm" class="w-24" />
        <UButton
          size="xs"
          variant="ghost"
          color="error"
          icon="i-tabler-trash"
          :title="t('common.delete')"
          @click="removeRow(i)"
        />
      </div>
    </div>

    <UButton size="xs" variant="soft" color="primary" icon="i-tabler-plus" @click="addRow">
      {{ t('common.add') }}
    </UButton>
    <p class="text-[10px] text-dimmed leading-snug">{{ t('inspector.dyn_outputs_hint') }}</p>
  </div>
</template>
