import { createRouter, createWebHashHistory } from 'vue-router'
import { useGameStore } from '@/stores/game'

const routes = [
  { path: '/', redirect: '/containers' },
  { path: '/containers', name: 'containers', component: () => import('@/views/ContainersView.vue') },
  {
    path: '/containers/:id/edit',
    name: 'container-edit',
    component: () => import('@/views/ContainerEditorView.vue'),
  },
  { path: '/library', name: 'library', component: () => import('@/views/LibraryView.vue') },
  { path: '/schedules', name: 'schedules', component: () => import('@/views/SchedulesView.vue') },
  { path: '/settings', name: 'settings', component: () => import('@/views/SettingsView.vue') },
  { path: '/help', name: 'help', component: () => import('@/views/HelpView.vue') },
  { path: '/about', name: 'about', component: () => import('@/views/AboutView.vue') },
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

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// 进编辑器路由时触发游戏检测 (容器编辑才需要 game.status; 列表页不需要).
router.beforeEach((to) => {
  if (to.name === 'container-edit') {
    useGameStore().detect()
  }
})
