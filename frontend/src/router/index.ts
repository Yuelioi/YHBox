import { createRouter, createWebHashHistory } from 'vue-router'
import { useGameStore } from '@/stores/game'
import { BOTS } from '@/lib/bot-registry'

// routes 仍然写死（vite dynamic import 不支持模板字符串路径）。
// 但 BOT_ROUTES 由 registry 派生，新增长跑 bot 只改 bot-registry.ts。
const routes = [
  { path: '/', name: 'fish', component: () => import('@/views/FishView.vue') },
  { path: '/cook', name: 'cook', component: () => import('@/views/CookView.vue') },
  { path: '/piano', name: 'piano', component: () => import('@/views/PianoView.vue') },
  { path: '/battle', name: 'battle', component: () => import('@/views/BattleView.vue') },
  { path: '/rhythm', name: 'rhythm', component: () => import('@/views/RhythmView.vue') },
  { path: '/containers', name: 'containers', component: () => import('@/views/ContainersView.vue') },
  { path: '/library', name: 'library', component: () => import('@/views/LibraryView.vue') },
  { path: '/schedules', name: 'schedules', component: () => import('@/views/SchedulesView.vue') },
  { path: '/settings', name: 'settings', component: () => import('@/views/SettingsView.vue') },
  { path: '/help', name: 'help', component: () => import('@/views/HelpView.vue') },
  { path: '/about', name: 'about', component: () => import('@/views/AboutView.vue') },
  // v2 Task 3.1: 删除 /actions 和 /action-editor 路由（actions 包已废，路由也清掉）
  {
    path: '/container-editor',
    name: 'container-editor',
    component: () => import('@/views/ContainerEditorView.vue'),
    meta: { standalone: true },
  },
  {
    path: '/tools/mouse-hud',
    name: 'mouse-hud',
    component: () => import('@/views/tools/MouseHUDView.vue'),
    meta: { standalone: true },
  },
  {
    path: '/tools/screen-picker',
    name: 'screen-picker',
    component: () => import('@/views/tools/ScreenPickerView.vue'),
    meta: { standalone: true },
  },
  {
    path: '/tools/recording-hud',
    name: 'recording-hud',
    component: () => import('@/views/tools/RecordingHUDView.vue'),
    meta: { standalone: true },
  },
]

// 切到 bot route 时自动触发游戏检测。长跑 bot 从 BOTS 派生 + battle / tasks 手动 append。
const BOT_ROUTES = new Set<string>([...BOTS.map((b) => b.kind), 'battle', 'tasks'])

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to) => {
  if (BOT_ROUTES.has(to.name as string)) {
    // 异步触发，不阻塞导航
    useGameStore().detect()
  }
})

export function isBotRoute(name: string | symbol | null | undefined): boolean {
  return BOT_ROUTES.has(name as string)
}
