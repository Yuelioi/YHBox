import { createRouter, createWebHashHistory } from 'vue-router'

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
  {
    path: '/tools/calibration-hud',
    name: 'calibration-hud',
    component: () => import('@/views/tools/CalibrationHUDView.vue'),
    meta: { standalone: true },
  },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})
