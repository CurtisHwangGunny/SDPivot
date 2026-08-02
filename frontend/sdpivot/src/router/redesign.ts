import type { RouteRecordRaw } from 'vue-router'

const isOpBuild = import.meta.env.MODE === 'op'

const redesignRoutes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: () => import('@/views/design/LoginPage.vue'), meta: { requiresAuth: false } },
  { path: '/register', redirect: '/login' },
  ...(!isOpBuild ? [{ path: '/org', component: () => import('@/views/design/OrgManagementPage.vue') }] : []),
  { path: '/settings', component: () => import('@/views/design/PersonalSettingsPage.vue') },
  { path: '/spaces', name: 'spaces', component: () => import('@/views/design/SpaceHomePage.vue') },
  { path: '/spaces/:id', name: 'spaceDetail', component: () => import('@/views/design/SpaceDetailPage.vue') },
  { path: '/spaces/:id/documents', name: 'spaceDocuments', component: () => import('@/views/design/DocumentsListPage.vue') },
  { path: '/qa', name: 'qa', component: () => import('@/views/design/QAWorkspacePage.vue') },
  { path: '/qa/usage', component: () => import('@/views/design/UsageStatisticsPage.vue') },
  { path: '/writing', name: 'writing', component: () => import('@/views/design/WritingWorkspacePage.vue') },
  { path: '/admin', name: 'admin', component: () => import('@/views/design/AdminDashboardPage.vue') },
  { path: '/admin/people', name: 'adminPeople', component: () => import('@/views/design/PeopleListPage.vue') },
  { path: '/admin/tags', name: 'adminTags', component: () => import('@/views/design/TagDictionaryPage.vue') },
]

export default redesignRoutes
