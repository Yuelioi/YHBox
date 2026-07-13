<!-- VSCode-style command palette. Ctrl+K to open. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div
        class="bg-default"
        style="min-height: 40vh; max-height: 70vh; display: flex; flex-direction: column"
      >
        <div class="p-3 border-b border-default flex items-center gap-2">
          <UIcon name="i-tabler-command" class="size-4 text-primary" />
          <UInput
            ref="searchInputRef"
            v-model="query"
            :placeholder="t('editor.palette.search_placeholder')"
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
            {{ t('editor.palette.empty') }}
          </div>
          <div v-else>
            <div v-for="(group, gIdx) in grouped" :key="group.name" :class="{ 'mt-2': gIdx > 0 }">
              <div class="px-2 mb-1 text-xs font-medium text-primary">
                {{ group.label }}
              </div>
              <button
                v-for="(cmd, idx) in group.commands"
                :key="cmd.id"
                type="button"
                class="w-full text-left px-3 py-2 text-xs rounded flex items-center gap-3 hover:bg-elevated/60"
                :class="{ 'bg-elevated/60': activeIdx === flatIdx(gIdx, idx) }"
                :disabled="cmd.disabled === true"
                @click="execute(cmd)"
                @mouseenter="activeIdx = flatIdx(gIdx, idx)"
              >
                <UIcon
                  v-if="cmd.icon"
                  :name="cmd.icon"
                  class="size-3.5"
                  :class="cmd.disabled ? 'text-dimmed' : 'text-default'"
                />
                <span class="flex-1" :class="cmd.disabled ? 'text-dimmed' : ''">{{
                  cmd.label
                }}</span>
                <span v-if="cmd.shortcut" class="text-[11px] text-dimmed font-mono">{{
                  cmd.shortcut
                }}</span>
              </button>
            </div>
          </div>
        </div>
        <div
          class="px-3 py-1.5 border-t border-default text-[11px] text-dimmed flex justify-between"
        >
          <span>{{ t('editor.palette.hint') }}</span>
          <span>{{ t('editor.palette.count', { n: filtered.length }) }}</span>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'

const { t } = useI18n()

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

const modelOpen = useDialogOpen(props, emit)

const query = ref('')
const activeIdx = ref(0)
const searchInputRef = ref<any>(null)

useAutoFocusOnOpen(modelOpen, searchInputRef, {
  onOpen: () => {
    query.value = ''
    activeIdx.value = 0
  },
})

watch(query, () => {
  activeIdx.value = 0
})

const filtered = computed(() => {
  const q = query.value.toLowerCase().trim()
  if (!q) return props.commands
  return props.commands.filter((c) => {
    const hay = `${c.label} ${c.id} ${(c.keywords ?? []).join(' ')}`.toLowerCase()
    return hay.includes(q)
  })
})

const GROUP_KEYS: Record<string, string> = {
  edit: 'editor.palette.group.edit',
  view: 'editor.palette.group.view',
  navigate: 'editor.palette.group.navigate',
  run: 'editor.palette.group.run',
  var: 'editor.palette.group.var',
  help: 'editor.palette.group.help',
}

const grouped = computed(() => {
  const map = new Map<string, Command[]>()
  for (const c of filtered.value) {
    if (!map.has(c.group)) map.set(c.group, [])
    map.get(c.group)!.push(c)
  }
  return Array.from(map.entries()).map(([name, commands]) => ({
    name,
    label: GROUP_KEYS[name] ? t(GROUP_KEYS[name]) : name,
    commands,
  }))
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
