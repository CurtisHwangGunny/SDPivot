<template>
  <div class="ops-shell">
    <section class="ops-brand-panel">
      <div class="brand-badge">Operations access</div>
      <h1>运营管理端登录</h1>
      <p>面向运营管理员的专用入口。保留首次登录改密流程，同时将视觉语言对齐到新版品牌体系。</p>
      <div class="ops-brand-points">
        <div class="brand-point">
          <strong>权限隔离</strong>
          <span>与普通用户登录态分离管理</span>
        </div>
        <div class="brand-point">
          <strong>首次改密</strong>
          <span>保留 must_change_password 流程</span>
        </div>
        <div class="brand-point">
          <strong>统一品牌</strong>
          <span>绿色体系与工作台骨架一致</span>
        </div>
      </div>
    </section>

    <section class="ops-form-panel">
      <div class="ops-form-card">
        <div class="ops-form-head">
          <p class="ops-eyebrow">Secure sign in</p>
          <h2>{{ mustChangePassword ? '首次登录，先修改密码' : '登录运营管理端' }}</h2>
          <p>{{ mustChangePassword ? '密码需满足 8 位以上，并包含大小写字母、数字与特殊字符。' : '请输入运营管理员账号与密码。' }}</p>
        </div>

        <div v-if="mustChangePassword" class="ops-change-password">
          <t-form ref="pwdForm" :data="pwdFormData" :rules="pwdRules" @submit="handleChangePassword">
            <t-form-item name="oldPassword" label="当前密码">
              <t-input v-model="pwdFormData.oldPassword" type="password" placeholder="请输入当前密码" />
            </t-form-item>
            <t-form-item name="newPassword" label="新密码">
              <t-input v-model="pwdFormData.newPassword" type="password" placeholder="请输入新密码" />
            </t-form-item>
            <t-form-item name="confirmPassword" label="确认新密码">
              <t-input v-model="pwdFormData.confirmPassword" type="password" placeholder="请再次输入新密码" />
            </t-form-item>
            <t-button theme="primary" type="submit" :loading="changing" class="ops-login-btn">确认修改并继续</t-button>
          </t-form>
          <p v-if="error" class="error-msg">{{ error }}</p>
        </div>

        <div v-else class="ops-login-form">
          <t-form ref="loginForm" :data="loginData" :rules="loginRules" @submit="handleLogin">
            <t-form-item name="email" label="运营管理员账号">
              <t-input v-model="loginData.email" placeholder="请输入运营管理员账号" clearable />
            </t-form-item>
            <t-form-item name="password" label="密码">
              <t-input v-model="loginData.password" type="password" placeholder="请输入密码" />
            </t-form-item>
            <t-button theme="primary" type="submit" :loading="logging" class="ops-login-btn">登录运营管理端</t-button>
          </t-form>
          <p v-if="error" class="error-msg">{{ error }}</p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import axios from 'axios'

const router = useRouter()
const logging = ref(false)
const changing = ref(false)
const mustChangePassword = ref(false)
const error = ref('')
const currentToken = ref('')

const loginData = reactive({
  email: '',
  password: '',
})

const pwdFormData = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const loginRules = {
  email: [{ required: true, message: '请输入账号' }],
  password: [{ required: true, message: '请输入密码' }],
}

const passwordPattern = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[!@#$%^&*()_+\-=[\]{};':"\|,.<>/?]).{8,}$/

const pwdRules = {
  oldPassword: [{ required: true, message: '请输入当前密码' }],
  newPassword: [
    { required: true, message: '请输入新密码' },
    { min: 8, message: '密码至少8位' },
    {
      validator: (val: string) => passwordPattern.test(val),
      message: '必须包含大小写字母+数字+特殊字符',
    },
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码' },
    {
      validator: (val: string) => val === pwdFormData.newPassword,
      message: '两次密码不一致',
    },
  ],
}

async function handleLogin() {
  logging.value = true
  error.value = ''

  try {
    const res = await axios.post('/api/v1/smartknora/ops/login', {
      email: loginData.email,
      password: loginData.password,
    })

    if (res.data.must_change_password) {
      mustChangePassword.value = true
      currentToken.value = res.data.access_token
      return
    }

    localStorage.setItem('ops_access_token', res.data.access_token)
    localStorage.setItem('ops_refresh_token', res.data.refresh_token)
    localStorage.setItem('ops_user', JSON.stringify(res.data.user))
    router.push('/ops')
  } catch (e: any) {
    error.value = e.response?.data?.error || '登录失败'
  } finally {
    logging.value = false
  }
}

async function handleChangePassword() {
  changing.value = true
  error.value = ''

  try {
    await axios.post(
      '/api/v1/smartknora/ops/change-password',
      {
        old_password: pwdFormData.oldPassword,
        new_password: pwdFormData.newPassword,
      },
      { headers: { Authorization: `Bearer ${currentToken.value}` } },
    )

    MessagePlugin.success('密码修改成功，请重新登录')
    mustChangePassword.value = false
    currentToken.value = ''
    loginData.password = ''
    pwdFormData.oldPassword = ''
    pwdFormData.newPassword = ''
    pwdFormData.confirmPassword = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || '修改密码失败'
  } finally {
    changing.value = false
  }
}
</script>

<style scoped>
.ops-shell {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(420px, 0.85fr);
  background:
    radial-gradient(circle at top left, rgba(0, 185, 107, 0.2), transparent 34%),
    linear-gradient(135deg, #f4fbf7 0%, #eef3f8 48%, #f9fbfd 100%);
}
.ops-brand-panel {
  padding: 72px clamp(28px, 5vw, 72px);
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 24px;
  background: linear-gradient(160deg, #112018 0%, #1f2937 100%);
  color: var(--sk-sidebar-text);
}
.brand-badge,
.ops-eyebrow {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--sk-sidebar-muted);
}
.ops-brand-panel h1,
.ops-form-head h2 {
  margin: 0;
  font-size: clamp(32px, 4vw, 48px);
  line-height: 1.08;
}
.ops-brand-panel p,
.ops-form-head p {
  margin: 0;
  max-width: 560px;
  color: var(--sk-sidebar-muted);
  line-height: 1.8;
}
.ops-brand-points {
  display: grid;
  gap: 14px;
}
.brand-point {
  border-radius: 20px;
  border: 1px solid var(--sidebar-border);
  background: color-mix(in srgb, white 6%, transparent);
  padding: 18px 20px;
  backdrop-filter: blur(10px);
}
.brand-point strong {
  display: block;
  margin-bottom: 6px;
}
.brand-point span {
  color: color-mix(in srgb, var(--sk-sidebar-text) 70%, transparent);
  font-size: 14px;
}
.ops-form-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
}
.ops-form-card {
  width: min(460px, 100%);
  border-radius: 28px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: color-mix(in srgb, var(--surface-elevated) 94%, transparent);
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.12);
  padding: 32px;
}
.ops-form-head {
  margin-bottom: 24px;
}
.ops-form-head .ops-eyebrow {
  color: var(--brand-primary);
}
.ops-form-head h2 {
  font-size: 28px;
  color: var(--text-primary);
}
.ops-form-head p {
  margin-top: 12px;
  color: var(--text-secondary);
}
.ops-login-btn {
  width: 100% !important;
  height: 44px;
  margin-top: 8px;
}
.error-msg {
  margin-top: 14px;
  text-align: center;
  color: var(--td-error-color, #e34d59);
  font-size: 13px;
}
@media (max-width: 980px) {
  .ops-shell {
    grid-template-columns: 1fr;
  }
  .ops-brand-panel {
    padding: 48px 24px 28px;
  }
  .ops-form-panel {
    padding-top: 0;
  }
}
</style>
