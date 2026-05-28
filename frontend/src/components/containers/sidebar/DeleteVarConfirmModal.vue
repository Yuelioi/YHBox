<!-- frontend/src/components/containers/sidebar/DeleteVarConfirmModal.vue -->
<!-- 删除变量 — 2 选项: 保留 invalid 节点 / cascade 删. 默认保留. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-md' }">
    <template #content>
      <div class="p-5 space-y-4 bg-default">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-alert-triangle" class="size-5 text-warning" />
          <h3 class="text-sm font-medium">删除变量 '{{ varName }}'</h3>
        </div>

        <p class="text-xs text-muted">
          有 <strong class="text-warning">{{ refCount }}</strong> 个节点引用此变量:
        </p>

        <div class="max-h-32 overflow-y-auto bg-elevated/40 rounded p-2 text-[10px] font-mono space-y-0.5">
          <div v-for="id in refIDs.slice(0, 20)" :key="id" class="text-dimmed">{{ id }}</div>
          <div v-if="refIDs.length > 20" class="text-dimmed italic">… 还有 {{ refIDs.length - 20 }} 个</div>
        </div>

        <p class="text-[11px] text-muted">选择处理方式:</p>

        <div class="flex flex-col gap-2 pt-1">
          <UButton color="primary" variant="soft" icon="i-tabler-shield" @click="onConfirm(false)">
            保留引用节点 (推荐)
            <template #trailing>
              <span class="text-[10px] text-dimmed ml-1">— 节点标红 INVALID_VAR_REF, 不删 graph</span>
            </template>
          </UButton>
          <UButton color="error" variant="soft" icon="i-tabler-trash" @click="onConfirm(true)">
            一并删除 {{ refCount }} 个节点 + 边
          </UButton>
          <UButton color="neutral" variant="ghost" @click="modelOpen = false">取消</UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'

const props = defineProps<{
  open: boolean
  varName: string
  refIDs: string[]
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  confirm: [cascade: boolean]
}>()

const modelOpen = useDialogOpen(props, emit)

const refCount = computed(() => props.refIDs.length)

function onConfirm(cascade: boolean) {
  emit('confirm', cascade)
  modelOpen.value = false
}
</script>
