<template>
  <div class="login-container">
    <section class="login-hero">
      <div class="hero-panel">
        <div class="hero-badge">企业知识智能平台</div>
        <div class="hero-logo">
          <svg viewBox="0 0 40 40" width="64" height="64" fill="none">
            <path d="M20 4L34 12V28L20 36L6 28V12L20 4Z" fill="#00B96B"/>
            <path d="M20 12L28 16V24L20 28L12 24V16L20 12Z" fill="#EAFBF2"/>
          </svg>
        </div>
        <h1 class="brand-name">随越·智枢</h1>
        <p class="brand-en">SmartKnora</p>
        <p class="tagline">让知识、问答与写作协同成为同一条工作流。</p>
        <ul class="hero-points">
          <li>知识空间集中管理企业资料</li>
          <li>AI 问答与 AI 写作共享统一知识底座</li>
          <li>延续 WeKnora 设计语言并升级品牌体验</li>
        </ul>
      </div>
    </section>

    <section class="login-side">
      <div class="login-card">
        <div class="login-card-head">
          <span class="login-eyebrow">欢迎回来</span>
          <h2 class="login-title">登录你的工作台</h2>
          <p class="login-subtitle">默认推荐手机号登录，邮箱登录作为备用入口保留。</p>
        </div>

        <t-tabs v-model="activeTab" class="login-tabs">
          <t-tab-panel value="phone" label="手机号登录">
            <div class="form-area">
              <t-input
                v-model="phoneForm.phone"
                placeholder="请输入手机号"
                :maxlength="11"
                @input="onPhoneInput"
                clearable
              >
                <template #prefix-icon>
                  <t-icon name="user" />
                </template>
              </t-input>
              <t-input
                v-model="phoneForm.password"
                type="password"
                placeholder="请输入密码"
                @enter="handlePhoneLogin"
              >
                <template #prefix-icon>
                  <t-icon name="lock-on" />
                </template>
              </t-input>
              <t-button theme="primary" block size="large" :loading="loading" :disabled="!agreed || !phoneForm.phone || !phoneForm.password" @click="handlePhoneLogin">
                登录
              </t-button>
            </div>
          </t-tab-panel>

          <t-tab-panel value="email" label="邮箱登录">
            <div class="form-area">
              <t-input v-model="emailForm.email" placeholder="请输入邮箱">
                <template #prefix-icon>
                  <t-icon name="mail" />
                </template>
              </t-input>
              <t-input
                v-model="emailForm.password"
                type="password"
                placeholder="请输入密码"
                @enter="handleEmailLogin"
              >
                <template #prefix-icon>
                  <t-icon name="lock-on" />
                </template>
              </t-input>
              <t-button theme="primary" block size="large" :loading="loading" :disabled="!agreed || !emailForm.email || !emailForm.password" @click="handleEmailLogin">
                登录
              </t-button>
            </div>
          </t-tab-panel>

          <t-tab-panel value="wechat" label="微信扫码" :disabled="true">
            <div class="wechat-placeholder">
              <t-icon name="qr-code" size="56px" />
              <p>微信扫码登录将在 Phase 2 上线</p>
            </div>
          </t-tab-panel>
        </t-tabs>

        <div class="login-footer">
          <t-checkbox v-model="agreed">
            我已阅读并同意 <t-link theme="primary">《服务协议》</t-link> 和 <t-link theme="primary">《隐私政策》</t-link>
          </t-checkbox>
        </div>

        <div class="register-link">
          还没有账号？<t-link theme="primary" @click="$router.push('/register')">立即注册</t-link>
        </div>

        <t-message v-if="errorMsg" theme="error" style="margin-top:12px">{{ errorMsg }}</t-message>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { login } from '@/api/auth'
import { MessagePlugin } from 'tdesign-vue-next'

const router = useRouter()
const authStore = useAuthStore()

const activeTab = ref('phone')
const agreed = ref(true)
const loading = ref(false)
const errorMsg = ref('')

const phoneForm = ref({ phone: '', password: '' })
const emailForm = ref({ email: '', password: '' })

function onPhoneInput(val: string) {
  phoneForm.value.phone = val.replace(/\D/g, '').slice(0, 11)
}

async function handlePhoneLogin() {
  if (!agreed.value) { errorMsg.value = '请先同意协议'; return }
  loading.value = true; errorMsg.value = ''
  try {
    const res = await login({ phone: phoneForm.value.phone, password: phoneForm.value.password })
    authStore.setAuth(res.data)
    MessagePlugin.success('登录成功')
    router.push('/spaces')
  } catch (e: any) {
    errorMsg.value = e.response?.data?.error || '登录失败，请检查账号密码'
  } finally { loading.value = false }
}

async function handleEmailLogin() {
  if (!agreed.value) { errorMsg.value = '请先同意协议'; return }
  loading.value = true; errorMsg.value = ''
  try {
    const res = await login({ email: emailForm.value.email, password: emailForm.value.password })
    authStore.setAuth(res.data)
    MessagePlugin.success('登录成功')
    router.push('/spaces')
  } catch (e: any) {
    errorMsg.value = e.response?.data?.error || '登录失败，请检查账号密码'
  } finally { loading.value = false }
}
</script>

<style scoped>
.login-container {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(440px, 0.8fr);
  background:
    radial-gradient(circle at top left, color-mix(in srgb, var(--brand-primary) 10%, transparent), transparent 34%),
    linear-gradient(135deg, var(--sk-bg) 0%, color-mix(in srgb, var(--sk-surface-soft) 88%, var(--sk-bg)) 100%);
}

.login-hero {
  padding: 40px;
  display: flex;
  align-items: stretch;
}

.hero-panel {
  flex: 1;
  border-radius: 32px;
  padding: 48px;
  color: var(--sk-sidebar-text);
  background:
    radial-gradient(circle at top right, rgba(126, 240, 182, 0.24), transparent 28%),
    linear-gradient(150deg, #163024 0%, #102219 35%, #00b96b 140%);
  box-shadow: 0 24px 80px rgba(13, 42, 26, 0.2);
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.hero-badge {
  width: fit-content;
  padding: 8px 14px;
  border-radius: 999px;
  background: color-mix(in srgb, white 12%, transparent);
  border: 1px solid color-mix(in srgb, white 14%, transparent);
  font-size: 12px;
  letter-spacing: 0.08em;
}

.hero-logo {
  margin-top: 24px;
}

.brand-name {
  margin: 22px 0 0;
  font-size: 42px;
  line-height: 1.05;
  font-weight: 700;
  letter-spacing: -0.05em;
}

.brand-en {
  margin-top: 10px;
  color: var(--sk-sidebar-muted);
  font-size: 15px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.tagline {
  max-width: 420px;
  margin-top: 24px;
  font-size: 20px;
  line-height: 1.6;
}

.hero-points {
  margin: 28px 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 12px;
}

.hero-points li {
  padding-left: 18px;
  position: relative;
  color: color-mix(in srgb, var(--sk-sidebar-text) 90%, transparent);
}

.hero-points li::before {
  content: "";
  position: absolute;
  left: 0;
  top: 11px;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #7ef0b6;
}

.login-side {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

.login-card {
  width: 100%;
  max-width: 440px;
  background: color-mix(in srgb, var(--surface-elevated) 96%, transparent);
  border: 1px solid rgba(219, 229, 219, 0.9);
  border-radius: 28px;
  padding: 36px;
  box-shadow: 0 20px 60px rgba(24, 43, 28, 0.08);
  backdrop-filter: blur(20px);
}

.login-card-head {
  margin-bottom: 16px;
}

.login-eyebrow {
  display: inline-block;
  margin-bottom: 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--brand-primary);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.login-title {
  margin: 0;
  font-size: 28px;
  line-height: 1.15;
  color: var(--text-primary);
  letter-spacing: -0.04em;
}

.login-subtitle {
  margin-top: 10px;
  color: var(--text-secondary);
  font-size: 14px;
}

.login-tabs {
  margin-bottom: 18px;
}

.form-area {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px 0 8px;
}

.wechat-placeholder {
  text-align: center;
  padding: 48px 0;
  color: var(--text-muted);
}

.wechat-placeholder p {
  margin-top: 10px;
  font-size: 14px;
}

.login-footer {
  margin-top: 14px;
}

.register-link {
  margin-top: 16px;
  text-align: center;
  font-size: 14px;
  color: var(--text-secondary);
}

@media (max-width: 1080px) {
  .login-container {
    grid-template-columns: 1fr;
  }

  .login-hero {
    padding: 24px 24px 0;
  }

  .hero-panel {
    padding: 32px;
  }

  .login-side {
    padding: 24px;
  }
}
</style>
