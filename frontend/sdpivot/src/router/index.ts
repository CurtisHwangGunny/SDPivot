import { STORAGE_KEYS } from '../utils/storage'
import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/views/auth/LoginPage.vue'), meta: { requiresAuth: false } },
    { path: '/register', name: 'register', component: () => import('@/views/auth/RegisterPage.vue'), meta: { requiresAuth: false } },
    { path: '/ops-login', name: 'opsLogin', component: () => import('@/views/ops/OpsLoginPage.vue'), meta: { requiresAuth: false } },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      redirect: '/spaces',
      children: [
        { path: 'spaces', name: 'spaces', component: () => import('@/views/spaces/SpacesPage.vue') },
        { path: 'spaces/:id', name: 'spaceDetail', component: () => import('@/views/spaces/SpaceDetailPage.vue') },
        { path: 'spaces/:id/documents', name: 'documents', component: () => import('@/views/spaces/DocumentsPage.vue') },
        { path: 'qa', name: 'qa', component: () => import('@/views/qa/QAPage.vue') },
        { path: 'writing', name: 'writing', component: () => import('@/views/writing/WritingPage.vue') },
        { path: 'admin', name: 'admin', component: () => import('@/views/admin/AdminPage.vue') },
        { path: 'ops', name: 'ops', component: () => import('@/views/ops/OpsPage.vue'), meta: { requiresOpsAuth: true } },
        { path: 'org', name: 'org', component: () => import('@/views/org/OrgPage.vue') },
        { path: 'usage', name: 'usage', component: () => import('@/views/qa/UsagePage.vue') },
        { path: 'settings', name: 'settings', component: () => import('@/views/settings/SettingsPage.vue') },
      ],
    },
  ],
})

router.beforeEach((to) => {
  if (to.meta.requiresOpsAuth) {
    const opsToken = localStorage.getItem(STORAGE_KEYS.opsAccessToken)
    if (!opsToken) return { name: 'opsLogin' }
    return true
  }

  const token = localStorage.getItem(STORAGE_KEYS.accessToken)
  if (to.meta.requiresAuth !== false && !token) return { name: 'login' }
  return true
})

export default router
