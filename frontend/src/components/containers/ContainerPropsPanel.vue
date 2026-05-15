<template>
  <div v-if="!container" class="text-sm text-dimmed">加载中...</div>

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
          {{ container.name || '(未命名)' }}
        </h3>
        <p class="text-[11px] text-dimmed mt-0.5">
          {{ container.graph.nodes.length }} 个节点 · {{ container.graph.edges.length }} 条边
        </p>
      </div>
    </header>

    <!-- Section: 基本信息 -->
    <section class="mb-6">
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">
        基本信息
      </h4>

      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="block text-xs text-toned">名称</label>
          <UInput
            :model-value="container.name"
            size="md"
            class="w-full"
            placeholder="给容器起个名字"
            @update:model-value="$emit('update', { name: String($event) })"
          />
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs text-toned">触发热键</label>
          <UInput
            :model-value="container.hotkey"
            size="md"
            class="w-full"
            placeholder="如 Ctrl+Shift+1"
            @update:model-value="$emit('update', { hotkey: String($event) })"
          />
          <p class="text-[11px] text-dimmed leading-snug">
            按下热键立即运行此容器一次。留空则需手动触发。
          </p>
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs text-toned">描述</label>
          <UTextarea
            :model-value="container.description"
            size="md"
            :rows="2"
            class="w-full"
            placeholder="可选 · 给自己或队友看的说明"
            @update:model-value="$emit('update', { description: String($event) })"
          />
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs text-toned">分类</label>
          <UInput
            :model-value="container.category"
            size="md"
            class="w-full"
            placeholder="可选 · 如「日常」「副本」"
            @update:model-value="$emit('update', { category: String($event) })"
          />
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs text-toned">运行模式</label>
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
            后台：PostMessage 直发，不抢焦点（默认，可在其他窗口工作时跑）。<br />
            前台：启动前自动激活游戏窗口（必须前台焦点的脚本，如 SendInput）。
          </p>
        </div>
      </div>
    </section>

    <!-- Section: 变量 -->
    <section class="pt-5 border-t border-default">
      <div class="flex items-center justify-between mb-3">
        <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">
          变量
          <span class="text-toned ml-1">({{ container.vars?.length ?? 0 }})</span>
        </h4>
        <UButton size="xs" variant="soft" color="primary" icon="i-tabler-plus" @click="addVar"
          >添加</UButton
        >
      </div>

      <p
        v-if="(container.vars?.length ?? 0) === 0"
        class="text-[11px] text-dimmed leading-relaxed py-2"
      >
        声明变量后可在节点 config 表达式里用
        <code class="bg-elevated/60 px-1 rounded text-toned font-mono text-[10px]"
          >$vars.&lt;name&gt;</code
        >
        引用。
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
              placeholder="变量名"
              class="flex-1"
              @update:model-value="updateVar(i, 'name', String($event))"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-trash"
              :title="`删除变量 ${v.name || ''}`"
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
import type { Container, VarDecl } from '@/lib/backend'

const props = defineProps<{ container: Container | null }>()
const emit = defineEmits<{ update: [patch: Partial<Container>] }>()

const varTypes = [
  { label: 'number', value: 'number' },
  { label: 'bool', value: 'bool' },
  { label: 'string', value: 'string' },
  { label: 'point', value: 'point' },
]

const runModes = [
  { label: '后台运行（默认）', value: 'background' },
  { label: '前台运行（自动激活游戏）', value: 'foreground' },
]

function defaultPlaceholder(type: string): string {
  switch (type) {
    case 'number':
      return '0'
    case 'bool':
      return 'true / false'
    case 'string':
      return '空字符串'
    case 'point':
      return '{"x":0,"y":0}'
  }
  return '默认值'
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
