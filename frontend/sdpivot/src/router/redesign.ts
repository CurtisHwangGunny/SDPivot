import type { RouteRecordRaw } from 'vue-router'

const redesignRoutes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: () => import('@/views/design/LoginPage.vue'), meta: { requiresAuth: false } },
  { path: '/spaces', name: 'spaces', component: () => import('@/views/design/SpaceHomePage.vue') },
  { path: '/qa', name: 'qa', component: () => import('@/views/design/QAWorkspacePage.vue') },
  { path: '/writing', name: 'writing', component: () => import('@/views/design/WritingWorkspacePage.vue') },
  { path: '/admin', name: 'admin', component: () => import('@/views/design/AdminDashboardPage.vue') },
  { path: '/admin/people', name: 'adminPeople', component: () => import('@/views/design/PeopleListPage.vue') },
  { path: '/admin/tags', name: 'adminTags', component: () => import('@/views/design/TagDictionaryPage.vue') },
]

export default redesignRoutes
