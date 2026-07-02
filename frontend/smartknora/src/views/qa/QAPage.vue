<template>
  <div class="qa-page">
    <div class="qa-sidebar">
      <div class="sidebar-header">
        <h3>AI 问答</h3>
        <t-button theme="primary" size="small" @click="handleNewSession">
          <template #icon><t-icon name="add" /></template>
          新对话
        </t-button>
      </div>
      <div class="session-list">
        <div
          v-for="s in sessions"
          :key="s.id"
          :class="['session-item', { active: s.id === currentSessionId }]"
          @click="switchSession(s.id)"
        >
          <t-icon name="chat" size="16px" />
          <span class="session-title">{{ s.title }}</span>
        </div>
        <div v-if="sessions.length === 0" class="no-sessions">暂无历史对话</div>
      </div>
    </div>

    <div class="qa-main">
      <div v-if="!currentSessionId" class="qa-welcome">
        <t-icon name="chat" size="64px" style="color:#d0d0d0" />
        <h2>开始 AI 问答</h2>
        <p>选择知识空间，创建新对话，向知识库提问</p>
        <t-button theme="primary" size="large" @click="handleNewSession">
          <template #icon><t-icon name="add" /></template>
          新建对话
        </t-button>
      </div>

      <template v-else>
        <div class="messages-area" ref="msgArea">
          <div v-for="msg in messages" :key="msg.id" :class="['msg-row', msg.role]">
            <div class="msg-avatar">
              <t-icon :name="msg.role === 'user' ? 'user' : 'robot'" size="24px" />
            </div>
            <div class="msg-bubble">
              <div class="msg-content">{{ msg.content }}</div>
              <div v-if="msg.role === 'assistant' && msg.sources && msg.sources !== '[]'" class="msg-sources">
                <t-tag theme="primary" variant="light" size="small">📎 引用来源</t-tag>
              </div>
            </div>
          </div>
          <div v-if="sending" class="msg-row assistant">
            <div class="msg-avatar"><t-icon name="robot" size="24px" /></div>
            <div class="msg-bubble"><t-loading size="small" /></div>
          </div>
        </div>

        <div class="input-area">
          <t-textarea
            v-model="inputText"
            placeholder="输入您的问题..."
            :autosize="{ minRows: 1, maxRows: 4 }"
            @enter="handleSend"
            :disabled="sending"
          />
          <t-button theme="primary" :loading="sending" :disabled="!inputText.trim()" @click="handleSend">
            <template #icon><t-icon name="send" /></template>
            发送
          </t-button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { listSessions, createSession, getMessages, sendMessage, type QASession, type QAMessage } from '@/api/qa'
import { MessagePlugin } from 'tdesign-vue-next'

const sessions = ref<QASession[]>([])
const currentSessionId = ref('')
const messages = ref<QAMessage[]>([])
const inputText = ref('')
const sending = ref(false)
const msgArea = ref<HTMLElement>()

async function loadSessions() {
  try {
    const res = await listSessions()
    sessions.value = (res.data as any).sessions || []
  } catch { sessions.value = [] }
}

async function handleNewSession() {
  try {
    const res = await createSession({ title: '新对话' })
    const session = (res.data as any).session
    sessions.value.unshift(session)
    currentSessionId.value = session.id
    messages.value = []
  } catch { MessagePlugin.error('创建会话失败') }
}

async function switchSession(id: string) {
  currentSessionId.value = id
  try {
    const res = await getMessages(id)
    messages.value = (res.data as any).messages || []
    scrollToBottom()
  } catch { messages.value = [] }
}

async function handleSend() {
  if (!inputText.value.trim() || !currentSessionId.value) return
  const text = inputText.value.trim()
  inputText.value = ''
  sending.value = true

  // 立即显示用户消息
  messages.value.push({
    id: 'temp-' + Date.now(),
    session_id: currentSessionId.value,
    role: 'user',
    content: text,
    sources: '[]',
    created_at: new Date().toISOString(),
  })
  scrollToBottom()

  try {
    const res = await sendMessage(currentSessionId.value, text)
    const data = res.data as any
    messages.value.push(data.assistant_message)
    scrollToBottom()
  } catch {
    MessagePlugin.error('发送失败')
  } finally {
    sending.value = false
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (msgArea.value) msgArea.value.scrollTop = msgArea.value.scrollHeight
  })
}

onMounted(loadSessions)
</script>

<style scoped>
.qa-page { display: flex; height: calc(100vh - 104px); background: #fff; border-radius: 8px; overflow: hidden; }
.qa-sidebar { width: 260px; border-right: 1px solid #e7e7e7; display: flex; flex-direction: column; }
.sidebar-header { display: flex; justify-content: space-between; align-items: center; padding: 16px; border-bottom: 1px solid #f0f0f0; }
.sidebar-header h3 { font-size: 16px; font-weight: 600; }
.session-list { flex: 1; overflow-y: auto; padding: 8px; }
.session-item { display: flex; align-items: center; gap: 8px; padding: 10px 12px; border-radius: 6px; cursor: pointer; font-size: 14px; color: #333; margin-bottom: 2px; }
.session-item:hover { background: #f5f5f5; }
.session-item.active { background: #e8f0ff; color: #014DB2; }
.session-title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.no-sessions { text-align: center; padding: 24px; color: #ccc; font-size: 13px; }
.qa-main { flex: 1; display: flex; flex-direction: column; }
.qa-welcome { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: #999; }
.qa-welcome h2 { font-size: 22px; color: #333; }
.messages-area { flex: 1; overflow-y: auto; padding: 24px; }
.msg-row { display: flex; gap: 12px; margin-bottom: 20px; }
.msg-avatar { width: 36px; height: 36px; border-radius: 50%; display: flex; align-items: center; justify-content: center; flex-shrink: 0; background: #f0f2f5; color: #666; }
.msg-row.user .msg-avatar { background: #014DB2; color: #fff; }
.msg-bubble { max-width: 70%; padding: 12px 16px; border-radius: 12px; font-size: 14px; line-height: 1.6; }
.msg-row.user .msg-bubble { background: #014DB2; color: #fff; border-bottom-right-radius: 4px; }
.msg-row.assistant .msg-bubble { background: #f5f5f5; color: #333; border-bottom-left-radius: 4px; }
.msg-content { white-space: pre-wrap; }
.msg-sources { margin-top: 8px; }
.input-area { display: flex; gap: 12px; padding: 16px 24px; border-top: 1px solid #e7e7e7; align-items: flex-end; }
.input-area :deep(.t-textarea) { flex: 1; }
</style>
