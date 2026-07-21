<template>
  <div class="main-layout">
    <aside class="sidebar-shell">
      <div class="sidebar-top">
        <div class="brand-block">
          <div class="brand-mark">
            <svg viewBox="0 0 40 40" width="28" height="28" fill="none">
              <path d="M20 4L34 12V28L20 36L6 28V12L20 4Z" fill="#00B96B"/>
              <path d="M20 12L28 16V24L20 28L12 24V16L20 12Z" fill="#EAFBF2"/>
            </svg>
          </div>
          <div>
            <div class="brand-title">SDPivot</div>
            <div class="brand-subtitle">SDP Workspace</div>
          </div>
        </div>

        <div class="theme-switcher-card">
          <div>
            <div class="theme-switcher-title">界面主题</div>
            <div class="theme-switcher-subtitle">{{ currentThemeLabel }}</div>
          </div>
          <div class="theme-actions">
            <button type="button" class="theme-chip" :class="{ active: themeMode === 'light' }" @click="setTheme('light')">浅色</button>
            <button type="button" class="theme-chip" :class="{ active: themeMode === 'dark' }" @click="setTheme('dark')">深色</button>
            <button type="button" class="theme-chip" :class="{ active: themeMode === 'system' }" @click="setTheme('system')">系统</button>
          </div>
        </div>

        <div class="sidebar-caption">知识工作台</div>
        <t-menu v-model="activeMenu" theme="dark" class="main-menu" @change="onMenuChange">
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
      </div>

      <div class="sidebar-bottom">
        <t-dropdown :options="userMenuOptions" trigger="click" @click="onUserMenuClick">
          <button class="user-panel" type="button">
            <div class="user-avatar">
              {{ userInitial }}
            </div>
            <div class="user-meta">
              <span class="user-name">{{ authStore.user?.nickname || authStore.user?.username || '当前用户' }}</span>
              <span class="user-role">{{ isOpsRoute ? '运营身份' : '企业身份' }}</span>
            </div>
            <t-icon name="chevron-up-down" size="16px" />
          </button>
        </t-dropdown>
      </div>
    </aside>

    <main class="content-shell">
      <div class="content-inner page-shell">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { logout } from '@/api/auth'
import { MessagePlugin } from 'tdesign-vue-next'
import { useTheme } from '@/composables/useTheme'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { themeMode, currentThemeLabel, setTheme } = useTheme()

const isOpsRoute = computed(() => route.path.startsWith('/ops'))

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

const userInitial = computed(() => {
  const name = authStore.user?.nickname || authStore.user?.username || 'U'
  return String(name).trim().charAt(0).toUpperCase() || 'U'
})

const userMenuOptions = computed(() => [
  { content: isOpsRoute.value ? '运营设置' : '个人设置', value: 'settings' },
  { content: isOpsRoute.value ? '退出运营登录' : '退出登录', value: 'logout', theme: 'error' as const },
])

function onMenuChange(val: string) {
  router.push(`/${val}`)
}

async function onUserMenuClick(val: string) {
  if (val === 'logout') {
    if (isOpsRoute.value) {
      authStore.clearOpsAuth()
      MessagePlugin.success('已退出运营登录')
      router.push('/ops-login')
      return
    }
    try {
      await logout()
    } catch {}
    authStore.clearAuth()
    MessagePlugin.success('已退出登录')
    router.push('/login')
  } else if (val === 'settings') {
    router.push(isOpsRoute.value ? '/ops' : '/settings')
  }
}
</script>

<style scoped>
.main-layout {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr);
  background: transparent;
}

.sidebar-shell {
  position: sticky;
  top: 0;
  height: 100dvh;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 20px 14px 16px;
  background: linear-gradient(180deg, var(--sdp-sidebar) 0%, var(--sdp-sidebar-strong) 100%);
  border-right: 1px solid var(--sidebar-border);
}

.sidebar-top {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.brand-block {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 8px 12px;
}

.brand-mark {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.08);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.12);
}

.brand-title {
  font-size: 17px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.95);
  letter-spacing: 0.01em;
}

.brand-subtitle {
  margin-top: 2px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.58);
}

.theme-switcher-card {
  padding: 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.theme-switcher-title {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.92);
}

.theme-switcher-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.58);
}

.theme-actions {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-top: 12px;
}

.theme-chip {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.76);
  padding: 8px 10px;
  cursor: pointer;
  transition: transform 0.24s ease, background 0.24s ease, color 0.24s ease, border-color 0.24s ease;
}

.theme-chip:hover,
.theme-chip.active {
  background: rgba(0, 185, 107, 0.18);
  color: #ffffff;
  transform: translateY(-1px);
}

.sidebar-caption {
  padding: 0 12px;
  font-size: 12px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.48);
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.main-menu {
  border: none;
  background: transparent;
}

.main-menu :deep(.t-menu__item) {
  margin-bottom: 6px;
  border-radius: 12px;
}

.main-menu :deep(.t-menu__item.t-is-active) {
  background: rgba(0, 185, 107, 0.16);
  color: #ffffff;
}

.main-menu :deep(.t-menu__item:hover) {
  background: rgba(255, 255, 255, 0.08);
}

.sidebar-bottom {
  padding-top: 16px;
}

.user-panel {
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.04);
  border-radius: 12px;
  padding: 12px;
  display: flex;
  align-items: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.92);
  cursor: pointer;
  transition: transform 0.24s cubic-bezier(0.16, 1, 0.3, 1), background 0.24s ease;
}

.user-panel:hover {
  transform: translateY(-1px);
  background: rgba(255, 255, 255, 0.08);
}

.user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
  color: #052814;
  background: linear-gradient(135deg, #7ef0b6 0%, #00b96b 100%);
}

.user-meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  text-align: left;
}

.user-name {
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-role {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.56);
}

.content-shell {
  min-width: 0;
  padding: 32px;
}

.content-inner {
  min-height: calc(100dvh - 64px);
}

@media (max-width: 960px) {
  .main-layout {
    grid-template-columns: 1fr;
  }

  .sidebar-shell {
    position: relative;
    height: auto;
    padding-bottom: 16px;
  }

  .content-shell {
    padding: 20px;
  }
}
</style>
