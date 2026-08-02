<template>
  <main class="sdp-register">
    <section class="sdp-register__brand" aria-labelledby="sdp-register-brand-title">
      <div class="sdp-register__brand-mark" aria-hidden="true">SD</div>
      <div class="sdp-register__brand-copy">
        <p class="sdp-register__eyebrow">Private Deployment</p>
        <h1 id="sdp-register-brand-title">企业知识中枢，从安全开通开始</h1>
        <p>注册入口用于私有化部署初始管理员创建、试用演示与受控邀请场景。</p>
      </div>
      <dl class="sdp-register__stats" aria-label="注册入口说明">
        <div>
          <dt>OP</dt>
          <dd>私有化部署</dd>
        </div>
        <div>
          <dt>AA</dt>
          <dd>无障碍标准</dd>
        </div>
        <div>
          <dt>2FA</dt>
          <dd>安全校验预留</dd>
        </div>
      </dl>
    </section>

    <section class="sdp-register__form-panel" aria-labelledby="sdp-register-title">
      <form class="sdp-register__card" aria-label="创建管理员账号" @submit.prevent="handleRegister">
        <header class="sdp-register__header">
          <p class="sdp-register__eyebrow">Create Admin Account</p>
          <h2 id="sdp-register-title">创建管理员账号</h2>
          <p>仅用于安装向导或管理员邀请链路。生产环境建议关闭公开注册。</p>
        </header>

        <div class="sdp-register__fields">
          <div class="sdp-register__field">
            <label for="sdp-register-name">管理员姓名</label>
            <input
              id="sdp-register-name"
              v-model.trim="form.nickname"
              class="sdp-register__input"
              type="text"
              autocomplete="name"
              aria-label="管理员姓名"
              placeholder="请输入管理员姓名"
            />
          </div>

          <div class="sdp-register__field">
            <label for="sdp-register-phone">手机号</label>
            <input
              id="sdp-register-phone"
              :value="form.phone"
              class="sdp-register__input"
              type="tel"
              inputmode="numeric"
              autocomplete="tel"
              maxlength="11"
              required
              aria-label="管理员手机号"
              aria-describedby="sdp-register-phone-help"
              placeholder="请输入 11 位手机号"
              @input="onPhoneInput"
            />
            <span id="sdp-register-phone-help" class="sdp-register__help">手机号将作为管理员登录账号</span>
          </div>

          <div class="sdp-register__field">
            <label for="sdp-register-password">登录密码</label>
            <input
              id="sdp-register-password"
              v-model="form.password"
              class="sdp-register__input"
              type="password"
              autocomplete="new-password"
              minlength="8"
              required
              aria-label="登录密码"
              aria-describedby="sdp-register-password-help"
              placeholder="至少 8 位字符"
            />
            <span id="sdp-register-password-help" class="sdp-register__help">请使用至少 8 位字符</span>
          </div>

          <div class="sdp-register__field">
            <label for="sdp-register-confirm-password">确认密码</label>
            <input
              id="sdp-register-confirm-password"
              v-model="form.confirmPassword"
              class="sdp-register__input"
              type="password"
              autocomplete="new-password"
              minlength="8"
              required
              aria-label="确认登录密码"
              placeholder="再次输入登录密码"
            />
          </div>
        </div>

        <label class="sdp-register__agreement">
          <input v-model="accepted" type="checkbox" required aria-label="同意私有化部署协议" />
          <span>我已阅读并同意私有化部署协议</span>
        </label>

        <p v-if="errorMessage" class="sdp-register__error" role="alert" aria-live="assertive">
          {{ errorMessage }}
        </p>

        <SdpButton
          class="sdp-register__submit"
          size="lg"
          :loading="loading"
          :disabled="!canSubmit"
          aria-label="创建管理员账号"
          @click="handleRegister"
        >
          创建账号
        </SdpButton>

        <p class="sdp-register__login-link">
          已有账号？
          <RouterLink to="/login" aria-label="返回登录页面">返回登录</RouterLink>
        </p>
      </form>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { register } from '@/api/auth'
import SdpButton from '@/components/design/SdpButton.vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const errorMessage = ref('')
const accepted = ref(false)
const form = reactive({
  nickname: '',
  phone: '',
  password: '',
  confirmPassword: '',
})

const canSubmit = computed(() => (
  accepted.value
  && form.phone.length === 11
  && form.password.length >= 8
  && form.password === form.confirmPassword
))

function onPhoneInput(event: Event) {
  form.phone = (event.target as HTMLInputElement).value.replace(/\D/g, '').slice(0, 11)
}

function getRegisterError(error: unknown) {
  if (typeof error === 'object' && error !== null) {
    const requestError = error as {
      message?: string
      response?: { data?: { error?: string; message?: string } }
    }
    return requestError.response?.data?.error
      || requestError.response?.data?.message
      || requestError.message
      || '注册失败，请稍后重试'
  }
  return '注册失败，请稍后重试'
}

async function handleRegister() {
  if (!canSubmit.value || loading.value) return

  loading.value = true
  errorMessage.value = ''
  try {
    await register({
      phone: form.phone,
      password: form.password,
      nickname: form.nickname || undefined,
    })
    authStore.clearAuth()
    await router.push('/login')
  } catch (error: unknown) {
    errorMessage.value = getRegisterError(error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.sdp-register {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  color: var(--ink-900);
  background: var(--ink-50);
  font-family: var(--font-body);
}

.sdp-register__brand {
  display: flex;
  min-height: 100dvh;
  flex-direction: column;
  justify-content: space-between;
  gap: var(--space-12);
  padding: var(--space-14);
  color: var(--ink-50);
  background:
    radial-gradient(circle at top left, var(--brand-600), transparent),
    linear-gradient(to bottom right, var(--ink-950), var(--brand-900));
}

.sdp-register__brand-mark {
  display: grid;
  width: var(--space-10);
  height: var(--space-10);
  place-items: center;
  border-radius: var(--radius-md);
  color: var(--ink-950);
  background: var(--brand-400);
  font-family: var(--font-display);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-extrabold);
}

.sdp-register__brand-copy {
  max-width: calc(var(--space-24) * 5);
}

.sdp-register__eyebrow {
  color: var(--brand-300);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-semibold);
  letter-spacing: var(--space-1);
  text-transform: uppercase;
}

.sdp-register__brand-copy h1 {
  max-width: calc(var(--space-24) * 4);
  margin-top: var(--space-4);
  color: var(--ink-50);
  font-family: var(--font-display);
  font-size: var(--text-4xl);
  font-weight: var(--font-weight-bold);
  line-height: var(--leading-tight);
}

.sdp-register__brand-copy > p:last-child {
  margin-top: var(--space-6);
  color: var(--ink-200);
  font-size: var(--text-base);
  line-height: var(--leading-relaxed);
}

.sdp-register__stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-3);
}

.sdp-register__stats div {
  padding: var(--space-4);
  border: 1px solid var(--ink-700);
  border-radius: var(--radius-lg);
  background: var(--ink-900);
}

.sdp-register__stats dt {
  color: var(--brand-300);
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: var(--font-weight-bold);
}

.sdp-register__stats dd {
  margin-top: var(--space-1);
  color: var(--ink-300);
  font-size: var(--text-xs);
}

.sdp-register__form-panel {
  display: grid;
  min-height: 100dvh;
  place-items: center;
  padding: var(--space-12);
}

.sdp-register__card {
  display: grid;
  width: 100%;
  max-width: calc(var(--space-24) * 5);
  gap: var(--space-6);
  padding: var(--space-8);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-xl);
  background: var(--ink-50);
  box-shadow: var(--shadow-lg);
}

.sdp-register__header h2 {
  margin-top: var(--space-3);
  color: var(--ink-950);
  font-family: var(--font-display);
  font-size: var(--text-3xl);
  font-weight: var(--font-weight-bold);
  line-height: var(--leading-tight);
}

.sdp-register__header > p:last-child {
  margin-top: var(--space-3);
  color: var(--ink-600);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}

.sdp-register__fields {
  display: grid;
  gap: var(--space-4);
}

.sdp-register__field {
  display: grid;
  gap: var(--space-2);
}

.sdp-register__field label {
  color: var(--ink-800);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-semibold);
}

.sdp-register__input {
  width: 100%;
  min-height: var(--space-12);
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-sm);
  color: var(--ink-950);
  background: var(--ink-50);
  font-family: var(--font-body);
  font-size: var(--text-base);
  line-height: var(--leading-normal);
  transition:
    border-color var(--duration-fast) var(--ease-out-quart),
    box-shadow var(--duration-fast) var(--ease-out-quart);
}

.sdp-register__input::placeholder {
  color: var(--ink-500);
}

.sdp-register__input:hover {
  border-color: var(--ink-500);
}

.sdp-register__input:focus-visible,
.sdp-register__agreement input:focus-visible,
.sdp-register__login-link a:focus-visible {
  outline: 2px solid var(--brand-500);
  outline-offset: var(--space-1);
}

.sdp-register__input:focus-visible {
  border-color: var(--brand-600);
  box-shadow: var(--shadow-xs);
}

.sdp-register__help {
  color: var(--ink-600);
  font-size: var(--text-xs);
}

.sdp-register__agreement {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  color: var(--ink-700);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
  cursor: pointer;
}

.sdp-register__agreement input {
  width: var(--space-4);
  height: var(--space-4);
  flex: 0 0 var(--space-4);
  margin-top: var(--space-1);
  accent-color: var(--brand-600);
}

.sdp-register__error {
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--ink-500);
  border-radius: var(--radius-sm);
  color: var(--ink-950);
  background: var(--ink-100);
  font-size: var(--text-sm);
}

.sdp-register__submit {
  width: 100%;
}

.sdp-register__login-link {
  color: var(--ink-600);
  font-size: var(--text-sm);
  text-align: center;
}

.sdp-register__login-link a {
  color: var(--brand-700);
  font-weight: var(--font-weight-semibold);
}

@media (max-width: 1023px) {
  .sdp-register {
    grid-template-columns: minmax(0, 1fr);
  }

  .sdp-register__brand,
  .sdp-register__form-panel {
    min-height: auto;
    padding: var(--space-8);
  }

  .sdp-register__brand {
    gap: var(--space-8);
  }
}

@media (max-width: 639px) {
  .sdp-register__brand,
  .sdp-register__form-panel {
    padding: var(--space-6);
  }

  .sdp-register__stats {
    grid-template-columns: minmax(0, 1fr);
  }

  .sdp-register__card {
    padding: var(--space-6);
  }
}
</style>
