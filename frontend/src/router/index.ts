import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/workflows' },
  {
    path: '/workflows',
    name: 'workflows',
    component: () => import('@/views/WorkflowsView.vue'),
  },
  {
    path: '/workflows/:id/edit',
    name: 'workflow-edit',
    component: () => import('@/views/WorkflowEditorView.vue'),
  },
  { path: '/schedules', name: 'schedules', component: () => import('@/views/SchedulesView.vue') },
  { path: '/settings', name: 'settings', component: () => import('@/views/SettingsView.vue') },
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
  {
    path: '/tools/launcher',
    name: 'launcher',
    component: () => import('@/views/tools/FloatingLauncherView.vue'),
    meta: { standalone: true },
  },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})
