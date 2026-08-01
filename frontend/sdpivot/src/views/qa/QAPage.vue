<template>
  <div class="qa-page page-section-card">
    <aside class="qa-sidebar">
      <div class="sidebar-head">
        <p class="sidebar-tag">AI Q&A</p>
        <h2>问答工作台</h2>
        <p>管理会话、沉淀上下文，并持续追踪知识引用来源。</p>
      </div>

      <t-button theme="primary" block size="large" @click="createNewSession">
        <template #icon><t-icon name="add" /></template>
        新建会话
      </t-button>

      <div class="session-list">
        <template v-if="sessions.length">
          <button
            v-for="session in sessions"
            :key="session.id"
            type="button"
            class="session-item"
            :class="{ active: activeSessionId === session.id }"
            @click="openSession(session.id)"
          >
            <div class="session-icon"><t-icon name="chat" /></div>
            <div class="session-copy">
              <strong>{{ session.title || '未命名会话' }}</strong>
              <span>{{ session.updated_at ? formatDate(session.updated_at) : '刚刚创建' }}</span>
            </div>
          </button>
        </template>
        <div v-else class="no-sessions">还没有会话，先开启一次提问吧。</div>
      </div>
    </aside>

    <section class="qa-main page-section-card">
      <header class="qa-header" v-if="activeSessionId">
        <div>
          <span class="qa-header-tag">Current session</span>
          <h3>{{ activeSessionTitle || '当前会话' }}</h3>
        </div>
        <t-space>
          <t-tag theme="success" variant="light">{{ selectedModelName || '知识问答' }}</t-tag>
          <t-button variant="outline" @click="createNewSession">新会话</t-button>
        </t-space>
      </header>

      <div v-if="!activeSessionId" class="qa-welcome">
        <div class="welcome-mark"><t-icon name="chat-bubble-smile" size="30px" /></div>
        <h3>开始一场高质量问答</h3>
        <p>从知识空间中抽取上下文，让回答更贴近业务语境，并可按需选择当前租户已配置的回答模型。</p>
        <t-button theme="primary" size="large" @click="createNewSession">立即开始</t-button>
      </div>

      <template v-else>
        <Suspense>
          <QAMessagesPanel :messages="messages" :sending="sending" />
          <template #fallback>
            <div class="qa-panel-loading">
              <t-loading size="small" text="正在加载消息区..." />
            </div>
          </template>
        </Suspense>

        <Suspense>
          <QAInputPanel
            v-model="inputText"
            v-model:model-id="selectedModelId"
            :models="availableModels"
            :sending="sending"
            @submit="submitMessage"
          />
          <template #fallback>
            <div class="qa-panel-loading input-loading">
              <t-loading size="small" text="正在加载输入区..." />
            </div>
          </template>
        </Suspense>
      </template>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref } from 'vue'
import {
  listSessions,
  createSession,
  getMessages,
  sendMessage,
  listQAModels,
  type QAAvailableModel,
  type QAMessage,
  type QASession,
} from '@/api/qa'
import { MessagePlugin } from 'tdesign-vue-next'

const QAMessagesPanel = defineAsyncComponent(() => import('@/components/qa/QAMessagesPanel.vue'))
const QAInputPanel = defineAsyncComponent(() => import('@/components/qa/QAInputPanel.vue'))

const sessions = ref<QASession[]>([])
const messages = ref<QAMessage[]>([])
const activeSessionId = ref('')
const inputText = ref('')
const sending = ref(false)
const availableModels = ref<QAAvailableModel[]>([])
const selectedModelId = ref('')

const selectedModelName = computed(() => {
  const model = availableModels.value.find(item => item.id === selectedModelId.value)
  return model?.display_name || model?.name || ''
})

const activeSessionTitle = computed(() => sessions.value.find(item => String(item.id) === String(activeSessionId.value))?.title || '')

function formatDate(value: string) {
  return value ? new Date(value).toLocaleDateString('zh-CN') : ''
}

async function loadModels() {
  try {
    const res = await listQAModels()
    availableModels.value = res.data.models || []
    const preferred = availableModels.value.find(item => item.is_default) || availableModels.value[0]
    if (!availableModels.value.some(item => item.id === selectedModelId.value)) {
      selectedModelId.value = preferred?.id || ''
    }
  } catch {
    availableModels.value = []
    selectedModelId.value = ''
  }
}

async function loadSessions() {
  try {
    const res = await listSessions()
    sessions.value = res.data.sessions || []
    if (!activeSessionId.value && sessions.value.length) {
      activeSessionId.value = sessions.value[0].id
      await loadMessages(activeSessionId.value)
    }
  } catch {
    sessions.value = []
  }
}

async function loadMessages(sessionId: string) {
  const requestedSessionId = sessionId
  try {
    const res = await getMessages(requestedSessionId)
    if (activeSessionId.value !== requestedSessionId) return
    messages.value = res.data.messages || []
  } catch {
    if (activeSessionId.value !== requestedSessionId) return
    messages.value = []
  }
}

async function createNewSession() {
  try {
    const res = await createSession({ title: `新会话 ${new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}` })
    const session = res.data.session
    if (session?.id) {
      activeSessionId.value = session.id
      await loadSessions()
      messages.value = []
    }
  } catch {
    MessagePlugin.error('创建会话失败')
  }
}

async function openSession(sessionId: string) {
  activeSessionId.value = sessionId
  await loadMessages(sessionId)
}

async function submitMessage() {
  if (!inputText.value.trim()) return
  if (!activeSessionId.value) {
    await createNewSession()
  }
  const content = inputText.value.trim()
  inputText.value = ''
  sending.value = true
  try {
    const res = await sendMessage(activeSessionId.value, content, selectedModelId.value)
    if (res.data.model_id) selectedModelId.value = res.data.model_id
    await loadMessages(activeSessionId.value)
    await loadSessions()
  } catch {
    MessagePlugin.error('发送失败')
    inputText.value = content
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadModels(), loadSessions()])
})
</script>

<style scoped>
.qa-page {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 18px;
  padding: 18px;
  background: transparent;
}

.qa-sidebar {
  padding: 20px;
  border-radius: 12px;
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-elevated) 92%, transparent) 0%, color-mix(in srgb, var(--sdp-surface-soft) 86%, transparent) 100%);
  border: 1px solid var(--border-soft);
}

.sidebar-head h2 {
  margin: 8px 0 0;
  font-size: 24px;
  color: var(--text-primary);
}

.sidebar-head p {
  margin: 8px 0 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.sidebar-tag {
  display: inline-block;
  font-size: 12px;
  color: var(--brand-primary);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.session-list {
  margin-top: 20px;
}

.session-item {
  width: 100%;
  display: flex;
  gap: 12px;
  padding: 12px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  transition: background 0.24s ease, transform 0.24s cubic-bezier(0.16, 1, 0.3, 1);
}

.session-item + .session-item {
  margin-top: 8px;
}

.session-item:hover {
  background: rgba(0, 185, 107, 0.08);
}

.session-item.active {
  background: rgba(0, 185, 107, 0.14);
  transform: translateX(2px);
}

.session-icon {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: color-mix(in srgb, var(--surface-elevated) 92%, transparent);
  color: var(--brand-primary);
  border: 1px solid var(--border-soft);
}

.session-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.session-copy strong {
  font-size: 14px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session-copy span {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.no-sessions {
  padding: 18px 10px;
  color: var(--text-secondary);
  font-size: 13px;
}

.qa-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-elevated) 92%, transparent) 0%, color-mix(in srgb, var(--sdp-surface-soft) 88%, transparent) 100%);
}

.qa-header {
  padding: 22px 24px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  border-bottom: 1px solid color-mix(in srgb, var(--border-soft) 88%, transparent);
}

.qa-header-tag {
  display: inline-block;
  font-size: 12px;
  color: var(--text-secondary);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.qa-header h3 {
  margin: 8px 0 0;
  font-size: 24px;
  color: var(--text-primary);
}

.qa-welcome {
  flex: 1;
  padding: 28px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.welcome-mark {
  width: 72px;
  height: 72px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: var(--sdp-brand-soft);
  color: var(--brand-primary);
}

.qa-welcome h3 {
  margin: 20px 0 0;
  font-size: 28px;
  color: var(--text-primary);
}

.qa-welcome p {
  max-width: 520px;
  margin: 12px 0 24px;
  color: var(--text-secondary);
}

.qa-panel-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 160px;
  color: var(--text-secondary);
}

.input-loading {
  min-height: 120px;
}

@media (max-width: 1080px) {
  .qa-page {
    grid-template-columns: 1fr;
  }

  .qa-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
