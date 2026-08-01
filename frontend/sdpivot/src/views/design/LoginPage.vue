<template>
  <main class="sdp-login">
    <section class="sdp-login__brand" aria-labelledby="sdp-login-brand">
      <div class="sdp-login__brand-content">
        <div class="sdp-login__identity">
          <svg class="sdp-login__logo" viewBox="0 0 48 48" role="img" aria-label="SDPivot logo">
            <path d="M24 4 42 14v20L24 44 6 34V14L24 4Z" />
            <path d="m24 14 9 5v10l-9 5-9-5V19l9-5Z" />
          </svg>
          <div>
            <p class="sdp-login__product">企业知识智能平台</p>
            <h1 id="sdp-login-brand">SDPivot</h1>
          </div>
        </div>

        <p class="sdp-login__statement">让组织知识流动起来，让每一次提问都有据可依。</p>

        <ul class="sdp-login__values" aria-label="平台价值">
          <li>
            <span class="sdp-login__value-icon" aria-hidden="true">01</span>
            <div>
              <strong>统一知识空间</strong>
              <p>集中沉淀、管理并检索企业核心资料。</p>
            </div>
          </li>
          <li>
            <span class="sdp-login__value-icon" aria-hidden="true">02</span>
            <div>
              <strong>可信智能问答</strong>
              <p>基于组织知识生成可追溯、可验证的答案。</p>
            </div>
          </li>
          <li>
            <span class="sdp-login__value-icon" aria-hidden="true">03</span>
            <div>
              <strong>协同内容创作</strong>
              <p>让知识检索、问答与写作共享同一工作流。</p>
            </div>
          </li>
        </ul>

        <dl class="sdp-login__stats" aria-label="平台数据">
          <div>
            <dt>10×</dt>
            <dd>知识检索效率</dd>
          </div>
          <div>
            <dt>99.9%</dt>
            <dd>服务可用性</dd>
          </div>
          <div>
            <dt>7×24</dt>
            <dd>智能知识服务</dd>
          </div>
        </dl>
      </div>
    </section>

    <section class="sdp-login__form-panel" aria-labelledby="sdp-login-title">
      <div class="sdp-login__form-shell">
        <header class="sdp-login__header">
          <p>SDPivot Workspace</p>
          <h2 id="sdp-login-title">欢迎回来</h2>
          <span>登录后继续访问你的知识工作台</span>
        </header>

        <t-tabs v-model="activeTab" class="sdp-login__tabs" aria-label="选择登录方式">
          <t-tab-panel value="phone" label="手机号登录">
            <form class="sdp-login__form" aria-label="手机号登录表单" @submit.prevent="handlePhoneLogin">
              <div class="sdp-login__field">
                <label for="sdp-login-phone">手机号</label>
                <t-input
                  id="sdp-login-phone"
                  v-model="phoneForm.phone"
                  :maxlength="11"
                  autocomplete="tel"
                  inputmode="numeric"
                  placeholder="请输入 11 位手机号"
                  aria-label="手机号"
                  @input="onPhoneInput"
                />
              </div>
              <div class="sdp-login__field">
                <label for="sdp-login-phone-password">密码</label>
                <t-input
                  id="sdp-login-phone-password"
                  v-model="phoneForm.password"
                  type="password"
                  autocomplete="current-password"
                  placeholder="请输入密码"
                  aria-label="手机号登录密码"
                  @enter="handlePhoneLogin"
                />
              </div>
              <SdpButton
                class="sdp-login__submit"
                size="lg"
                :loading="loading"
                :disabled="!canSubmitPhone"
                aria-label="使用手机号登录"
                @click="handlePhoneLogin"
              >
                登录
              </SdpButton>
            </form>
          </t-tab-panel>

          <t-tab-panel value="email" label="邮箱登录">
            <form class="sdp-login__form" aria-label="邮箱登录表单" @submit.prevent="handleEmailLogin">
              <div class="sdp-login__field">
                <label for="sdp-login-email">邮箱</label>
                <t-input
                  id="sdp-login-email"
                  v-model="emailForm.email"
                  type="email"
                  autocomplete="email"
                  placeholder="name@company.com"
                  aria-label="邮箱地址"
                />
              </div>
              <div class="sdp-login__field">
                <label for="sdp-login-email-password">密码</label>
                <t-input
                  id="sdp-login-email-password"
                  v-model="emailForm.password"
                  type="password"
                  autocomplete="current-password"
                  placeholder="请输入密码"
                  aria-label="邮箱登录密码"
                  @enter="handleEmailLogin"
                />
              </div>
              <SdpButton
                class="sdp-login__submit"
                size="lg"
                :loading="loading"
                :disabled="!canSubmitEmail"
                aria-label="使用邮箱登录"
                @click="handleEmailLogin"
              >
                登录
              </SdpButton>
            </form>
          </t-tab-panel>

          <t-tab-panel value="wechat" label="微信扫码" :disabled="true">
            <div class="sdp-login__wechat" role="status">
              <span aria-hidden="true">Phase 2</span>
              <p>微信扫码登录将在 Phase 2 上线</p>
            </div>
          </t-tab-panel>
        </t-tabs>

        <t-message
          v-if="errorMsg"
          class="sdp-login__message"
          theme="error"
          role="alert"
          aria-live="assertive"
        >
          {{ errorMsg }}
        </t-message>

        <div class="sdp-login__agreement">
          <t-checkbox v-model="agreed" aria-label="同意服务协议和隐私政策">
            我已阅读并同意
          </t-checkbox>
          <t-link href="/service-agreement" target="_blank" aria-label="查看服务协议">《服务协议》</t-link>
          <span>和</span>
          <t-link href="/privacy-policy" target="_blank" aria-label="查看隐私政策">《隐私政策》</t-link>
        </div>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '@/api/auth'
import SdpButton from '@/components/design/SdpButton.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const activeTab = ref('phone')
const agreed = ref(false)
const loading = ref(false)
const errorMsg = ref('')
const phoneForm = ref({ phone: '', password: '' })
const emailForm = ref({ email: '', password: '' })

const canSubmitPhone = computed(() => (
  agreed.value && phoneForm.value.phone.length === 11 && Boolean(phoneForm.value.password)
))
const canSubmitEmail = computed(() => (
  agreed.value && Boolean(emailForm.value.email) && Boolean(emailForm.value.password)
))

function loginErrorMessage(error: unknown) {
  if (typeof error === 'object' && error !== null) {
    const requestError = error as {
      message?: string
      response?: { data?: { error?: string; message?: string } }
    }
    return requestError.response?.data?.error
      || requestError.response?.data?.message
      || requestError.message
      || '登录失败，请检查账号密码'
  }
  return '登录失败，请检查账号密码'
}

async function finishLogin(data: Parameters<typeof authStore.setAuth>[0]) {
  authStore.setAuth(data)
  await router.replace({ name: 'spaces' })
}

function onPhoneInput(value: string) {
  phoneForm.value.phone = value.replace(/\D/g, '').slice(0, 11)
}

async function handlePhoneLogin() {
  if (!canSubmitPhone.value || loading.value) return
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await login(phoneForm.value)
    await finishLogin(res.data)
  } catch (error: unknown) {
    errorMsg.value = loginErrorMessage(error)
  } finally {
    loading.value = false
  }
}

async function handleEmailLogin() {
  if (!canSubmitEmail.value || loading.value) return
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await login(emailForm.value)
    await finishLogin(res.data)
  } catch (error: unknown) {
    errorMsg.value = loginErrorMessage(error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.sdp-login {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(30rem, 0.92fr);
  color: var(--ink-900);
  background: var(--ink-50);
  font-family: var(--font-body);
}

.sdp-login__brand {
  display: flex;
  min-height: 100dvh;
  align-items: center;
  padding: var(--space-16);
  color: var(--ink-50);
  background:
    radial-gradient(circle at 82% 14%, var(--brand-600), transparent 34%),
    linear-gradient(145deg, var(--ink-950), var(--brand-900));
}

.sdp-login__brand-content {
  width: 100%;
  max-width: 42rem;
  margin-inline: auto;
}

.sdp-login__identity {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.sdp-login__logo {
  width: var(--space-14);
  height: var(--space-14);
  flex: 0 0 var(--space-14);
}

.sdp-login__logo path:first-child {
  fill: var(--brand-400);
}

.sdp-login__logo path:last-child {
  fill: var(--ink-950);
}

.sdp-login__product {
  margin-bottom: var(--space-1);
  color: var(--brand-200);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-medium);
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.sdp-login__identity h1 {
  font-family: var(--font-display);
  font-size: var(--text-4xl);
  font-weight: var(--font-weight-bold);
  line-height: var(--leading-tight);
}

.sdp-login__statement {
  max-width: 34rem;
  margin-top: var(--space-12);
  color: var(--ink-100);
  font-family: var(--font-display);
  font-size: var(--text-3xl);
  font-weight: var(--font-weight-semibold);
  line-height: var(--leading-relaxed);
}

.sdp-login__values {
  display: grid;
  gap: var(--space-6);
  margin-top: var(--space-12);
  list-style: none;
}

.sdp-login__values li {
  display: grid;
  grid-template-columns: var(--space-10) 1fr;
  gap: var(--space-4);
  align-items: start;
}

.sdp-login__value-icon {
  display: inline-flex;
  width: var(--space-10);
  height: var(--space-10);
  align-items: center;
  justify-content: center;
  border: 1px solid var(--brand-700);
  border-radius: var(--radius-pill);
  color: var(--brand-200);
  background: var(--brand-900);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
}

.sdp-login__values strong {
  display: block;
  margin-bottom: var(--space-1);
  color: var(--ink-50);
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: var(--font-weight-semibold);
}

.sdp-login__values p {
  color: var(--ink-300);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}

.sdp-login__stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-6);
  margin-top: var(--space-12);
  padding-top: var(--space-8);
  border-top: 1px solid var(--ink-700);
}

.sdp-login__stats dt {
  color: var(--brand-300);
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: var(--font-weight-bold);
}

.sdp-login__stats dd {
  margin-top: var(--space-1);
  color: var(--ink-300);
  font-size: var(--text-xs);
}

.sdp-login__form-panel {
  display: flex;
  min-height: 100dvh;
  align-items: center;
  justify-content: center;
  padding: var(--space-12);
}

.sdp-login__form-shell {
  width: 100%;
  max-width: 28rem;
}

.sdp-login__header {
  margin-bottom: var(--space-8);
}

.sdp-login__header p {
  margin-bottom: var(--space-3);
  color: var(--brand-700);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-semibold);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sdp-login__header h2 {
  color: var(--ink-950);
  font-family: var(--font-display);
  font-size: var(--text-4xl);
  font-weight: var(--font-weight-bold);
  line-height: var(--leading-tight);
}

.sdp-login__header span {
  display: block;
  margin-top: var(--space-3);
  color: var(--ink-600);
  font-size: var(--text-sm);
}

.sdp-login__form {
  display: grid;
  gap: var(--space-5);
  padding-top: var(--space-6);
}

.sdp-login__field {
  display: grid;
  gap: var(--space-2);
}

.sdp-login__field label {
  color: var(--ink-800);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-semibold);
}

.sdp-login__submit {
  width: 100%;
  margin-top: var(--space-2);
}

.sdp-login__wechat {
  display: grid;
  min-height: 16rem;
  place-content: center;
  gap: var(--space-3);
  color: var(--ink-600);
  text-align: center;
}

.sdp-login__wechat span {
  color: var(--brand-700);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
}

.sdp-login__message {
  margin-top: var(--space-4);
}

.sdp-login__agreement {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-1);
  margin-top: var(--space-6);
  color: var(--ink-600);
  font-size: var(--text-xs);
  line-height: var(--leading-relaxed);
}

.sdp-login__agreement :deep(.t-link) {
  color: var(--brand-700);
}

.sdp-login__tabs :deep(.t-tabs__nav-item) {
  color: var(--ink-600);
  font-family: var(--font-body);
  font-size: var(--text-sm);
}

.sdp-login__tabs :deep(.t-tabs__nav-item.t-is-active) {
  color: var(--brand-700);
  font-weight: var(--font-weight-semibold);
}

.sdp-login__tabs :deep(.t-tabs__bar) {
  background: var(--brand-600);
}

.sdp-login__field :deep(.t-input) {
  min-height: var(--space-12);
  border-color: var(--ink-300);
  border-radius: var(--radius-sm);
  color: var(--ink-900);
  background: var(--ink-50);
  font-family: var(--font-body);
  font-size: var(--text-sm);
}

.sdp-login__field :deep(.t-input:hover) {
  border-color: var(--ink-500);
}

.sdp-login__field :deep(.t-input:focus-within) {
  border-color: var(--brand-600);
  box-shadow: 0 0 0 2px var(--brand-100);
}

.sdp-login :deep(a:focus-visible),
.sdp-login :deep(input:focus-visible),
.sdp-login :deep([role="tab"]:focus-visible),
.sdp-login :deep(.t-checkbox:focus-within) {
  outline: 2px solid var(--brand-500);
  outline-offset: 2px;
}

@media (max-width: 64rem) {
  .sdp-login {
    grid-template-columns: minmax(0, 0.9fr) minmax(28rem, 1.1fr);
  }

  .sdp-login__brand,
  .sdp-login__form-panel {
    padding: var(--space-8);
  }

  .sdp-login__statement {
    font-size: var(--text-2xl);
  }
}

@media (max-width: 48rem) {
  .sdp-login {
    grid-template-columns: 1fr;
  }

  .sdp-login__brand {
    min-height: auto;
    padding: var(--space-8) var(--space-6);
  }

  .sdp-login__statement,
  .sdp-login__values {
    margin-top: var(--space-8);
  }

  .sdp-login__statement {
    font-size: var(--text-xl);
  }

  .sdp-login__values {
    gap: var(--space-4);
  }

  .sdp-login__stats {
    margin-top: var(--space-8);
    padding-top: var(--space-6);
  }

  .sdp-login__form-panel {
    min-height: auto;
    padding: var(--space-10) var(--space-6);
  }
}

@media (max-width: 30rem) {
  .sdp-login__stats {
    grid-template-columns: 1fr;
    gap: var(--space-4);
  }

  .sdp-login__header h2 {
    font-size: var(--text-3xl);
  }

  .sdp-login__tabs :deep(.t-tabs__nav-item) {
    font-size: var(--text-xs);
  }
}
</style>
