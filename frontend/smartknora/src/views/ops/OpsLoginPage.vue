<template>
  <div class="ops-login-container">
    <div class="ops-login-card">
      <div class="ops-login-header">
        <h1>随越·智枢</h1>
        <p class="ops-subtitle">运营管理端</p>
      </div>

      <!-- 首次登录强制修改密码 -->
      <div v-if="mustChangePassword" class="ops-change-password">
        <h3>首次登录 · 修改密码</h3>
        <p class="hint">密码需满足：≥8位，包含大小写字母+数字+特殊字符</p>
        <t-form ref="pwdForm" :data="pwdFormData" :rules="pwdRules" @submit="handleChangePassword">
          <t-form-item name="oldPassword">
            <t-input v-model="pwdFormData.oldPassword" type="password" placeholder="当前密码" />
          </t-form-item>
          <t-form-item name="newPassword">
            <t-input v-model="pwdFormData.newPassword" type="password" placeholder="新密码" />
          </t-form-item>
          <t-form-item name="confirmPassword">
            <t-input v-model="pwdFormData.confirmPassword" type="password" placeholder="确认新密码" />
          </t-form-item>
          <t-button theme="primary" type="submit" :loading="changing" class="ops-login-btn">
            确认修改
          </t-button>
        </t-form>
      </div>

      <!-- 正常登录 -->
      <div v-else class="ops-login-form">
        <t-form ref="loginForm" :data="loginData" :rules="loginRules" @submit="handleLogin">
          <t-form-item name="email">
            <t-input v-model="loginData.email" placeholder="运营管理员账号" clearable />
          </t-form-item>
          <t-form-item name="password">
            <t-input v-model="loginData.password" type="password" placeholder="密码" />
          </t-form-item>
          <t-button theme="primary" type="submit" :loading="logging" class="ops-login-btn">
            登录运营管理端
          </t-button>
        </t-form>
        <p v-if="error" class="error-msg">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import axios from 'axios'

const router = useRouter()
const logging = ref(false)
const changing = ref(false)
const mustChangePassword = ref(false)
const error = ref('')
const currentToken = ref('')

const loginData = reactive({ email: '', password: '' })
const pwdFormData = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const loginRules = {
  email: [{ required: true, message: '请输入账号' }],
  password: [{ required: true, message: '请输入密码' }],
}

const pwdRules = {
  oldPassword: [{ required: true, message: '请输入当前密码' }],
  newPassword: [
    { required: true, message: '请输入新密码' },
    { min: 8, message: '密码至少8位' },
    {
      validator: (val: string) => /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]).{8,}$/.test(val),
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
    } else {
      // 正常进入运营管理端
      localStorage.setItem('ops_access_token', res.data.access_token)
      localStorage.setItem('ops_refresh_token', res.data.refresh_token)
      localStorage.setItem('ops_user', JSON.stringify(res.data.user))
      router.push('/ops/dashboard')
    }
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
    const res = await axios.post(
      '/api/v1/smartknora/ops/change-password',
      {
        old_password: pwdFormData.oldPassword,
        new_password: pwdFormData.newPassword,
      },
      { headers: { Authorization: `Bearer ${currentToken.value}` } }
    )
    MessagePlugin.success('密码修改成功，请重新登录')
    mustChangePassword.value = false
    currentToken.value = ''
    loginData.password = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || '修改密码失败'
  } finally {
    changing.value = false
  }
}
</script>

<style scoped>
.ops-login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #014db2 0%, #003d8f 100%);
}
.ops-login-card {
  width: 420px;
  max-width: 90vw;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  box-sizing: border-box;
}
.ops-login-header {
  text-align: center;
  margin-bottom: 32px;
}
.ops-login-header h1 {
  font-size: 24px;
  color: #014db2;
  margin: 0;
}
.ops-subtitle {
  font-size: 14px;
  color: #666;
  margin-top: 8px;
}
.ops-change-password h3 {
  font-size: 16px;
  margin-bottom: 16px;
  text-align: center;
}
.hint {
  font-size: 12px;
  color: #999;
  text-align: center;
  margin-bottom: 16px;
}
.ops-login-form {
  margin-top: 16px;
}

.ops-login-form .t-form-item {
  margin-bottom: 20px;
}

.ops-login-btn {
  width: 50%;
  height: 40px;
  font-size: 14px;
  margin-top: 8px;
  display: block;
  margin-left: auto;
  margin-right: auto;
}
.error-msg {
  color: #e34d59;
  font-size: 13px;
  text-align: center;
  margin-top: 12px;
}
</style>
