import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/views/auth/LoginPage.vue'), meta: { requiresAuth: false } },
    { path: '/register', name: 'register', component: () => import('@/views/auth/RegisterPage.vue'), meta: { requiresAuth: false } },
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
        { path: 'ops', name: 'ops', component: () => import('@/views/ops/OpsPage.vue') },
        { path: 'org', name: 'org', component: () => import('@/views/org/OrgPage.vue') },
        { path: 'usage', name: 'usage', component: () => import('@/views/qa/UsagePage.vue') },
        { path: 'settings', name: 'settings', component: () => import('@/views/settings/SettingsPage.vue') },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('access_token')
  if (to.meta.requiresAuth !== false && !token) return { name: 'login' }
})

export default router
