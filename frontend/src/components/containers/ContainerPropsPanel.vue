<template>
  <div v-if="!container" class="text-sm text-dimmed">{{ t('common.loading') }}</div>

  <div v-else>
    <!-- Header -->
    <header class="flex items-center gap-3 pb-4 mb-4 border-b border-default">
      <div
        class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-primary/15 border border-primary/40"
      >
        <UIcon name="i-tabler-schema" class="size-5 text-primary" />
      </div>
      <div class="min-w-0 flex-1">
        <h3 class="text-sm font-medium text-highlighted truncate leading-tight">
          {{ container.name || t('common.untitled') }}
        </h3>
        <p class="text-[11px] text-dimmed mt-0.5">
          {{ t('containers.node_and_edge_count', { n: container.graph.nodes.length, m: container.graph.edges.length }) }}
        </p>
      </div>
    </header>

    <!-- Section: 基本信息 -->
    <section class="mb-6">
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">
        {{ t('containers.basic_info') }}
      </h4>

      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="block text-xs text-toned">{{ t('common.name') }}</label>
          <UInput
            :model-value="container.name"
            size="md"
            class="w-full"
            :placeholder="t('containers.name_placeholder')"
            @update:model-value="$emit('update', { name: String($event) })"
          />
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs text-toned">{{ t('containers.hotkey_label') }}</label>
          <HotkeyCaptureInput
            :model-value="container.hotkey ?? ''"
            @update:model-value="(v: string) => $emit('update', { hotkey: v })"
          />
          <p class="text-[11px] text-dimmed leading-snug">
            {{ t('containers.hotkey_hint') }}
          </p>
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs text-toned">{{ t('common.description') }}</label>
          <UTextarea
            :model-value="container.description"
            size="md"
            :rows="2"
            class="w-full"
            :placeholder="t('containers.description_placeholder')"
            @update:model-value="$emit('update', { description: String($event) })"
          />
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs text-toned">{{ t('common.tags') }}</label>
          <UInputMenu
            :model-value="container.tags ?? []"
            multiple
            creatable
            :items="allTags"
            size="md"
            class="w-full"
            :placeholder="t('containers.add_tag_placeholder')"
            @update:model-value="(v: string[]) => $emit('update', { tags: v })"
          />
          <p class="text-[10px] text-dimmed">{{ t('containers.tags_hint') }}</p>
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs text-toned">{{ t('containers.run_mode_label') }}</label>
          <USelect
            :model-value="container.runMode || 'background'"
            size="md"
            class="w-full"
            :items="runModes"
            @update:model-value="
              $emit('update', {
                runMode: $event === 'foreground' ? 'foreground' : 'background',
              })
            "
          />
          <p class="text-[11px] text-dimmed leading-snug">
            {{ t('containers.run_mode_bg_hint') }}<br />
            {{ t('containers.run_mode_fg_hint') }}
          </p>
        </div>
      </div>
    </section>

    <!-- Section: 变量 -->
    <section class="pt-5 border-t border-default">
      <div class="flex items-center justify-between mb-3">
        <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">
          {{ t('containers.variables_section') }}
          <span class="text-toned ml-1">({{ container.vars?.length ?? 0 }})</span>
        </h4>
        <UButton size="xs" variant="soft" color="primary" icon="i-tabler-plus" @click="addVar"
          >{{ t('containers.add_var') }}</UButton
        >
      </div>

      <p
        v-if="(container.vars?.length ?? 0) === 0"
        class="text-[11px] text-dimmed leading-relaxed py-2"
      >
        {{ t('containers.var_declare_hint') }}
      </p>

      <div v-else class="space-y-2">
        <div
          v-for="(v, i) in container.vars ?? []"
          :key="i"
          class="p-3 rounded-md bg-elevated/40 border border-default/60 space-y-2"
        >
          <div class="flex items-center gap-1.5">
            <UInput
              :model-value="v.name"
              size="sm"
              :placeholder="t('containers.var_name_placeholder')"
              class="flex-1"
              @update:model-value="updateVar(i, 'name', String($event))"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-trash"
              :title="t('containers.var_delete_tip', { name: v.name || '' })"
              @click="removeVar(i)"
            />
          </div>
          <div class="flex items-center gap-1.5">
            <USelect
              :model-value="v.type"
              size="sm"
              :items="varTypes"
              class="w-24"
              :ui="{ content: 'min-w-[120px]' }"
              @update:model-value="updateVar(i, 'type', String($event))"
            />
            <UInput
              :model-value="formatDefault(v.default)"
              size="sm"
              :placeholder="defaultPlaceholder(v.type)"
              class="flex-1"
              @update:model-value="updateVar(i, 'default', $event)"
            />
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Container, VarDecl } from '@/lib/backend'
import { useContainersStore } from '@/stores/containers'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'

const { t } = useI18n()

const props = defineProps<{ container: Container | null }>()
const emit = defineEmits<{ update: [patch: Partial<Container>] }>()

// 所有容器现有 tags 聚合，供 UInputMenu autocomplete
const containersStore = useContainersStore()
const allTags = computed(() => {
  const set = new Set<string>()
  for (const c of containersStore.list ?? []) {
    for (const t of (c as any).tags ?? []) set.add(t)
  }
  return [...set]
})

const varTypes = [
  { label: 'number', value: 'number' },
  { label: 'bool', value: 'bool' },
  { label: 'string', value: 'string' },
  { label: 'point', value: 'point' },
]

const runModes = computed(() => [
  { label: t('containers.run_mode_bg'), value: 'background' },
  { label: t('containers.run_mode_fg'), value: 'foreground' },
])

function defaultPlaceholder(type: string): string {
  switch (type) {
    case 'number':
      return '0'
    case 'bool':
      return 'true / false'
    case 'string':
      return t('containers.empty_string_label')
    case 'point':
      return '{"x":0,"y":0}'
  }
  return t('containers.default_value_label')
}

function formatDefault(v: any): string {
  if (v == null) return ''
  if (typeof v === 'string') return v
  return JSON.stringify(v)
}

function parseDefault(input: any, type: string): any {
  const s = String(input ?? '').trim()
  if (s === '') return null
  switch (type) {
    case 'number': {
      const n = Number(s)
      return Number.isNaN(n) ? 0 : n
    }
    case 'bool':
      return s === 'true'
    case 'string':
      return s
    case 'point':
      try {
        return JSON.parse(s)
      } catch {
        return null
      }
  }
  return s
}

function addVar() {
  if (!props.container) return
  const vars = [...(props.container.vars ?? [])]
  vars.push({ name: '', type: 'number', default: 0 })
  emit('update', { vars })
}

function removeVar(i: number) {
  if (!props.container?.vars) return
  const vars = [...props.container.vars]
  vars.splice(i, 1)
  emit('update', { vars })
}

function updateVar(i: number, field: keyof VarDecl, val: any) {
  if (!props.container?.vars) return
  const vars = props.container.vars.map((v) => ({ ...v }))
  const v = vars[i]
  if (field === 'name') v.name = String(val)
  else if (field === 'type') {
    v.type = val as VarDecl['type']
    v.default = val === 'bool' ? false : val === 'string' ? '' : val === 'point' ? null : 0
  } else if (field === 'default') {
    v.default = parseDefault(val, v.type)
  }
  emit('update', { vars })
}
</script>
