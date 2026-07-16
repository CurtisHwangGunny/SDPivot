<template>
  <div class="messages-area">
    <div v-for="msg in messages" :key="msg.id" class="msg-row" :class="msg.role">
      <div class="msg-avatar">
        <t-icon :name="msg.role === 'user' ? 'user' : 'chat-bubble-ai'" />
      </div>
      <div class="msg-bubble">
        <div class="msg-role">{{ msg.role === 'user' ? '用户' : 'AI 助手' }}</div>
        <div class="msg-content">{{ msg.content }}</div>
        <div v-if="msg.sources" class="msg-sources">
          <t-tag variant="light" size="small" class="source-tag">
            {{ msg.sources }}
          </t-tag>
        </div>
      </div>
    </div>
    <div v-if="sending" class="msg-row pending">
      <div class="msg-avatar"><t-icon name="chat-bubble-ai" /></div>
      <div class="msg-bubble">
        <t-loading size="small" text="正在生成回答..." />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { QAMessage } from '@/api/qa'

defineProps<{
  messages: QAMessage[]
  sending: boolean
}>()
</script>

<style scoped>
.messages-area {
  flex: 1;
  padding: 24px 28px 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
  overflow-y: auto;
}

.msg-row {
  display: flex;
  gap: 14px;
  align-items: flex-start;
}

.msg-row.user {
  flex-direction: row-reverse;
}

.msg-avatar {
  width: 40px;
  height: 40px;
  border-radius: 14px;
  display: grid;
  place-items: center;
  background: color-mix(in srgb, var(--surface-elevated) 92%, transparent);
  color: var(--brand-primary);
  border: 1px solid var(--border-soft);
  flex-shrink: 0;
}

.msg-bubble {
  max-width: min(760px, 100%);
  padding: 16px 18px;
  border-radius: 22px;
  background: color-mix(in srgb, var(--surface-elevated) 94%, transparent);
  border: 1px solid var(--border-soft);
}

.msg-row.user .msg-bubble {
  background: color-mix(in srgb, var(--sk-brand-soft) 88%, transparent);
}

.msg-role {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.msg-content {
  white-space: pre-wrap;
  color: var(--text-primary);
  line-height: 1.75;
}

.msg-sources {
  margin-top: 12px;
}

.source-tag {
  max-width: 100%;
}
</style>
