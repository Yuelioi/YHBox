<script setup lang="ts">
// 动态输入声明编辑区 — spec.dynamicInputs 节点 (Expr / Script) 通用.
// 编辑 config.Inputs[] (PascalCase Name/Type/Var/Scope, 镜像后端 ParseDynamicInputDecls)。
// 每行有「来源」: 绑定变量 (默认, 不渲染引脚, 值直读变量) / 连线 (传统 data-in pin)。
// 画布引脚 / literal 区 / code 补全 (dynamicInputNames) 据 config 联动。
// 模式照 SwitchInspector: rows ref ↔ config 即时深 watch 同步 + add/remove 按钮.
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphNode } from '@/lib/backend'
import type { VarType } from '@/lib/variableRef'
import VarNameInput from '../inline/VarNameInput.vue'

const { t } = useI18n()

const props = defineProps<{
  node: GraphNode
  declaredVars: { name: string; type: VarType }[]
}>()

const emit = defineEmits<{
  (e: 'update', config: Record<string, any>): void
  (e: 'declare-var', a: { name: string; type: VarType; default: unknown }): void
}>()

// 名字校验 — 镜像后端动态 pin 名约束 (标识符).
const NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/
const TYPE_ITEMS = ['number', 'string', 'bool', 'point', 'any']
const SCOPE_ITEMS = ['auto', 'local', 'global'] // 标识符保留不翻译 (node-spec-style §10)

interface InputRow {
  /** stable UUID — decoupled from the name so edits don't cause key flicker */
  id: string
  name: string
  type: string
  source: 'var' | 'wire'
  var: string
  scope: string
}

const sourceItems = computed(() => [
  { value: 'var', label: t('inspector.dyn_input_source_var') },
  { value: 'wire', label: t('inspector.dyn_input_source_wire') },
])

function buildRows(): InputRow[] {
  const raw = Array.isArray(props.node.config?.Inputs) ? props.node.config.Inputs : []
  return (raw as Array<Record<string, unknown>>).map((d) => {
    const varName = typeof d?.Var === 'string' ? d.Var : ''
    return {
      id: crypto.randomUUID(),
      name: typeof d?.Name === 'string' ? d.Name : '',
      type: typeof d?.Type === 'string' && d.Type !== '' ? (d.Type as string) : 'any',
      source: varName !== '' ? 'var' : 'wire',
      var: varName,
      scope: typeof d?.Scope === 'string' && d.Scope !== '' ? (d.Scope as string) : 'auto',
    }
  })
}

const rows = ref<InputRow[]>(buildRows())

// rows → config.Inputs 即时深 watch 同步 (跟 SwitchInspector 一致)。
// 不写入: 空/非法 Name 的行 (半截名字不挂 pin); 变量模式但还没选变量的行
// (写 Var:"" 会被后端当连线模式渲染出悬空引脚)。重名照写, validator 报。
watch(
  rows,
  () => {
    emit('update', {
      ...(props.node.config ?? {}),
      Inputs: rows.value
        .filter((r) => NAME_RE.test(r.name))
        .filter((r) => r.source === 'wire' || r.var.trim() !== '')
        .map((r) =>
          r.source === 'var'
            ? { Name: r.name, Type: r.type, Var: r.var.trim(), Scope: r.scope }
            : { Name: r.name, Type: r.type },
        ),
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

// 选定绑定变量 → Type 自动回填该变量类型 (validator 类型兼容 + 表达式类型提示都吃它)。
function onPickVar(row: InputRow, name: string) {
  row.var = name
  const decl = props.declaredVars.find((v) => v.name === name)
  if (decl) row.type = decl.type
}

function addRow() {
  rows.value.push({
    id: crypto.randomUUID(),
    name: '',
    type: 'any',
    source: 'var', // 用户拍板: 绑定变量是默认; 连线留作非变量数据源的兜底
    var: '',
    scope: 'auto',
  })
}

function removeRow(i: number) {
  rows.value.splice(i, 1)
}
</script>

<template>
  <div class="space-y-2">
    <div class="space-y-1.5">
      <div
        v-for="(row, i) in rows"
        :key="row.id"
        class="space-y-1 rounded-md border border-default/60 p-1.5"
      >
        <div class="flex gap-1.5 items-center">
          <UInput
            v-model="row.name"
            size="sm"
            placeholder="hp"
            class="flex-1 min-w-0 font-mono"
            :color="rowInvalid(row) ? 'error' : undefined"
            :highlight="rowInvalid(row)"
          />
          <USelect v-model="row.source" :items="sourceItems" size="sm" class="w-24 shrink-0" />
          <UButton
            size="xs"
            variant="ghost"
            color="error"
            icon="i-tabler-trash"
            :title="t('common.delete')"
            @click="removeRow(i)"
          />
        </div>
        <div v-if="row.source === 'var'" class="flex gap-1.5 items-center">
          <div class="flex-1 min-w-0">
            <VarNameInput
              :model-value="row.var"
              :declared-vars="declaredVars"
              @update:model-value="(v: string) => onPickVar(row, v)"
              @declare-var="(a) => { emit('declare-var', a); onPickVar(row, a.name) }"
            />
          </div>
          <USelect v-model="row.scope" :items="SCOPE_ITEMS" size="sm" class="w-20 shrink-0" />
        </div>
        <div v-else class="flex gap-1.5 items-center">
          <USelect v-model="row.type" :items="TYPE_ITEMS" size="sm" class="w-28 shrink-0" />
          <span class="text-[10px] text-dimmed">{{ t('inspector.dyn_input_wire_hint') }}</span>
        </div>
      </div>
    </div>

    <UButton size="xs" variant="soft" color="primary" icon="i-tabler-plus" @click="addRow">
      {{ t('common.add') }}
    </UButton>
  </div>
</template>
