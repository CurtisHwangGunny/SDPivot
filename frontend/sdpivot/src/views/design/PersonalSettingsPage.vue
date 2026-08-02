<template>
  <main class="sdp-personal-settings">
    <div class="sdp-personal-settings__shell">
      <header class="sdp-personal-settings__header">
        <div>
          <p class="sdp-personal-settings__eyebrow">User / Settings</p>
          <h1>个人设置</h1>
          <p class="sdp-personal-settings__subtitle">管理个人资料、界面外观、通知偏好与账号安全。</p>
        </div>
        <SdpButton
          v-if="currentTab === 'profile'"
          variant="secondary"
          :disabled="loading || savingProfile"
          aria-label="重新加载个人资料"
          @click="loadProfile"
        >
          重置修改
        </SdpButton>
      </header>

      <SdpNotice v-if="statusMessage" :type="statusType" :title="statusMessage" />

      <section class="sdp-personal-settings__layout">
        <nav class="sdp-personal-settings__tabs" aria-label="个人设置分类">
          <button
            v-for="tab in tabs"
            :id="`settings-tab-${tab.key}`"
            :key="tab.key"
            type="button"
            role="tab"
            :aria-controls="`settings-panel-${tab.key}`"
            :aria-selected="currentTab === tab.key"
            :tabindex="currentTab === tab.key ? 0 : -1"
            class="sdp-personal-settings__tab"
            :class="{ 'sdp-personal-settings__tab--active': currentTab === tab.key }"
            @click="selectTab(tab.key)"
            @keydown="handleTabKeydown"
          >
            <strong>{{ tab.label }}</strong>
            <span>{{ tab.description }}</span>
          </button>
        </nav>

        <section
          :id="`settings-panel-${currentTab}`"
          class="sdp-personal-settings__panel"
          role="tabpanel"
          :aria-labelledby="`settings-tab-${currentTab}`"
          tabindex="0"
        >
          <div v-if="currentTab === 'profile'" class="sdp-personal-settings__stack">
            <form class="sdp-personal-settings__section" @submit.prevent="saveProfile">
              <div class="sdp-personal-settings__section-head">
                <div>
                  <p class="sdp-personal-settings__kicker">Profile</p>
                  <h2>个人资料</h2>
                </div>
                <span class="sdp-personal-settings__badge">{{ loading ? '同步中' : '已同步登录账号' }}</span>
              </div>

              <div class="sdp-personal-settings__form-grid" :aria-busy="loading">
                <label class="sdp-personal-settings__field">
                  <span>显示名称</span>
                  <input v-model.trim="profile.name" name="name" autocomplete="name" :disabled="loading" required />
                </label>
                <label class="sdp-personal-settings__field">
                  <span>邮箱</span>
                  <input v-model.trim="profile.email" name="email" type="email" autocomplete="email" :disabled="loading" required />
                </label>
                <label class="sdp-personal-settings__field sdp-personal-settings__field--wide">
                  <span>手机号</span>
                  <input v-model.trim="profile.phone" name="phone" type="tel" autocomplete="tel" :disabled="loading" />
                </label>
              </div>

              <div class="sdp-personal-settings__actions">
                <SdpButton :loading="savingProfile" :disabled="loading" aria-label="保存个人资料" @click="saveProfile">
                  保存资料
                </SdpButton>
              </div>
            </form>

            <form class="sdp-personal-settings__section" @submit.prevent="changePassword">
              <div class="sdp-personal-settings__section-head">
                <div>
                  <p class="sdp-personal-settings__kicker">Security</p>
                  <h2>修改密码</h2>
                </div>
              </div>

              <div class="sdp-personal-settings__form-grid">
                <label class="sdp-personal-settings__field sdp-personal-settings__field--wide">
                  <span>当前密码</span>
                  <input v-model="password.oldPassword" name="current-password" type="password" autocomplete="current-password" required />
                </label>
                <label class="sdp-personal-settings__field">
                  <span>新密码</span>
                  <input v-model="password.newPassword" name="new-password" type="password" autocomplete="new-password" minlength="8" required />
                </label>
                <label class="sdp-personal-settings__field">
                  <span>确认新密码</span>
                  <input v-model="password.confirmPassword" name="confirm-password" type="password" autocomplete="new-password" minlength="8" required />
                </label>
              </div>

              <p class="sdp-personal-settings__hint">新密码至少需要 8 个字符。</p>
              <div class="sdp-personal-settings__actions">
                <SdpButton :loading="changingPassword" aria-label="提交密码修改" @click="changePassword">更新密码</SdpButton>
              </div>
            </form>
          </div>

          <div v-else-if="currentTab === 'appearance'" class="sdp-personal-settings__section">
            <div class="sdp-personal-settings__section-head">
              <div>
                <p class="sdp-personal-settings__kicker">Appearance</p>
                <h2>外观主题</h2>
                <p>当前设置：{{ currentThemeLabel }}</p>
              </div>
            </div>

            <fieldset class="sdp-personal-settings__choice-grid">
              <legend class="sr-only">选择界面主题</legend>
              <label v-for="option in themeOptions" :key="option.value" class="sdp-personal-settings__choice">
                <input
                  :checked="themeMode === option.value"
                  type="radio"
                  name="theme"
                  :value="option.value"
                  @change="setTheme(option.value)"
                />
                <span class="sdp-personal-settings__choice-copy">
                  <strong>{{ option.label }}</strong>
                  <small>{{ option.description }}</small>
                </span>
              </label>
            </fieldset>
          </div>

          <form v-else class="sdp-personal-settings__section" @submit.prevent="saveNotifications">
            <div class="sdp-personal-settings__section-head">
              <div>
                <p class="sdp-personal-settings__kicker">Notifications</p>
                <h2>通知偏好</h2>
                <p>选择需要在当前设备接收的工作提醒。</p>
              </div>
            </div>

            <fieldset class="sdp-personal-settings__notification-list">
              <legend class="sr-only">通知选项</legend>
              <label v-for="option in notificationOptions" :key="option.key" class="sdp-personal-settings__notification">
                <span>
                  <strong>{{ option.label }}</strong>
                  <small>{{ option.description }}</small>
                </span>
                <input v-model="notifications[option.key]" type="checkbox" :name="option.key" />
              </label>
            </fieldset>

            <div class="sdp-personal-settings__actions">
              <SdpButton :loading="savingNotifications" aria-label="保存通知偏好" @click="saveNotifications">保存通知偏好</SdpButton>
            </div>
          </form>
        </section>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import client from '@/api/client'
import { SdpButton, SdpNotice } from '@/components/design'
import { type ThemeMode, useTheme } from '@/composables/useTheme'
import { useAuthStore } from '@/stores/auth'

type TabKey = 'profile' | 'appearance' | 'notifications'
type NoticeType = 'success' | 'info' | 'warning' | 'danger'
type UserProfile = {
  id?: string
  username?: string
  nickname?: string
  name?: string
  email?: string
  phone?: string
  mobile?: string
}
type NotificationKey = 'system' | 'tasks' | 'security'

const NOTIFICATION_STORAGE_KEY = 'sdp_personal_notification_settings'
const tabs: Array<{ key: TabKey; label: string; description: string }> = [
  { key: 'profile', label: '个人资料', description: '账号信息与密码安全' },
  { key: 'appearance', label: '外观主题', description: '浅色、深色或跟随系统' },
  { key: 'notifications', label: '通知偏好', description: '任务与安全提醒' },
]
const themeOptions: Array<{ value: ThemeMode; label: string; description: string }> = [
  { value: 'light', label: '浅色模式', description: '适合明亮环境下使用' },
  { value: 'dark', label: '深色模式', description: '降低暗光环境中的视觉压力' },
  { value: 'system', label: '跟随系统', description: '自动匹配设备外观设置' },
]
const notificationOptions: Array<{ key: NotificationKey; label: string; description: string }> = [
  { key: 'system', label: '系统通知', description: '接收服务状态和重要产品消息' },
  { key: 'tasks', label: '任务提醒', description: '接收文档导入与生成任务结果' },
  { key: 'security', label: '安全提醒', description: '接收密码与账号安全相关消息' },
]

const authStore = useAuthStore()
const { themeMode, currentThemeLabel, setTheme } = useTheme()
const currentTab = ref<TabKey>('profile')
const loading = ref(false)
const savingProfile = ref(false)
const changingPassword = ref(false)
const savingNotifications = ref(false)
const statusMessage = ref('')
const statusType = ref<NoticeType>('info')
const profile = reactive({ name: '', email: '', phone: '' })
const password = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const notifications = reactive<Record<NotificationKey, boolean>>({ system: true, tasks: true, security: true })

function showStatus(message: string, type: NoticeType) {
  statusMessage.value = message
  statusType.value = type
}

function errorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || error === null) return fallback
  const requestError = error as { message?: string; response?: { data?: { error?: string; message?: string } } }
  return requestError.response?.data?.error || requestError.response?.data?.message || requestError.message || fallback
}

function applyUser(user: UserProfile) {
  profile.name = user.nickname || user.name || user.username || ''
  profile.email = user.email || ''
  profile.phone = user.phone || user.mobile || ''
}

function syncAuthUser(user: UserProfile) {
  const nextUser = { ...(authStore.user || {}), ...user }
  authStore.setAuth({
    access_token: authStore.token,
    refresh_token: authStore.refreshToken,
    user: nextUser,
  })
}

function loadProfile() {
  loading.value = true
  statusMessage.value = ''
  if (authStore.user) applyUser(authStore.user)
  loading.value = false
}

function saveProfile() {
  if (!profile.name || !profile.email) {
    showStatus('请填写显示名称和邮箱。', 'warning')
    return
  }
  savingProfile.value = true
  statusMessage.value = ''
  const user = { ...profile, nickname: profile.name }
  applyUser(user)
  syncAuthUser(user)
  savingProfile.value = false
  showStatus('个人资料已保存。', 'success')
}

async function changePassword() {
  if (!password.oldPassword || password.newPassword.length < 8) {
    showStatus('请填写当前密码，并输入至少 8 个字符的新密码。', 'warning')
    return
  }
  if (password.newPassword !== password.confirmPassword) {
    showStatus('两次输入的新密码不一致。', 'warning')
    return
  }
  changingPassword.value = true
  statusMessage.value = ''
  try {
    await client.put('/password', {
      old_password: password.oldPassword,
      new_password: password.newPassword,
    })
    password.oldPassword = ''
    password.newPassword = ''
    password.confirmPassword = ''
    showStatus('密码已更新。', 'success')
  } catch (error: unknown) {
    showStatus(errorMessage(error, '密码更新失败，请检查当前密码。'), 'danger')
  } finally {
    changingPassword.value = false
  }
}

function loadNotifications() {
  try {
    const saved = window.localStorage.getItem(NOTIFICATION_STORAGE_KEY)
    if (saved) Object.assign(notifications, JSON.parse(saved))
  } catch {
    window.localStorage.removeItem(NOTIFICATION_STORAGE_KEY)
  }
}

function saveNotifications() {
  savingNotifications.value = true
  window.localStorage.setItem(NOTIFICATION_STORAGE_KEY, JSON.stringify(notifications))
  savingNotifications.value = false
  showStatus('通知偏好已保存到当前设备。', 'success')
}

function selectTab(tab: TabKey) {
  currentTab.value = tab
  statusMessage.value = ''
}

function handleTabKeydown(event: KeyboardEvent) {
  if (!['ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return
  const currentIndex = tabs.findIndex((tab) => tab.key === currentTab.value)
  let nextIndex = currentIndex
  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = tabs.length - 1
  if (event.key === 'ArrowDown') nextIndex = (currentIndex + 1) % tabs.length
  if (event.key === 'ArrowUp') nextIndex = (currentIndex - 1 + tabs.length) % tabs.length
  event.preventDefault()
  currentTab.value = tabs[nextIndex].key
  document.getElementById(`settings-tab-${currentTab.value}`)?.focus()
}

onMounted(() => {
  loadNotifications()
  loadProfile()
})
</script>

<style scoped>
.sdp-personal-settings {
  min-height: 100dvh;
  color: var(--ink-900);
  background: var(--ink-100);
  font-family: var(--font-body);
}

.sdp-personal-settings__shell {
  display: grid;
  gap: var(--space-6);
  width: min(100%, var(--content-max-width));
  margin-inline: auto;
  padding: var(--space-8);
}

.sdp-personal-settings__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-6);
  padding: var(--space-8);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-xl);
  background: var(--ink-50);
  box-shadow: var(--shadow-sm);
}

.sdp-personal-settings__eyebrow,
.sdp-personal-settings__kicker {
  margin: 0 0 var(--space-2);
  color: var(--brand-700);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-bold);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sdp-personal-settings h1,
.sdp-personal-settings h2 {
  margin: 0;
  color: var(--ink-900);
  font-family: var(--font-display);
  line-height: var(--leading-tight);
}

.sdp-personal-settings h1 {
  font-size: var(--text-3xl);
}

.sdp-personal-settings h2 {
  font-size: var(--text-xl);
}

.sdp-personal-settings__subtitle,
.sdp-personal-settings__section-head p:not(.sdp-personal-settings__kicker) {
  margin: var(--space-2) 0 0;
  color: var(--ink-600);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}

.sdp-personal-settings__layout {
  display: grid;
  grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
  gap: var(--space-6);
  align-items: start;
}

.sdp-personal-settings__tabs,
.sdp-personal-settings__panel,
.sdp-personal-settings__section {
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-lg);
  background: var(--ink-50);
  box-shadow: var(--shadow-sm);
}

.sdp-personal-settings__tabs {
  display: grid;
  gap: var(--space-2);
  padding: var(--space-3);
}

.sdp-personal-settings__tab {
  display: grid;
  gap: var(--space-1);
  width: 100%;
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--ink-50);
  border-radius: var(--radius-sm);
  color: var(--ink-700);
  background: var(--ink-50);
  font-family: var(--font-body);
  text-align: left;
  cursor: pointer;
}

.sdp-personal-settings__tab:hover,
.sdp-personal-settings__tab--active {
  border-color: var(--brand-300);
  color: var(--brand-900);
  background: var(--brand-50);
}

.sdp-personal-settings__tab strong {
  font-size: var(--text-sm);
}

.sdp-personal-settings__tab span {
  color: var(--ink-600);
  font-size: var(--text-xs);
  line-height: var(--leading-normal);
}

.sdp-personal-settings__panel {
  min-width: 0;
  padding: var(--space-5);
}

.sdp-personal-settings__stack {
  display: grid;
  gap: var(--space-5);
}

.sdp-personal-settings__section {
  padding: var(--space-6);
}

.sdp-personal-settings__section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
}

.sdp-personal-settings__badge {
  display: inline-flex;
  align-items: center;
  min-height: var(--space-6);
  padding-inline: var(--space-3);
  border: 1px solid var(--brand-300);
  border-radius: var(--radius-pill);
  color: var(--brand-900);
  background: var(--brand-50);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-semibold);
}

.sdp-personal-settings__form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-4);
}

.sdp-personal-settings__field {
  display: grid;
  gap: var(--space-2);
  color: var(--ink-700);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-semibold);
}

.sdp-personal-settings__field--wide {
  grid-column: 1 / -1;
}

.sdp-personal-settings__field input {
  width: 100%;
  min-height: var(--space-10);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-sm);
  color: var(--ink-900);
  background: var(--ink-50);
  font: inherit;
  font-weight: var(--font-weight-normal);
}

.sdp-personal-settings__field input:disabled {
  color: var(--ink-500);
  background: var(--ink-100);
}

.sdp-personal-settings__hint {
  margin: var(--space-3) 0 0;
  color: var(--ink-600);
  font-size: var(--text-xs);
}

.sdp-personal-settings__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--space-5);
}

.sdp-personal-settings__choice-grid,
.sdp-personal-settings__notification-list {
  display: grid;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  border: 0;
}

.sdp-personal-settings__choice-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.sdp-personal-settings__choice,
.sdp-personal-settings__notification {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-md);
  color: var(--ink-800);
  background: var(--ink-50);
  cursor: pointer;
}

.sdp-personal-settings__choice:has(input:checked),
.sdp-personal-settings__notification:has(input:checked) {
  border-color: var(--brand-500);
  background: var(--brand-50);
}

.sdp-personal-settings__choice input,
.sdp-personal-settings__notification input {
  width: var(--space-4);
  height: var(--space-4);
  flex: 0 0 var(--space-4);
  margin-top: var(--space-1);
  accent-color: var(--brand-600);
}

.sdp-personal-settings__choice-copy,
.sdp-personal-settings__notification > span {
  display: grid;
  gap: var(--space-1);
}

.sdp-personal-settings__choice strong,
.sdp-personal-settings__notification strong {
  font-size: var(--text-sm);
}

.sdp-personal-settings__choice small,
.sdp-personal-settings__notification small {
  color: var(--ink-600);
  font-size: var(--text-xs);
  line-height: var(--leading-relaxed);
}

.sdp-personal-settings__notification {
  align-items: center;
  justify-content: space-between;
}

@media (max-width: 64rem) {
  .sdp-personal-settings__layout {
    grid-template-columns: 1fr;
  }

  .sdp-personal-settings__tabs {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 40rem) {
  .sdp-personal-settings__shell {
    padding: var(--space-4);
  }

  .sdp-personal-settings__header,
  .sdp-personal-settings__section-head {
    flex-direction: column;
  }

  .sdp-personal-settings__header,
  .sdp-personal-settings__panel,
  .sdp-personal-settings__section {
    padding: var(--space-5);
  }

  .sdp-personal-settings__tabs,
  .sdp-personal-settings__form-grid,
  .sdp-personal-settings__choice-grid {
    grid-template-columns: 1fr;
  }
}
</style>
