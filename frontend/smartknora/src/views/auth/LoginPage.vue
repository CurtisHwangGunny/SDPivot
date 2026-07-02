<template>
  <div class="login-container">
    <div class="login-left">
      <div class="brand">
        <div class="logo-hex">
          <svg viewBox="0 0 40 40" width="56" height="56" fill="none">
            <path d="M20 4L34 12V28L20 36L6 28V12L20 4Z" fill="#014DB2"/>
            <path d="M20 12L28 16V24L20 28L12 24V16L20 12Z" fill="#F59E0B"/>
          </svg>
        </div>
        <h1 class="brand-name">随越·智枢</h1>
        <p class="brand-en">smartKnora</p>
        <p class="tagline">企业知识智能平台</p>
        <p class="desc">让知识成为企业核心竞争力</p>
      </div>
    </div>

    <div class="login-right">
      <div class="login-card">
        <h2 class="login-title">登录</h2>

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
              <t-input
                v-model="emailForm.email"
                placeholder="请输入邮箱"
              >
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
              <t-icon name="qr-code" size="64px" style="color:#d0d0d0" />
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
    </div>
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
.login-container { display: flex; min-height: 100vh; }
.login-left {
  flex: 1; display: flex; align-items: center; justify-content: center;
  background: linear-gradient(135deg, #014DB2 0%, #0A7AFF 100%);
  color: #fff; padding: 40px;
}
.brand { text-align: center; }
.logo-hex { margin-bottom: 24px; display: flex; justify-content: center; }
.brand-name { font-size: 32px; font-weight: 700; letter-spacing: 1px; }
.brand-en { font-size: 16px; opacity: 0.7; margin-top: 4px; }
.tagline { font-size: 20px; margin-top: 20px; opacity: 0.9; }
.desc { font-size: 14px; opacity: 0.7; margin-top: 8px; }
.login-right {
  flex: 1; display: flex; align-items: center; justify-content: center; padding: 40px;
}
.login-card {
  width: 100%; max-width: 420px; background: #fff; border-radius: 12px;
  padding: 40px 36px; box-shadow: 0 2px 16px rgba(0,0,0,0.08);
}
.login-title { font-size: 24px; font-weight: 600; color: #0a1628; margin-bottom: 24px; }
.login-tabs { margin-bottom: 16px; }
.form-area { display: flex; flex-direction: column; gap: 16px; padding: 20px 0; }
.wechat-placeholder { text-align: center; padding: 60px 0; color: #999; }
.wechat-placeholder p { margin-top: 12px; font-size: 14px; }
.login-footer { margin-top: 16px; }
.register-link { margin-top: 16px; text-align: center; font-size: 14px; color: #666; }
</style>
