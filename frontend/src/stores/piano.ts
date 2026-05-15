import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type PianoProgressEvent } from '@/lib/backend'

interface LibraryEntry {
  asset: string
  display: string
}
interface TrackInfo {
  index: number
  name: string
  noteCount: number
}
interface MIDIInfo {
  path: string
  totalMs: number
  tracks: TrackInfo[]
}

export const usePianoStore = defineStore('piano', () => {
  const state = ref<'idle' | 'running' | 'paused'>('idle')
  const progress = ref<PianoProgressEvent>({ curMs: 0, totalMs: 0 })
  const library = ref<LibraryEntry[]>([])
  const currentMidi = ref<MIDIInfo | null>(null)

  function setState(s: 'idle' | 'running' | 'paused') {
    state.value = s
  }
  function setProgress(p: PianoProgressEvent) {
    progress.value = p
  }
  function setCurrentMidi(m: MIDIInfo | null) {
    currentMidi.value = m
    progress.value = { curMs: 0, totalMs: m?.totalMs ?? 0 }
  }

  async function loadLibrary() {
    const r = await backend.piano.getLibrary()
    if (r) library.value = r as LibraryEntry[]
  }
  async function loadBuiltin(asset: string) {
    const r = await backend.piano.loadBuiltin(asset)
    if (r) setCurrentMidi(r as MIDIInfo)
    return r
  }
  async function loadFile(path: string) {
    const r = await backend.piano.loadFile(path)
    if (r) setCurrentMidi(r as MIDIInfo)
    return r
  }

  async function start() {
    return backend.piano.start()
  }
  async function pause() {
    return backend.piano.pause()
  }
  async function resume() {
    return backend.piano.resume()
  }
  async function stop() {
    return backend.piano.stop()
  }
  async function seek(ms: number) {
    return backend.piano.seek(ms)
  }
  async function setMode(m: number) {
    return backend.piano.setMode(m)
  }
  async function setTrackPick(p: number) {
    return backend.piano.setTrackPick(p)
  }
  async function setAutoTranspose(v: boolean) {
    return backend.piano.setAutoTranspose(v)
  }
  async function setMelodyOnly(v: boolean) {
    return backend.piano.setMelodyOnly(v)
  }
  async function setOctaveOffset(v: number) {
    return backend.piano.setOctaveOffset(v)
  }

  return {
    state,
    progress,
    library,
    currentMidi,
    setState,
    setProgress,
    setCurrentMidi,
    loadLibrary,
    loadBuiltin,
    loadFile,
    start,
    pause,
    resume,
    stop,
    seek,
    setMode,
    setTrackPick,
    setAutoTranspose,
    setMelodyOnly,
    setOctaveOffset,
  }
})
