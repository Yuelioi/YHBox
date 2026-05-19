<!-- frontend/src/components/containers/ContainerSettingsModal.vue -->
<!-- 容器基本信息 — 从右抽屉抽出, 顶部 ⚙ 设置 / Ctrl+, 触发. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-lg' }">
    <template #content>
      <div class="p-5 space-y-4 bg-default">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-settings" class="size-5 text-primary" />
          <h3 class="text-sm font-medium">容器设置</h3>
        </div>

        <div class="space-y-3">
          <UFormField label="名称" required>
            <UInput v-model="form.name" placeholder="容器名" size="sm" />
          </UFormField>

          <UFormField label="触发热键" hint="按下热键即运行此容器一次. 留空则需手动触发.">
            <HotkeyCaptureInput v-model="form.hotkey" />
          </UFormField>

          <UFormField label="描述">
            <UTextarea v-model="form.description" :rows="3" size="sm" placeholder="可选 — 给自己或队友看的说明" />
          </UFormField>

          <UFormField label="标签" hint="用于在容器列表筛选; 可自由命名">
            <UInputMenu v-model="form.tags" multiple :items="allTags" creatable size="sm" />
          </UFormField>

          <UFormField label="运行模式">
            <USelect v-model="form.runMode" :items="RUN_MODE_OPTIONS" size="sm" />
            <template #help>
              <p class="text-xs text-muted">
                后台: PostMessage 直发, 不抢焦点 (默认, 可在其他窗口工作时跑).<br>
                前台: 启动前自动激活游戏窗口 (必须前台焦点的脚本, 如 SendInput).
              </p>
            </template>
          </UFormField>
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <UButton variant="ghost" color="neutral" @click="modelOpen = false">取消</UButton>
          <UButton color="primary" icon="i-tabler-check" @click="onConfirm">保存</UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'

interface FormState {
  name: string
  hotkey: string
  description: string
  tags: string[]
  runMode: string
}

const props = defineProps<{
  open: boolean
  initial: FormState
  allTags: string[]
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  save: [v: FormState]
}>()

const modelOpen = ref(props.open)
watch(() => props.open, v => { modelOpen.value = v })
watch(modelOpen, v => emit('update:open', v))

const form = ref<FormState>({ ...props.initial })
watch(() => props.initial, v => { form.value = { ...v } }, { deep: true })

const RUN_MODE_OPTIONS = [
  { value: 'background', label: '后台运行（默认）' },
  { value: 'foreground', label: '前台运行' },
]

function onConfirm() {
  emit('save', { ...form.value })
  modelOpen.value = false
}
</script>
