<template>
  <div class="px-8 py-6 space-y-6">
    <!-- Top row: 时长在左，主操作按钮放右（统一布局） -->
    <div class="flex items-center justify-between gap-4">
      <div v-if="pianoStore.currentMidi" class="text-sm text-muted">
        <span class="text-dimmed mr-2">时长</span>
        <span class="tabular-nums text-default font-medium">{{
          fmtMs(pianoStore.currentMidi.totalMs)
        }}</span>
      </div>
      <div v-else />
      <BotControls
        :state="pianoStore.state"
        :can-start="canStart"
        :disabled-reason="disabledReason"
        @start="pianoStore.start()"
        @pause="pianoStore.pause()"
        @resume="pianoStore.resume()"
        @stop="pianoStore.stop()"
      />
    </div>

    <!-- MIDI source selection -->
    <div class="rounded-xl bg-default border border-default p-5 space-y-4">
      <h3 class="text-sm font-medium text-highlighted flex items-center gap-2">
        <UIcon name="i-tabler-playlist" class="size-4 text-dimmed" />
        MIDI 曲谱
      </h3>

      <div class="grid grid-cols-[5rem_1fr] gap-x-3 gap-y-3 items-center">
        <label class="text-xs text-dimmed">内置曲库</label>
        <USelect
          v-model="builtinPick"
          :items="builtinItems"
          placeholder="(选一首内置曲)"
          class="w-full max-w-xs"
          @update:model-value="onBuiltinPick"
        />

        <label class="text-xs text-dimmed">外部文件</label>
        <div class="flex items-center gap-2 w-full max-w-xl">
          <UInput
            v-model="externalPath"
            placeholder="C:\path\to\song.mid (直接粘贴文件路径)"
            class="flex-1"
          />
          <UButton color="neutral" icon="i-tabler-folder-open" @click="onLoadFile">加载</UButton>
        </div>
      </div>

      <div
        v-if="pianoStore.currentMidi"
        class="pt-3 border-t border-default/60 flex items-center gap-3 text-xs"
      >
        <UIcon name="i-tabler-circle-check" class="size-3.5 text-primary" />
        <span class="text-toned truncate">{{ pianoStore.currentMidi.path }}</span>
        <span class="text-dimmed">·</span>
        <span class="text-dimmed">{{ pianoStore.currentMidi.tracks.length }} 轨</span>
      </div>
    </div>

    <!-- Config grid -->
    <div class="rounded-xl bg-default border border-default p-5 space-y-4">
      <h3 class="text-sm font-medium text-highlighted flex items-center gap-2">
        <UIcon name="i-tabler-adjustments-horizontal" class="size-4 text-dimmed" />
        弹奏配置
      </h3>

      <div class="grid grid-cols-2 gap-x-6 gap-y-4">
        <!-- Mode -->
        <div>
          <label class="block text-xs text-dimmed mb-1.5">键位</label>
          <USelect
            v-model="mode"
            :items="modeItems"
            class="w-full"
            @update:model-value="onModeChange"
          />
        </div>
        <!-- TrackPick: 占满整行，轨道名往往很长（"Track 3: Acoustic Grand Piano (1247 音符)"） -->
        <div class="col-span-2">
          <label class="block text-xs text-dimmed mb-1.5">选轨</label>
          <USelect
            v-model="trackPick"
            :items="trackItems"
            class="w-full"
            @update:model-value="onTrackPickChange"
          />
          <p class="text-[11px] text-dimmed mt-1">
            智能选轨：按音符数 + 音域挑最像主旋律的那条；全部
            track：把所有轨道合在一起弹；或手动指定某条。
          </p>
        </div>
        <!-- AutoTranspose -->
        <div class="flex items-center justify-between">
          <div>
            <label class="text-xs text-dimmed">自动对齐音域</label>
            <p class="text-[11px] text-dimmed mt-0.5">整曲八度平移让最多音落进游戏范围</p>
          </div>
          <USwitch v-model="autoTranspose" @update:model-value="onAutoTransposeChange" />
        </div>
        <!-- MelodyOnly -->
        <div class="flex items-center justify-between">
          <div>
            <label class="text-xs text-dimmed">只弹主旋律</label>
            <p class="text-[11px] text-dimmed mt-0.5">同时音只保留最高音（去伴奏）</p>
          </div>
          <USwitch v-model="melodyOnly" @update:model-value="onMelodyOnlyChange" />
        </div>
        <!-- OctaveOffset -->
        <div class="col-span-2 flex items-center justify-between">
          <div>
            <label class="text-xs text-dimmed">八度微调</label>
            <p class="text-[11px] text-dimmed mt-0.5">在自动偏移基础上手动 ±2 个八度</p>
          </div>
          <UInputNumber
            v-model="octaveOffset"
            :min="-2"
            :max="2"
            :step="1"
            class="w-24"
            @update:model-value="onOctaveOffsetChange"
          />
        </div>
      </div>
    </div>

    <!-- Progress (visible only when playing or has totalMs) -->
    <div
      v-if="pianoStore.progress.totalMs > 0"
      class="rounded-xl bg-default border border-default p-5 space-y-3"
    >
      <div class="flex items-center justify-between text-xs">
        <span class="text-dimmed">演奏进度</span>
        <span class="tabular-nums text-toned">
          {{ fmtMs(pianoStore.progress.curMs) }} / {{ fmtMs(pianoStore.progress.totalMs) }}
        </span>
      </div>
      <USlider
        v-model="seekPct"
        :min="0"
        :max="1000"
        :disabled="pianoStore.state === 'idle'"
        @update:model-value="onSeekCommit"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useToast } from '@nuxt/ui/composables'
import { Dialogs } from '@wailsio/runtime'
import { usePianoStore } from '@/stores/piano'
import { useGameStore } from '@/stores/game'
import { useSettingsStore } from '@/stores/settings'
import BotControls from '@/components/BotControls.vue'

const pianoStore = usePianoStore()
const gameStore = useGameStore()
const settingsStore = useSettingsStore()
const toast = useToast()

// Load library + restore last external MIDI on mount
onMounted(async () => {
  await pianoStore.loadLibrary()
  const last = settingsStore.data?.piano.lastMidiPath
  if (last) {
    externalPath.value = last
    await pianoStore.loadFile(last)
  }
})

// Builtin library picker
const builtinItems = computed(() =>
  pianoStore.library.map((e) => ({ label: e.display, value: e.asset })),
)
const builtinPick = ref<string>('')

async function onBuiltinPick(asset: string) {
  if (!asset) return
  const r = await pianoStore.loadBuiltin(asset)
  if (r) {
    externalPath.value = ''
    const name = (asset as string).replace(/^midi\//, '').replace(/\.mid$/, '')
    toast.add({ title: `已加载 ${name}`, color: 'success', icon: 'i-tabler-check' })
  }
}

// External file picker
const externalPath = ref<string>('')

// 加载按钮：input 空 → 弹原生选择对话框；input 有值 → 用 input 的路径。
// （Wails3 webview 里 <input type="file"> 拿不到完整路径，必须走 Dialogs API。）
async function onLoadFile() {
  const typed = externalPath.value.trim()
  if (typed) {
    await loadMidiPath(typed)
    return
  }
  const picked = await Dialogs.OpenFile({
    Title: '选择 MIDI 文件',
    Filters: [{ DisplayName: 'MIDI 文件 (*.mid *.midi)', Pattern: '*.mid;*.midi' }],
  })
  if (typeof picked !== 'string' || !picked) return
  externalPath.value = picked
  await loadMidiPath(picked)
}

async function loadMidiPath(p: string) {
  const r = await pianoStore.loadFile(p)
  if (r) {
    builtinPick.value = ''
    toast.add({ title: '已加载 MIDI 文件', color: 'success', icon: 'i-tabler-check' })
  }
}

// Config bindings (hydrated from settings)
const mode = ref<number>(settingsStore.data?.piano.mode ?? 0)
const trackPick = ref<number>(settingsStore.data?.piano.trackPick ?? -1)
const autoTranspose = ref<boolean>(settingsStore.data?.piano.autoTranspose ?? true)
const melodyOnly = ref<boolean>(settingsStore.data?.piano.melodyOnly ?? false)
const octaveOffset = ref<number>(settingsStore.data?.piano.octaveOffset ?? 0)

watch(
  () => settingsStore.data?.piano,
  (p) => {
    if (!p) return
    mode.value = p.mode
    trackPick.value = p.trackPick
    autoTranspose.value = p.autoTranspose
    melodyOnly.value = p.melodyOnly
    octaveOffset.value = p.octaveOffset
  },
  { deep: true },
)

const modeItems = [
  { label: '36 键 (含半音)', value: 0 },
  { label: '21 键 (仅自然音)', value: 1 },
]

const trackItems = computed(() => {
  const base: { label: string; value: number }[] = [
    { label: '智能选轨', value: -1 },
    { label: '全部 track', value: -2 },
  ]
  const tracks = pianoStore.currentMidi?.tracks ?? []
  for (const t of tracks) {
    base.push({ label: `Track ${t.index}: ${t.name} (${t.noteCount} 音符)`, value: t.index })
  }
  return base
})

async function onModeChange(v: number) {
  await pianoStore.setMode(v)
  if (settingsStore.data) settingsStore.data.piano.mode = v
}
async function onTrackPickChange(v: number) {
  await pianoStore.setTrackPick(v)
  if (settingsStore.data) settingsStore.data.piano.trackPick = v
}
async function onAutoTransposeChange(v: boolean) {
  await pianoStore.setAutoTranspose(v)
  if (settingsStore.data) settingsStore.data.piano.autoTranspose = v
}
async function onMelodyOnlyChange(v: boolean) {
  await pianoStore.setMelodyOnly(v)
  if (settingsStore.data) settingsStore.data.piano.melodyOnly = v
}
async function onOctaveOffsetChange(v: number) {
  await pianoStore.setOctaveOffset(v)
  if (settingsStore.data) settingsStore.data.piano.octaveOffset = v
}

// Seek slider: 0-1000 range maps to 0-totalMs
// Bidirectional: reads from progress, writes via seek
const seeking = ref(false)
const seekPct = ref<number>(0)

watch(
  () => pianoStore.progress,
  (p) => {
    if (seeking.value) return
    const pct = p.totalMs > 0 ? Math.floor((p.curMs * 1000) / p.totalMs) : 0
    seekPct.value = pct
  },
  { deep: true },
)

async function onSeekCommit(v: number) {
  seeking.value = true
  const total = pianoStore.progress.totalMs
  if (total > 0) {
    const targetMs = Math.floor((v * total) / 1000)
    await pianoStore.seek(targetMs)
  }
  // small delay to let progress event catch up before resuming sync
  setTimeout(() => {
    seeking.value = false
  }, 200)
}

const canStart = computed(
  () => pianoStore.state === 'idle' && !!gameStore.status?.ok && !!pianoStore.currentMidi,
)
const disabledReason = computed(() => {
  if (!gameStore.status?.ok) return '未检测到异环窗口'
  if (!pianoStore.currentMidi) return '请先加载 MIDI'
  if (pianoStore.state !== 'idle') return '已在演奏'
  return ''
})

function fmtMs(ms: number): string {
  if (ms < 0) ms = 0
  const totalSec = Math.floor(ms / 1000)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}:${String(s).padStart(2, '0')}`
}
</script>
