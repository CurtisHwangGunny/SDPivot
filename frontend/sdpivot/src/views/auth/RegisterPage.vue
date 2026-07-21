<template>
  <div class="register-shell">
    <section class="register-brand-panel">
      <div class="brand-badge">Create account</div>
      <h1>加入 SDPivot</h1>
      <p>创建你的专属账号，进入统一的知识空间、AI 问答与 AI 写作工作台。</p>
      <div class="brand-points">
        <div class="brand-point">
          <strong>统一工作台</strong>
          <span>空间、问答、写作与设置体验已经开始全面对齐</span>
        </div>
        <div class="brand-point">
          <strong>安全注册</strong>
          <span>注册成功后自动登录，并直接进入知识空间首页</span>
        </div>
      </div>
    </section>

    <section class="register-form-panel">
      <div class="register-card">
        <div class="register-head">
          <p class="register-eyebrow">Sign up</p>
          <h2>注册账号</h2>
          <p>使用手机号快速创建账号，后续可在设置中心补充更多资料与偏好。</p>
        </div>

        <div class="form-area">
          <t-input v-model="form.phone" placeholder="手机号" :maxlength="11" @input="(v:string)=>form.phone=v.replace(/\D/g,'').slice(0,11)">
            <template #prefix-icon><t-icon name="user" /></template>
          </t-input>
          <t-input v-model="form.nickname" placeholder="昵称（选填）">
            <template #prefix-icon><t-icon name="user-circle" /></template>
          </t-input>
          <t-input v-model="form.password" type="password" placeholder="密码（至少8位）">
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
    </section>
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
  if (form.value.password.length < 8) { errorMsg.value = '密码至少8位'; return }
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
.register-shell {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(420px, 0.9fr);
  background:
    radial-gradient(circle at top left, rgba(0, 185, 107, 0.18), transparent 32%),
    linear-gradient(135deg, #f4fbf7 0%, #eef3f8 48%, #f9fbfd 100%);
}
.register-brand-panel {
  padding: 72px clamp(28px, 5vw, 72px);
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 24px;
  background: linear-gradient(160deg, #112018 0%, #1f2937 100%);
  color: rgba(255, 255, 255, 0.94);
}
.brand-badge,
.register-eyebrow {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.7);
}
.register-brand-panel h1,
.register-head h2 {
  margin: 0;
  font-size: clamp(32px, 4vw, 48px);
  line-height: 1.08;
}
.register-brand-panel p,
.register-head p {
  margin: 0;
  max-width: 560px;
  color: rgba(255, 255, 255, 0.72);
  line-height: 1.8;
}
.brand-points {
  display: grid;
  gap: 14px;
}
.brand-point {
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.06);
  padding: 18px 20px;
  backdrop-filter: blur(10px);
}
.brand-point strong {
  display: block;
  margin-bottom: 6px;
}
.brand-point span {
  color: rgba(255, 255, 255, 0.66);
  font-size: 14px;
}
.register-form-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
}
.register-card {
  width: min(460px, 100%);
  background: color-mix(in srgb, var(--surface-elevated) 94%, transparent);
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.12);
  border: 1px solid var(--border-soft);
}
.register-head {
  margin-bottom: 24px;
}
.register-head .register-eyebrow {
  color: var(--brand-primary);
}
.register-head h2 {
  font-size: 28px;
  color: var(--text-primary);
}
.register-head p {
  margin-top: 12px;
  color: var(--text-secondary);
}
.form-area { display: flex; flex-direction: column; gap: 16px; }
.login-link { margin-top: 16px; text-align: center; font-size: 14px; color: var(--text-secondary); }
@media (max-width: 980px) {
  .register-shell {
    grid-template-columns: 1fr;
  }
  .register-brand-panel {
    padding: 48px 24px 28px;
  }
  .register-form-panel {
    padding-top: 0;
  }
}
</style>
