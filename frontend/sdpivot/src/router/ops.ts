import { STORAGE_KEYS } from '../utils/storage'
import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/login',
      name: 'opsLogin',
      component: () => import('@/views/ops/OpsLoginPage.vue'),
    },
    {
      path: '/dashboard',
      name: 'opsDashboard',
      component: () => import('@/views/ops/OpsPage.vue'),
      meta: { requiresOpsAuth: true },
    },
    { path: '/:pathMatch(.*)*', redirect: '/login' },
  ],
})

router.beforeEach((to) => {
  if (to.meta.requiresOpsAuth) {
    const token = localStorage.getItem(STORAGE_KEYS.opsAccessToken)
    if (!token) return { name: 'opsLogin' }
  }
})

export default router
