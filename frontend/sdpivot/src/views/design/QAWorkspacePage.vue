<template>
  <SdpSidebarLayout>
    <div class="sdp-qa-workspace">
      <aside class="sdp-qa-workspace__sessions" aria-labelledby="qa-session-title">
        <header class="sdp-qa-workspace__pane-header">
          <div>
            <p>AI Q&amp;A</p>
            <h1 id="qa-session-title">问答会话</h1>
          </div>
          <button
            class="sdp-qa-workspace__new-button"
            type="button"
            aria-label="新建问答会话"
            :disabled="creatingSession"
            @click="createNewSession"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
            {{ creatingSession ? '创建中' : '新会话' }}
          </button>
        </header>

        <label class="sdp-qa-workspace__search" for="qa-session-search">
          <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m16 16 5 5" /></svg>
          <span class="sr-only">搜索问答会话</span>
          <input
            id="qa-session-search"
            v-model="searchQuery"
            type="search"
            placeholder="搜索会话"
            aria-label="搜索问答会话"
          >
        </label>

        <p v-if="sessionError" class="sdp-qa-workspace__error" role="alert">{{ sessionError }}</p>
        <div v-if="loadingSessions" class="sdp-qa-workspace__loading" role="status">正在加载会话...</div>
        <div v-else-if="groupedSessions.length" class="sdp-qa-workspace__session-scroll">
          <section v-for="group in groupedSessions" :key="group.label" class="sdp-qa-workspace__session-group">
            <h2>{{ group.label }}</h2>
            <div role="list" :aria-label="`${group.label}的问答会话`" @keydown="handleListKeydown">
              <button
                v-for="session in group.items"
                :key="session.id"
                class="sdp-qa-workspace__session-item"
                :class="{ 'sdp-qa-workspace__session-item--active': activeSessionId === session.id }"
                type="button"
                :aria-label="`打开会话 ${session.title || '未命名会话'}`"
                :aria-current="activeSessionId === session.id ? 'true' : undefined"
                @click="openSession(session.id)"
              >
                <span class="sdp-qa-workspace__session-mark" aria-hidden="true">
                  <svg viewBox="0 0 24 24"><path d="M5 4h14a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-7l-5 3v-3H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Z" /></svg>
                </span>
                <span>
                  <strong>{{ session.title || '未命名会话' }}</strong>
                  <small>{{ formatTime(session.updated_at) }}</small>
                </span>
              </button>
            </div>
          </section>
        </div>
        <div v-else class="sdp-qa-workspace__empty">
          <strong>{{ searchQuery ? '没有匹配的会话' : '还没有问答会话' }}</strong>
          <span>{{ searchQuery ? '尝试使用其他关键词。' : '创建会话，开始基于知识库提问。' }}</span>
        </div>
      </aside>

      <section class="sdp-qa-workspace__chat" aria-labelledby="qa-chat-title">
        <header class="sdp-qa-workspace__chat-header">
          <div>
            <p>Knowledge Conversation</p>
            <h2 id="qa-chat-title">{{ activeSession?.title || '新的知识问答' }}</h2>
          </div>
          <label class="sdp-qa-workspace__model" for="qa-model-select">
            <span>回答模型</span>
            <select id="qa-model-select" v-model="selectedModelId" aria-label="选择问答模型">
              <option v-if="!models.length" value="">默认模型</option>
              <option v-for="model in models" :key="model.id" :value="model.id">
                {{ model.display_name || model.name }}{{ model.is_default ? ' · 默认' : '' }}
              </option>
            </select>
          </label>
        </header>

        <div ref="messageListRef" class="sdp-qa-workspace__messages" aria-live="polite" :aria-busy="sending">
          <div v-if="!messages.length && !sending" class="sdp-qa-workspace__welcome">
            <span aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M12 3a8 8 0 0 0-8 8c0 2.2.9 4.2 2.4 5.7L5 21l4.5-2.2c.8.2 1.6.3 2.5.3a8 8 0 1 0 0-16.1ZM9 11h.01M12 11h.01M15 11h.01" /></svg></span>
            <h3>从可信知识中获得答案</h3>
            <p>提出业务问题，回答将附带可核查的知识引用。</p>
          </div>

          <article
            v-for="message in messages"
            :key="message.id"
            class="sdp-qa-workspace__message"
            :class="`sdp-qa-workspace__message--${message.role === 'user' ? 'user' : 'assistant'}`"
            :aria-label="message.role === 'user' ? '你的消息' : 'AI 回答'"
          >
            <p class="sdp-qa-workspace__message-role">{{ message.role === 'user' ? 'You' : 'SDPivot·文枢 AI' }}</p>
            <div class="sdp-qa-workspace__bubble">{{ message.content }}</div>

            <div v-if="message.role !== 'user' && parseSources(message.sources).length" class="sdp-qa-workspace__citations" aria-label="回答引用来源">
              <p>引用来源</p>
              <div>
                <article v-for="(source, index) in parseSources(message.sources)" :key="`${message.id}-${index}`">
                  <span>{{ String(index + 1).padStart(2, '0') }}</span>
                  <div>
                    <strong>{{ source.title }}</strong>
                    <small>{{ source.detail }}</small>
                  </div>
                </article>
              </div>
            </div>

            <div v-if="message.role !== 'user'" class="sdp-qa-workspace__feedback" aria-label="评价此回答">
              <span>这个回答有帮助吗？</span>
              <button type="button" :aria-pressed="feedback[message.id] === 'up'" aria-label="赞同此回答" @click="feedback[message.id] = 'up'">
                <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 10v10H4V10h3Zm0 9h10.5a2 2 0 0 0 2-1.7l1-6A2 2 0 0 0 18.5 9H14l.7-3.2A2.3 2.3 0 0 0 12.5 3L7 10v9Z" /></svg>
              </button>
              <button type="button" :aria-pressed="feedback[message.id] === 'down'" aria-label="不赞同此回答" @click="feedback[message.id] = 'down'">
                <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 14V4H4v10h3Zm0-9h10.5a2 2 0 0 1 2 1.7l1 6a2 2 0 0 1-2 2.3H14l.7 3.2a2.3 2.3 0 0 1-2.2 2.8L7 14V5Z" /></svg>
              </button>
            </div>
          </article>

          <article v-if="sending" class="sdp-qa-workspace__message sdp-qa-workspace__message--assistant" role="status">
            <p class="sdp-qa-workspace__message-role">SDPivot·文枢 AI</p>
            <div class="sdp-qa-workspace__bubble sdp-qa-workspace__streaming">
              <span aria-hidden="true" /><span aria-hidden="true" /><span aria-hidden="true" />
              <strong>正在检索知识并生成回答</strong>
            </div>
          </article>
        </div>

        <form class="sdp-qa-workspace__composer" aria-label="发送问答消息" @submit.prevent="submitMessage">
          <div class="sdp-qa-workspace__scope-row">
            <label for="qa-message-input">向知识库提问</label>
            <details ref="scopeMenuRef" class="sdp-qa-workspace__scope">
              <summary>{{ scopeLabel }}</summary>
              <div class="sdp-qa-workspace__scope-menu">
                <label class="sdp-qa-workspace__scope-all">
                  <input type="checkbox" :checked="!selectedSpaceIds.length" @change="selectAllSpaces">
                  <span>全部知识库</span>
                </label>
                <div v-if="spaces.length" class="sdp-qa-workspace__scope-options">
                  <label v-for="space in spaces" :key="space.id">
                    <input v-model="selectedSpaceIds" type="checkbox" :value="space.id">
                    <span>{{ space.name }}</span>
                  </label>
                </div>
                <p v-else>{{ loadingSpaces ? '正在加载空间...' : '暂无可用空间' }}</p>
                <button type="button" @click="closeScopeMenu">完成</button>
              </div>
            </details>
          </div>
          <div>
            <textarea
              id="qa-message-input"
              v-model="inputText"
              rows="2"
              placeholder="输入问题，按 Enter 发送, Shift+Enter 换行"
              aria-label="问答消息内容"
              :disabled="sending"
              @keydown="handleComposerKeydown"
            />
            <button type="submit" aria-label="发送问答消息" :disabled="!inputText.trim() || sending">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 4 17 8-17 8 3-8-3-8Zm3 8h14" /></svg>
            </button>
          </div>
          <p v-if="sendError" role="alert">{{ sendError }}</p>
          <small>AI 生成内容可能存在误差，请核对引用来源。</small>
        </form>
      </section>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import {
  createSession,
  listModels,
  listSessions,
  sendMessage,
  type QAAvailableModel,
  type QAMessage,
  type QASession,
} from '@/api/qa'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'
import { listSpaces, type Space } from '@/api/spaces'

interface Citation {
  title: string
  detail: string
}

const sessions = ref<QASession[]>([])
const messages = ref<QAMessage[]>([])
const models = ref<QAAvailableModel[]>([])
const activeSessionId = ref('')
const selectedModelId = ref('')
const searchQuery = ref('')
const inputText = ref('')
const loadingSessions = ref(true)
const creatingSession = ref(false)
const sending = ref(false)
const sessionError = ref('')
const sendError = ref('')
const feedback = ref<Record<string, 'up' | 'down'>>({})
const messageListRef = ref<HTMLElement | null>(null)
const scopeMenuRef = ref<HTMLDetailsElement | null>(null)
const spaces = ref<Space[]>([])
const loadingSpaces = ref(true)
const sessionSpaceIds = ref<Record<string, string[]>>({})

const activeSession = computed(() => sessions.value.find(session => session.id === activeSessionId.value))
const selectedSpaceIds = computed<string[]>({
  get: () => sessionSpaceIds.value[activeSessionId.value] || [],
  set: value => { sessionSpaceIds.value[activeSessionId.value] = [...value] },
})
const scopeLabel = computed(() => selectedSpaceIds.value.length ? `已选 ${selectedSpaceIds.value.length} 个空间` : '全部知识库')
const filteredSessions = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  return sessions.value.filter(session => !query || session.title.toLocaleLowerCase().includes(query))
})
const groupedSessions = computed(() => {
  const groups = new Map<string, QASession[]>()
  filteredSessions.value.forEach((session) => {
    const label = dateGroup(session.updated_at)
    groups.set(label, [...(groups.get(label) || []), session])
  })
  return Array.from(groups, ([label, items]) => ({ label, items }))
})

function dateGroup(value: string) {
  const date = new Date(value)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const itemDay = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const difference = Math.round((today.getTime() - itemDay.getTime()) / 86400000)
  if (difference === 0) return '今天'
  if (difference === 1) return '昨天'
  if (difference < 7) return '本周'
  return '更早'
}

function formatTime(value: string) {
  const date = new Date(value)
  return dateGroup(value) === '今天'
    ? date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    : date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function parseSources(value: string): Citation[] {
  if (!value) return []
  try {
    const parsed: unknown = JSON.parse(value)
    if (!Array.isArray(parsed)) return []
    return parsed.slice(0, 3).map((source, index) => {
      if (typeof source === 'string') return { title: source, detail: '知识库文档' }
      if (typeof source !== 'object' || source === null) return { title: `知识来源 ${index + 1}`, detail: '知识库文档' }
      const item = source as Record<string, unknown>
      return {
        title: String(item.title || item.name || item.document_name || `知识来源 ${index + 1}`),
        detail: String(item.content || item.snippet || item.page || '知识库文档'),
      }
    })
  } catch {
    return [{ title: '知识引用', detail: value }]
  }
}

function errorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || error === null) return fallback
  const requestError = error as { message?: string; response?: { data?: { error?: string; message?: string } } }
  return requestError.response?.data?.error || requestError.response?.data?.message || requestError.message || fallback
}

async function loadWorkspace() {
  loadingSessions.value = true
  sessionError.value = ''
  const [modelResult, sessionResult, spaceResult] = await Promise.allSettled([listModels(), listSessions(), listSpaces()])
  if (modelResult.status === 'fulfilled') {
    models.value = modelResult.value.data.models || []
    selectedModelId.value = (models.value.find(model => model.is_default) || models.value[0])?.id || ''
  }
  if (sessionResult.status === 'fulfilled') {
    sessions.value = sessionResult.value.data.sessions || []
    activeSessionId.value = sessions.value[0]?.id || ''
  } else {
    sessionError.value = errorMessage(sessionResult.reason, '会话列表加载失败，请稍后重试。')
  }
  if (spaceResult.status === 'fulfilled') spaces.value = spaceResult.value.data.spaces || []
  loadingSpaces.value = false
  loadingSessions.value = false
}

async function createNewSession() {
  if (creatingSession.value) return
  creatingSession.value = true
  sessionError.value = ''
  try {
    const response = await createSession({ title: `新会话 ${new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}` })
    const session = response.data.session
    sessions.value = [session, ...sessions.value.filter(item => item.id !== session.id)]
    activeSessionId.value = session.id
    sessionSpaceIds.value[session.id] = []
    messages.value = []
  } catch (error: unknown) {
    sessionError.value = errorMessage(error, '新建会话失败，请稍后重试。')
  } finally {
    creatingSession.value = false
  }
}

function openSession(id: string) {
  activeSessionId.value = id
  messages.value = []
  sendError.value = ''
}

async function submitMessage() {
  const content = inputText.value.trim()
  if (!content || sending.value) return
  if (!activeSessionId.value) await createNewSession()
  if (!activeSessionId.value) return
  sending.value = true
  sendError.value = ''
  inputText.value = ''
  try {
    const response = await sendMessage(activeSessionId.value, content, selectedModelId.value || undefined, selectedSpaceIds.value)
    messages.value.push(response.data.user_message, response.data.assistant_message)
    if (response.data.model_id) selectedModelId.value = response.data.model_id
    await nextTick()
    messageListRef.value?.scrollTo({ top: messageListRef.value.scrollHeight, behavior: 'smooth' })
  } catch (error: unknown) {
    inputText.value = content
    sendError.value = errorMessage(error, '消息发送失败，请重试。')
  } finally {
    sending.value = false
  }
}

function handleComposerKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
  event.preventDefault()
  void submitMessage()
}

function selectAllSpaces() {
  selectedSpaceIds.value = []
}

function closeScopeMenu() {
  if (scopeMenuRef.value) scopeMenuRef.value.open = false
}

function handleListKeydown(event: KeyboardEvent) {
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  const buttons = Array.from(document.querySelectorAll<HTMLButtonElement>('.sdp-qa-workspace__session-item'))
  const currentIndex = buttons.indexOf(document.activeElement as HTMLButtonElement)
  if (currentIndex < 0 || !buttons.length) return
  event.preventDefault()
  let nextIndex = currentIndex
  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = buttons.length - 1
  if (event.key === 'ArrowDown') nextIndex = (currentIndex + 1) % buttons.length
  if (event.key === 'ArrowUp') nextIndex = (currentIndex - 1 + buttons.length) % buttons.length
  buttons[nextIndex]?.focus()
}

onMounted(loadWorkspace)
</script>

<style scoped>
.sdp-qa-workspace { min-height: 100dvh; display: grid; grid-template-columns: var(--session-pane-width) minmax(var(--space-0), 1fr); color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-qa-workspace__sessions { height: 100dvh; display: flex; flex-direction: column; gap: var(--space-5); padding: var(--space-6) var(--space-4); overflow: hidden; border-right: 1px solid var(--ink-200); background: var(--ink-50); }
.sdp-qa-workspace__pane-header, .sdp-qa-workspace__chat-header, .sdp-qa-workspace__new-button, .sdp-qa-workspace__search, .sdp-qa-workspace__session-item, .sdp-qa-workspace__model, .sdp-qa-workspace__feedback, .sdp-qa-workspace__composer > div { display: flex; align-items: center; }
.sdp-qa-workspace__pane-header, .sdp-qa-workspace__chat-header { justify-content: space-between; gap: var(--space-4); }
.sdp-qa-workspace__pane-header p, .sdp-qa-workspace__chat-header p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-qa-workspace__pane-header h1, .sdp-qa-workspace__chat-header h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-weight: var(--font-weight-bold); line-height: var(--leading-tight); }
.sdp-qa-workspace__pane-header h1 { font-size: var(--text-xl); }
.sdp-qa-workspace__chat-header h2 { font-size: var(--text-2xl); }
.sdp-qa-workspace__new-button { min-height: var(--space-10); gap: var(--space-2); padding: var(--space-2) var(--space-3); border: 1px solid var(--brand-600); border-radius: var(--radius-sm); color: var(--ink-950); background: var(--brand-500); font: var(--font-weight-semibold) var(--text-sm)/var(--leading-tight) var(--font-body); cursor: pointer; }
.sdp-qa-workspace__new-button svg, .sdp-qa-workspace__search svg, .sdp-qa-workspace__session-mark svg, .sdp-qa-workspace__feedback svg, .sdp-qa-workspace__composer button svg, .sdp-qa-workspace__welcome svg { width: var(--space-5); height: var(--space-5); fill: none; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 1.8; }
.sdp-qa-workspace__search { min-height: var(--space-10); gap: var(--space-2); padding-inline: var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-600); background: var(--ink-50); }
.sdp-qa-workspace__search input { width: 100%; min-height: var(--space-8); border: 0; outline: 0; color: var(--ink-900); background: transparent; font: var(--text-sm) var(--font-body); }
.sdp-qa-workspace__search input::placeholder, .sdp-qa-workspace__composer textarea::placeholder { color: var(--ink-500); }
.sdp-qa-workspace__search:focus-within, .sdp-qa-workspace__composer > div:focus-within { border-color: var(--brand-600); outline: 2px solid var(--brand-100); outline-offset: 2px; }
.sdp-qa-workspace__session-scroll { min-height: var(--space-0); flex: 1; overflow-y: auto; }
.sdp-qa-workspace__session-group + .sdp-qa-workspace__session-group { margin-top: var(--space-5); }
.sdp-qa-workspace__session-group h2 { padding-inline: var(--space-2); color: var(--ink-500); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-qa-workspace__session-group > div { display: grid; gap: var(--space-1); margin-top: var(--space-2); }
.sdp-qa-workspace__session-item { width: 100%; min-width: var(--space-0); gap: var(--space-3); padding: var(--space-3); border: 1px solid transparent; border-radius: var(--radius-sm); color: var(--ink-800); background: transparent; text-align: left; cursor: pointer; }
.sdp-qa-workspace__session-item:hover { background: var(--ink-100); }
.sdp-qa-workspace__session-item--active { border-color: var(--brand-200); color: var(--brand-900); background: var(--brand-50); }
.sdp-qa-workspace__session-mark { width: var(--space-8); height: var(--space-8); flex: 0 0 var(--space-8); display: grid; place-items: center; border-radius: var(--radius-sm); color: var(--brand-800); background: var(--brand-100); }
.sdp-qa-workspace__session-mark svg { width: var(--space-4); height: var(--space-4); }
.sdp-qa-workspace__session-item > span:last-child { min-width: var(--space-0); display: flex; flex-direction: column; }
.sdp-qa-workspace__session-item strong { overflow: hidden; font-size: var(--text-sm); font-weight: var(--font-weight-semibold); text-overflow: ellipsis; white-space: nowrap; }
.sdp-qa-workspace__session-item small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-qa-workspace__error { padding: var(--space-3); border: 1px solid var(--brand-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--brand-50); font-size: var(--text-xs); }
.sdp-qa-workspace__loading, .sdp-qa-workspace__empty { padding: var(--space-6) var(--space-3); color: var(--ink-600); font-size: var(--text-sm); text-align: center; }
.sdp-qa-workspace__empty { display: flex; flex-direction: column; gap: var(--space-2); }
.sdp-qa-workspace__empty strong { color: var(--ink-800); }
.sdp-qa-workspace__chat { height: 100dvh; min-width: var(--space-0); display: grid; grid-template-rows: auto minmax(var(--space-0), 1fr) auto; background: var(--ink-100); }
.sdp-qa-workspace__chat-header { min-height: var(--header-height); padding: var(--space-4) var(--space-8); border-bottom: 1px solid var(--ink-200); background: var(--ink-50); }
.sdp-qa-workspace__model { gap: var(--space-3); color: var(--ink-600); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-qa-workspace__model select { min-height: var(--space-10); padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font: var(--text-sm) var(--font-body); }
.sdp-qa-workspace__messages { min-height: var(--space-0); display: flex; flex-direction: column; gap: var(--space-6); overflow-y: auto; padding: var(--space-8) max(var(--space-8), calc((100% - 52rem) / 2)); }
.sdp-qa-workspace__welcome { margin: auto; max-width: calc(var(--space-24) * 5); text-align: center; }
.sdp-qa-workspace__welcome > span { width: var(--space-16); height: var(--space-16); display: grid; place-items: center; margin-inline: auto; border-radius: var(--radius-lg); color: var(--brand-800); background: var(--brand-100); }
.sdp-qa-workspace__welcome svg { width: var(--space-8); height: var(--space-8); }
.sdp-qa-workspace__welcome h3 { margin-top: var(--space-5); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-2xl); }
.sdp-qa-workspace__welcome p { margin-top: var(--space-2); color: var(--ink-600); line-height: var(--leading-relaxed); }
.sdp-qa-workspace__message { max-width: 84%; }
.sdp-qa-workspace__message--user { align-self: flex-end; }
.sdp-qa-workspace__message--assistant { align-self: flex-start; }
.sdp-qa-workspace__message-role { margin-bottom: var(--space-2); color: var(--ink-500); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-qa-workspace__message--user .sdp-qa-workspace__message-role { text-align: right; }
.sdp-qa-workspace__bubble { padding: var(--space-4) var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-lg); color: var(--ink-900); background: var(--ink-50); box-shadow: var(--shadow-xs); font-size: var(--text-sm); line-height: var(--leading-relaxed); white-space: pre-wrap; }
.sdp-qa-workspace__message--user .sdp-qa-workspace__bubble { border-color: var(--ink-950); border-bottom-right-radius: var(--radius-xs); color: var(--ink-50); background: var(--ink-950); }
.sdp-qa-workspace__message--assistant .sdp-qa-workspace__bubble { border-bottom-left-radius: var(--radius-xs); }
.sdp-qa-workspace__citations { margin-top: var(--space-3); padding: var(--space-4); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); }
.sdp-qa-workspace__citations > p { color: var(--ink-600); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-qa-workspace__citations > div { display: grid; grid-template-columns: repeat(3, minmax(var(--space-0), 1fr)); gap: var(--space-2); margin-top: var(--space-3); }
.sdp-qa-workspace__citations article { min-width: var(--space-0); display: flex; gap: var(--space-2); padding: var(--space-3); border-radius: var(--radius-sm); background: var(--ink-100); }
.sdp-qa-workspace__citations article > span { color: var(--brand-800); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-bold); }
.sdp-qa-workspace__citations article div { min-width: var(--space-0); display: flex; flex-direction: column; }
.sdp-qa-workspace__citations strong, .sdp-qa-workspace__citations small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sdp-qa-workspace__citations strong { color: var(--ink-800); font-size: var(--text-xs); }
.sdp-qa-workspace__citations small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-qa-workspace__feedback { gap: var(--space-2); margin-top: var(--space-3); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-qa-workspace__feedback button { width: var(--space-8); height: var(--space-8); display: grid; place-items: center; border: 1px solid var(--ink-300); border-radius: var(--radius-pill); color: var(--ink-600); background: var(--ink-50); cursor: pointer; }
.sdp-qa-workspace__feedback button[aria-pressed="true"] { border-color: var(--brand-500); color: var(--brand-900); background: var(--brand-100); }
.sdp-qa-workspace__feedback svg { width: var(--space-4); height: var(--space-4); }
.sdp-qa-workspace__streaming { display: flex; align-items: center; gap: var(--space-2); color: var(--ink-600); }
.sdp-qa-workspace__streaming > span { width: var(--space-2); height: var(--space-2); border-radius: var(--radius-pill); background: var(--brand-600); animation: sdp-qa-pulse 1s var(--ease-in-out) infinite alternate; }
.sdp-qa-workspace__streaming > span:nth-child(2) { animation-delay: var(--duration-fast); }
.sdp-qa-workspace__streaming > span:nth-child(3) { animation-delay: var(--duration-normal); }
.sdp-qa-workspace__streaming strong { margin-left: var(--space-1); font-size: var(--text-sm); }
.sdp-qa-workspace__composer { padding: var(--space-4) max(var(--space-8), calc((100% - 52rem) / 2)) var(--space-5); border-top: 1px solid var(--ink-200); background: var(--ink-50); }
.sdp-qa-workspace__scope-row { display: flex; align-items: center; justify-content: space-between; gap: var(--space-3); margin-bottom: var(--space-2); }
.sdp-qa-workspace__scope-row > label { color: var(--ink-800); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-qa-workspace__scope { position: relative; }
.sdp-qa-workspace__scope summary { min-height: var(--space-9); display: flex; align-items: center; padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-700); background: var(--ink-50); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); cursor: pointer; list-style: none; }
.sdp-qa-workspace__scope summary::after { margin-left: var(--space-2); content: '▾'; color: var(--brand-700); }
.sdp-qa-workspace__scope summary::-webkit-details-marker { display: none; }
.sdp-qa-workspace__scope-menu { position: absolute; right: 0; bottom: calc(100% + var(--space-2)); z-index: 10; width: 280px; max-height: 320px; display: grid; gap: var(--space-2); padding: var(--space-3); overflow: auto; border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); box-shadow: var(--shadow-lg); }
.sdp-qa-workspace__scope-menu label { display: flex; align-items: center; gap: var(--space-2); min-height: var(--space-9); padding: var(--space-2); border-radius: var(--radius-sm); color: var(--ink-800); font-size: var(--text-sm); cursor: pointer; }
.sdp-qa-workspace__scope-menu label:hover { background: var(--brand-50); }
.sdp-qa-workspace__scope-menu input { accent-color: var(--brand-700); }
.sdp-qa-workspace__scope-all { border-bottom: 1px solid var(--ink-200); }
.sdp-qa-workspace__scope-options { display: grid; }
.sdp-qa-workspace__scope-menu p { padding: var(--space-3); color: var(--ink-500); font-size: var(--text-xs); text-align: center; }
.sdp-qa-workspace__scope-menu > button { min-height: var(--space-9); border: 1px solid var(--brand-600); border-radius: var(--radius-sm); color: var(--ink-950); background: var(--brand-500); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-qa-workspace__composer > div { gap: var(--space-3); padding: var(--space-2); border: 1px solid var(--ink-300); border-radius: var(--radius-md); background: var(--ink-50); }
.sdp-qa-workspace__composer textarea { min-height: var(--space-12); flex: 1; resize: none; border: 0; outline: 0; color: var(--ink-900); background: transparent; font: var(--text-sm)/var(--leading-normal) var(--font-body); }
.sdp-qa-workspace__composer button { width: var(--space-12); height: var(--space-12); flex: 0 0 var(--space-12); display: grid; place-items: center; border: 1px solid var(--brand-600); border-radius: var(--radius-sm); color: var(--ink-950); background: var(--brand-500); cursor: pointer; }
.sdp-qa-workspace__composer button:disabled, .sdp-qa-workspace__new-button:disabled { cursor: not-allowed; opacity: .6; }
.sdp-qa-workspace__composer > p { margin-top: var(--space-2); color: var(--brand-900); font-size: var(--text-xs); }
.sdp-qa-workspace__composer > small { display: block; margin-top: var(--space-2); color: var(--ink-500); font-size: var(--text-xs); text-align: center; }
.sdp-qa-workspace button:focus-visible, .sdp-qa-workspace select:focus-visible, .sdp-qa-workspace textarea:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
@keyframes sdp-qa-pulse { to { opacity: .25; transform: translateY(calc(var(--space-1) * -1)); } }
@media (max-width: 64rem) { .sdp-qa-workspace { grid-template-columns: var(--session-pane-width) minmax(var(--space-0), 1fr); } .sdp-qa-workspace__citations > div { grid-template-columns: 1fr; } .sdp-qa-workspace__chat-header { align-items: flex-start; flex-direction: column; } }
@media (max-width: 48rem) { .sdp-qa-workspace { min-height: calc(100dvh - var(--header-height)); grid-template-columns: 1fr; } .sdp-qa-workspace__sessions { height: auto; max-height: calc(var(--space-24) * 3); border-right: 0; border-bottom: 1px solid var(--ink-200); } .sdp-qa-workspace__chat { height: auto; min-height: calc(100dvh - var(--header-height)); } .sdp-qa-workspace__messages, .sdp-qa-workspace__composer { padding-inline: var(--space-4); } .sdp-qa-workspace__message { max-width: 94%; } .sdp-qa-workspace__model { width: 100%; justify-content: space-between; } }
</style>
