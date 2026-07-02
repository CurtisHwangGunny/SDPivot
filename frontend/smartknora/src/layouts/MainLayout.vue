<template>
  <t-layout class="main-layout">
    <t-aside class="sidebar" width="240px">
      <div class="logo">
        <svg viewBox="0 0 40 40" width="32" height="32" fill="none">
          <path d="M20 4L34 12V28L20 36L6 28V12L20 4Z" fill="#014DB2"/>
          <path d="M20 12L28 16V24L20 28L12 24V16L20 12Z" fill="#F59E0B"/>
        </svg>
        <span class="logo-text">随越·智枢</span>
      </div>
      <t-menu v-model="activeMenu" theme="light" @change="onMenuChange">
        <t-menu-item value="spaces">
          <template #icon><t-icon name="folder" /></template>
          知识空间
        </t-menu-item>
        <t-menu-item value="qa">
          <template #icon><t-icon name="chat" /></template>
          AI 问答
        </t-menu-item>
        <t-menu-item value="writing">
          <template #icon><t-icon name="edit" /></template>
          AI 写作
        </t-menu-item>
        <t-menu-item value="admin">
          <template #icon><t-icon name="setting-1" /></template>
          管理后台
        </t-menu-item>
        <t-menu-item value="ops">
          <template #icon><t-icon name="control-platform" /></template>
          运营管理
        </t-menu-item>
        <t-menu-item value="org">
          <template #icon><t-icon name="building" /></template>
          企业管理
        </t-menu-item>
        <t-menu-item value="usage">
          <template #icon><t-icon name="chart-bar" /></template>
          用量统计
        </t-menu-item>
        <t-menu-item value="settings">
          <template #icon><t-icon name="setting" /></template>
          个人设置
        </t-menu-item>
      </t-menu>
    </t-aside>
    <t-layout>
      <t-header class="header">
        <div class="header-right">
          <t-dropdown :options="userMenuOptions" @click="onUserMenuClick">
            <t-button variant="text">
              <t-icon name="user-circle" size="20px" />
              <span class="username">{{ authStore.user?.nickname || authStore.user?.username || '用户' }}</span>
              <t-icon name="chevron-down" size="14px" />
            </t-button>
          </t-dropdown>
        </div>
      </t-header>
      <t-content class="content">
        <router-view />
      </t-content>
    </t-layout>
  </t-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { logout } from '@/api/auth'
import { MessagePlugin } from 'tdesign-vue-next'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const activeMenu = computed(() => {
  const path = route.path
  if (path.startsWith('/spaces')) return 'spaces'
  if (path.startsWith('/qa')) return 'qa'
  if (path.startsWith('/writing')) return 'writing'
  if (path.startsWith('/admin')) return 'admin'
  if (path.startsWith('/ops')) return 'ops'
  if (path.startsWith('/org')) return 'org'
  if (path.startsWith('/usage')) return 'usage'
  if (path.startsWith('/settings')) return 'settings'
  return 'spaces'
})

const userMenuOptions = [
  { content: '个人设置', value: 'settings' },
  { content: '退出登录', value: 'logout', theme: 'error' as const },
]

function onMenuChange(val: string) { router.push(`/${val}`) }

async function onUserMenuClick(val: string) {
  if (val === 'logout') {
    try { await logout() } catch {}
    authStore.clearAuth()
    MessagePlugin.success('已退出登录')
    router.push('/login')
  } else if (val === 'settings') {
    router.push('/settings')
  }
}
</script>

<style scoped>
.main-layout { height: 100vh; }
.sidebar { background: #fff; border-right: 1px solid #e7e7e7; display: flex; flex-direction: column; }
.logo { display: flex; align-items: center; gap: 10px; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.logo-text { font-size: 18px; font-weight: 700; color: #014DB2; }
.header { background: #fff; border-bottom: 1px solid #e7e7e7; display: flex; align-items: center; justify-content: flex-end; padding: 0 24px; height: 56px; }
.header-right { display: flex; align-items: center; gap: 12px; }
.username { margin: 0 4px; font-size: 14px; }
.content { padding: 24px; overflow-y: auto; background: #f0f2f5; }
</style>
