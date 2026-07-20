<template>
  <div class="settings-shell">
    <section class="settings-hero">
      <div>
        <p class="settings-eyebrow">Workspace settings</p>
        <h1>个人设置</h1>
        <p class="settings-subtitle">统一管理账号信息、安全设置与个性化偏好。当前已接入主题切换骨架，并为后续通知、快捷偏好与安全能力继续预留承载位。</p>
      </div>
      <div class="settings-hero-card">
        <span>当前租户</span>
        <strong>{{ authStore.user?.tenant_name || `租户 #${authStore.user?.tenant_id || '--'}` }}</strong>
        <small>账号信息已与当前登录态保持同步</small>
      </div>
    </section>

    <section class="settings-workspace">
      <aside class="settings-sidebar">
        <button
          v-for="item in sections"
          :key="item.key"
          type="button"
          class="settings-nav-item"
          :class="{ active: activeSection === item.key }"
          @click="activeSection = item.key"
        >
          <div>
            <strong>{{ item.label }}</strong>
            <span>{{ item.description }}</span>
          </div>
        </button>
      </aside>

      <div class="settings-content">
        <Suspense>
          <component
            :is="activePanelComponent"
            :user="authStore.user"
            :theme-mode="themeMode"
            :current-theme-label="currentThemeLabel"
            :set-theme="setTheme"
          />
          <template #fallback>
            <div class="settings-panel panel-loading">
              <t-loading size="small" text="正在加载设置面板..." />
            </div>
          </template>
        </Suspense>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'

const authStore = useAuthStore()
const activeSection = ref('profile')
const { themeMode, currentThemeLabel, setTheme } = useTheme()

const sections = [
  { key: 'profile', label: '账户资料', description: '查看当前账号与租户信息' },
  { key: 'appearance', label: '外观偏好', description: '主题、密度与阅读体验设置' },
  { key: 'notifications', label: '通知提醒', description: '系统消息与工作流通知规划' },
]

const ProfilePanel = defineAsyncComponent(() => import('@/components/settings/SettingsProfilePanel.vue'))
const AppearancePanel = defineAsyncComponent(() => import('@/components/settings/SettingsAppearancePanel.vue'))
const NotificationsPanel = defineAsyncComponent(() => import('@/components/settings/SettingsNotificationsPanel.vue'))

const activePanelComponent = computed(() => {
  if (activeSection.value === 'appearance') return AppearancePanel
  if (activeSection.value === 'notifications') return NotificationsPanel
  return ProfilePanel
})
</script>

<style scoped>
.settings-shell {
  display: grid;
  gap: 18px;
}
.settings-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.8fr) minmax(280px, 0.9fr);
  gap: 20px;
  align-items: stretch;
}
.settings-eyebrow,
.panel-kicker {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--brand-primary);
}
.settings-hero h1,
.panel-head h2 {
  margin: 0;
  font-size: 28px;
  line-height: 1.1;
  color: var(--text-primary);
}
.settings-subtitle {
  max-width: 720px;
  margin: 14px 0 0;
  color: var(--text-secondary);
  line-height: 1.7;
}
.settings-hero-card,
.settings-panel,
.settings-sidebar {
  background: var(--surface-elevated);
  border: 1px solid var(--border-soft);
  border-radius: 16px;
  box-shadow: var(--shadow-soft);
}
.settings-hero-card {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  padding: 22px;
  background: linear-gradient(135deg, rgba(0, 185, 107, 0.12), rgba(31, 41, 55, 0.04));
}
.settings-hero-card span {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.settings-hero-card strong {
  font-size: 22px;
  color: var(--text-primary);
}
.settings-hero-card small {
  color: var(--text-secondary);
  line-height: 1.6;
}
.settings-workspace {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}
.settings-sidebar {
  padding: 12px;
  position: sticky;
  top: 20px;
}
.settings-nav-item {
  width: 100%;
  display: block;
  text-align: left;
  border: 0;
  background: transparent;
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: transform 0.25s ease, background 0.25s ease, box-shadow 0.25s ease;
}
.settings-nav-item strong {
  display: block;
  color: var(--text-primary);
  font-size: 15px;
}
.settings-nav-item span {
  display: block;
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-secondary);
}
.settings-nav-item:hover,
.settings-nav-item.active {
  background: rgba(0, 185, 107, 0.12);
  box-shadow: inset 0 0 0 1px rgba(0, 185, 107, 0.18);
  transform: translateY(-1px);
}
.settings-content {
  display: grid;
  gap: 20px;
}
.panel-loading {
  min-height: 280px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
}
@media (max-width: 1080px) {
  .settings-hero,
  .settings-workspace {
    grid-template-columns: 1fr;
  }
  .settings-sidebar {
    position: static;
  }
}
</style>
