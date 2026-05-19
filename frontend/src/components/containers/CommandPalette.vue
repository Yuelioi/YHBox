<!-- frontend/src/components/containers/CommandPalette.vue -->
<!-- Editor v2 C — VSCode-style command palette. Ctrl+K to open.
     Spec: editor-v2-quick-actions-design.md §3. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div class="bg-default" style="min-height: 40vh; max-height: 70vh; display: flex; flex-direction: column;">
        <div class="p-3 border-b border-default flex items-center gap-2">
          <UIcon name="i-tabler-command" class="size-4 text-primary" />
          <UInput
            ref="searchInputRef"
            v-model="query"
            placeholder="搜命令 (e.g. 对齐 / undo / 保存)..."
            size="md"
            class="flex-1"
            @keydown.escape.stop="close"
            @keydown.enter.stop="execFirst"
            @keydown.up.prevent="selectPrev"
            @keydown.down.prevent="selectNext"
          />
        </div>
        <div class="flex-1 overflow-y-auto p-2">
          <div v-if="filtered.length === 0" class="text-center text-xs text-dimmed py-8 italic">
            无匹配命令
          </div>
          <div v-else>
            <div v-for="(group, gIdx) in grouped" :key="group.name" :class="{'mt-2': gIdx > 0}">
              <div class="text-[10px] font-medium text-primary px-2 mb-1 uppercase tracking-wider">
                {{ group.label }}
              </div>
              <button
                v-for="(cmd, idx) in group.commands"
                :key="cmd.id"
                type="button"
                class="w-full text-left px-3 py-2 text-[11px] rounded flex items-center gap-3 hover:bg-elevated/60"
                :class="{'bg-elevated/60': activeIdx === flatIdx(gIdx, idx)}"
                :disabled="cmd.disabled === true"
                @click="execute(cmd)"
                @mouseenter="activeIdx = flatIdx(gIdx, idx)"
              >
                <UIcon v-if="cmd.icon" :name="cmd.icon" class="size-3.5" :class="cmd.disabled ? 'text-dimmed' : 'text-default'" />
                <span class="flex-1" :class="cmd.disabled ? 'text-dimmed' : ''">{{ cmd.label }}</span>
                <span v-if="cmd.shortcut" class="text-[10px] text-dimmed font-mono">{{ cmd.shortcut }}</span>
              </button>
            </div>
          </div>
        </div>
        <div class="px-3 py-1.5 border-t border-default text-[9px] text-dimmed flex justify-between">
          <span>↑↓ 导航 · Enter 执行 · Esc 关</span>
          <span>{{ filtered.length }} 命令</span>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'

export interface Command {
  id: string
  label: string
  group: string
  icon?: string
  shortcut?: string
  keywords?: string[]
  disabled?: boolean
  exec: () => void
}

const props = defineProps<{
  open: boolean
  commands: Command[]
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
}>()

const modelOpen = ref(props.open)
watch(() => props.open, v => { modelOpen.value = v })
watch(modelOpen, v => emit('update:open', v))

const query = ref('')
const activeIdx = ref(0)
const searchInputRef = ref<any>(null)

watch(() => modelOpen.value, async (v) => {
  if (v) {
    query.value = ''
    activeIdx.value = 0
    await nextTick()
    searchInputRef.value?.input?.focus?.()
  }
})

watch(query, () => { activeIdx.value = 0 })

const filtered = computed(() => {
  const q = query.value.toLowerCase().trim()
  if (!q) return props.commands
  return props.commands.filter(c => {
    const hay = `${c.label} ${c.id} ${(c.keywords ?? []).join(' ')}`.toLowerCase()
    return hay.includes(q)
  })
})

const GROUP_LABELS: Record<string, string> = {
  edit: '编辑',
  view: '视图',
  navigate: '导航',
  run: '运行',
  var: '变量',
  help: '帮助',
}

const grouped = computed(() => {
  const map = new Map<string, Command[]>()
  for (const c of filtered.value) {
    if (!map.has(c.group)) map.set(c.group, [])
    map.get(c.group)!.push(c)
  }
  return Array.from(map.entries())
    .map(([name, commands]) => ({ name, label: GROUP_LABELS[name] ?? name, commands }))
})

function flatIdx(gIdx: number, idx: number): number {
  let acc = 0
  for (let i = 0; i < gIdx; i++) acc += grouped.value[i].commands.length
  return acc + idx
}

const flatCommands = computed(() => filtered.value)

function selectPrev() {
  if (flatCommands.value.length === 0) return
  activeIdx.value = (activeIdx.value - 1 + flatCommands.value.length) % flatCommands.value.length
}
function selectNext() {
  if (flatCommands.value.length === 0) return
  activeIdx.value = (activeIdx.value + 1) % flatCommands.value.length
}

function execFirst() {
  const cmd = flatCommands.value[activeIdx.value]
  if (cmd && !cmd.disabled) execute(cmd)
}

function execute(cmd: Command) {
  if (cmd.disabled) return
  cmd.exec()
  close()
}

function close() {
  modelOpen.value = false
}
</script>
