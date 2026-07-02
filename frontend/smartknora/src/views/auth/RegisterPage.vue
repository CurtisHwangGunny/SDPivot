<template>
  <div class="register-container">
    <div class="register-card">
      <h2 class="register-title">注册账号</h2>
      <div class="form-area">
        <t-input v-model="form.phone" placeholder="手机号" :maxlength="11" @input="(v:string)=>form.phone=v.replace(/\D/g,'').slice(0,11)">
          <template #prefix-icon><t-icon name="user" /></template>
        </t-input>
        <t-input v-model="form.nickname" placeholder="昵称（选填）">
          <template #prefix-icon><t-icon name="user-circle" /></template>
        </t-input>
        <t-input v-model="form.password" type="password" placeholder="密码（至少6位）">
          <template #prefix-icon><t-icon name="lock-on" /></template>
        </t-input>
        <t-input v-model="form.confirmPassword" type="password" placeholder="确认密码">
          <template #prefix-icon><t-icon name="lock-on" /></template>
        </t-input>
        <t-checkbox v-model="agreed">
          我已阅读并同意 <t-link theme="primary">《服务协议》</t-link> 和 <t-link theme="primary">《隐私政策》</t-link>
        </t-checkbox>
        <t-button theme="primary" block size="large" :loading="loading" :disabled="!agreed || !form.phone || !form.password || form.password !== form.confirmPassword" @click="handleRegister">
          注册
        </t-button>
      </div>
      <div class="login-link">
        已有账号？<t-link theme="primary" @click="$router.push('/login')">返回登录</t-link>
      </div>
      <t-message v-if="errorMsg" theme="error" style="margin-top:12px">{{ errorMsg }}</t-message>
      <t-message v-if="successMsg" theme="success" style="margin-top:12px">{{ successMsg }}</t-message>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { MessagePlugin } from 'tdesign-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const agreed = ref(true)
const loading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const form = ref({ phone: '', nickname: '', password: '', confirmPassword: '' })

async function handleRegister() {
  if (form.value.password !== form.value.confirmPassword) { errorMsg.value = '两次密码不一致'; return }
  if (form.value.password.length < 6) { errorMsg.value = '密码至少6位'; return }
  loading.value = true; errorMsg.value = ''; successMsg.value = ''
  try {
    const res = await register({ phone: form.value.phone, password: form.value.password, nickname: form.value.nickname })
    authStore.setAuth(res.data)
    MessagePlugin.success('注册成功，欢迎加入！')
    router.push('/spaces')
  } catch (e: any) {
    errorMsg.value = e.response?.data?.error || '注册失败'
  } finally { loading.value = false }
}
</script>
<style scoped>
.register-container { display: flex; align-items: center; justify-content: center; min-height: 100vh; background: linear-gradient(135deg, #014DB2 0%, #0A7AFF 100%); }
.register-card { width: 420px; background: #fff; border-radius: 12px; padding: 40px 36px; box-shadow: 0 4px 24px rgba(0,0,0,0.15); }
.register-title { font-size: 24px; font-weight: 600; color: #0a1628; margin-bottom: 24px; text-align: center; }
.form-area { display: flex; flex-direction: column; gap: 16px; }
.login-link { margin-top: 16px; text-align: center; font-size: 14px; color: #666; }
</style>
