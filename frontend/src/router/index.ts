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
  { path: '/tasks', name: 'tasks', component: () => import('@/views/TasksView.vue') },
  // /actions 老路由：redirect 到 /tasks（动作 tab）。原 ActionsView 内容已搬进 ActionsTab。
  { path: '/actions', redirect: '/tasks' },
  { path: '/schedules', name: 'schedules', component: () => import('@/views/SchedulesView.vue') },
  { path: '/settings', name: 'settings', component: () => import('@/views/SettingsView.vue') },
  { path: '/help', name: 'help', component: () => import('@/views/HelpView.vue') },
  { path: '/about', name: 'about', component: () => import('@/views/AboutView.vue') },
  // 独立窗口路由：wails3 子窗口加载 #/action-editor?id=xxx。
  // meta.standalone=true 让 App.vue 走极简渲染（不包 sidebar / titlebar / log panel）。
  {
    path: '/action-editor',
    name: 'action-editor',
    component: () => import('@/views/ActionEditorView.vue'),
    meta: { standalone: true },
  },
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
