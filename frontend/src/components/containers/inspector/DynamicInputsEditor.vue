<script setup lang="ts">
// 动态输入声明编辑区 — spec.dynamicInputs 节点 (Expr / Script) 通用.
// 编辑 config.Inputs[] (PascalCase Name/Type, 镜像后端 ParseDynamicInputDecls),
// 画布 data-in 引脚 / Inspector literal 区 / code 补全 (dynamicInputNames) 据此联动.
// 模式照 SwitchInspector: rows ref ↔ config 即时深 watch 同步 + add/remove 按钮.
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

// 名字校验 — 镜像后端动态 pin 名约束 (标识符).
const NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/
const TYPE_ITEMS = ['number', 'string', 'bool', 'point', 'any']

interface InputRow {
  /** stable UUID — decoupled from the name so edits don't cause key flicker */
  id: string
  name: string
  type: string
}

function buildRows(): InputRow[] {
  const raw = Array.isArray(props.node.config?.Inputs) ? props.node.config.Inputs : []
  return (raw as Array<Record<string, unknown>>).map((d) => ({
    id: crypto.randomUUID(),
    name: typeof d?.Name === 'string' ? d.Name : '',
    type: typeof d?.Type === 'string' && d.Type !== '' ? (d.Type as string) : 'any',
  }))
}

const rows = ref<InputRow[]>(buildRows())

// rows → config.Inputs 即时深 watch 同步 (跟 SwitchInspector 一致)。
// 空/非法 Name 的行不写进 config.Inputs (镜像后端空 Name 跳过) — 半截名字不挂 pin,
// 避免每个 keystroke 重命名引脚造成边断裂。重名照写, validator 报重名错。
watch(
  rows,
  () => {
    emit('update', {
      ...(props.node.config ?? {}),
      Inputs: rows.value
        .filter((r) => NAME_RE.test(r.name))
        .map((r) => ({ Name: r.name, Type: r.type })),
    })
  },
  { deep: true },
)

// 切换选中节点 → 从新节点 config 重建行
watch(
  () => props.node.id,
  () => {
    rows.value = buildRows()
  },
)

// 重名集合 — 标红用 (写入不拦, validator 报)。
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

// 非法 (非空但不匹配标识符) 或重名 → 标红。空行不红 (刚添加还没填)。
function rowInvalid(r: InputRow): boolean {
  return (r.name !== '' && !NAME_RE.test(r.name)) || dupNames.value.has(r.name)
}

function addRow() {
  rows.value.push({ id: crypto.randomUUID(), name: '', type: 'number' })
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
          placeholder="hp"
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
  </div>
</template>
