import { useEffect, useState, useRef } from 'react'
import { Card, Button, Input, List, Tag, Avatar, MessagePlugin } from 'tdesign-react'
import client from '../../api/client'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  created_at: string
}

interface Session {
  id: string
  title: string
  updated_at: string
}

export default function QAPage() {
  const [sessions, setSessions] = useState<Session[]>([])
  const [currentSession, setCurrentSession] = useState<string | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    client.get('/qa/sessions').then(res => setSessions(res.data.sessions || [])).catch((err) => { console.error('API error:', err) })
  }, [])

  useEffect(() => {
    if (currentSession) {
      client.get(`/qa/sessions/${currentSession}/messages`)
        .then(res => setMessages(res.data.messages || []))
        .catch((err) => { console.error('API error:', err) })
    }
  }, [currentSession])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleNewSession = async () => {
    try {
      const res = await client.post('/qa/sessions', { title: '新对话' })
      const session = res.data.session
      setSessions(prev => [session, ...prev])
      setCurrentSession(session.id)
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '创建会话失败')
    }
  }

  const handleSend = async () => {
    if (!input.trim() || !currentSession || sending) return
    setSending(true)
    const content = input
    setInput('')

    try {
      const res = await client.post(`/qa/sessions/${currentSession}/messages`, { content })
      setMessages(prev => [...prev, res.data.user_message, res.data.assistant_message])
    } catch (err: any) {
      // Show error as system message
      setMessages(prev => [...prev, {
        id: Date.now().toString(),
        role: 'assistant',
        content: '抱歉，发生了错误：' + (err?.response?.data?.error || '未知错误'),
        created_at: new Date().toISOString()
      }])
    } finally {
      setSending(false)
    }
  }

  return (
    <div style={{ display: 'flex', height: '100%' }}>
      {/* Session sidebar */}
      <div style={{ width: 280, borderRight: '1px solid #e8e8e8', background: '#fafafa', display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: 16, borderBottom: '1px solid #e8e8e8' }}>
          <Button theme="primary" block onClick={handleNewSession}>+ 新对话</Button>
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          {sessions.map(s => (
            <div
              key={s.id}
              onClick={() => setCurrentSession(s.id)}
              style={{
                padding: '12px 16px',
                cursor: 'pointer',
                background: currentSession === s.id ? '#e8f4ff' : 'transparent',
                borderBottom: '1px solid #f0f0f0',
              }}
            >
              <div style={{ fontSize: 14, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {s.title}
              </div>
              <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                {new Date(s.updated_at).toLocaleDateString()}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Chat area */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Header */}
        <div style={{ padding: '16px 24px', borderBottom: '1px solid #e8e8e8', background: '#fff' }}>
          <h3 style={{ margin: 0 }}>🤖 AI 问答</h3>
        </div>

        {/* Messages */}
        <div style={{ flex: 1, overflow: 'auto', padding: 24, background: '#f8f9fa' }}>
          {messages.length === 0 ? (
            <div style={{ textAlign: 'center', color: '#999', paddingTop: 100 }}>
              <p style={{ fontSize: 48 }}>🤖</p>
              <p>开始提问，AI 将基于知识库回答</p>
            </div>
          ) : (
            messages.map(msg => (
              <div key={msg.id} style={{
                display: 'flex',
                justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
                marginBottom: 16,
              }}>
                <div style={{
                  maxWidth: '70%',
                  padding: '12px 16px',
                  borderRadius: 12,
                  background: msg.role === 'user' ? '#0052D9' : '#fff',
                  color: msg.role === 'user' ? '#fff' : '#333',
                  boxShadow: msg.role === 'assistant' ? '0 1px 3px rgba(0,0,0,0.1)' : 'none',
                }}>
                  {msg.content}
                </div>
              </div>
            ))
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* Input */}
        <div style={{ padding: 16, borderTop: '1px solid #e8e8e8', background: '#fff', display: 'flex', gap: 12 }}>
          <Input
            placeholder="输入问题..."
            value={input}
            onChange={setInput}
            onPressEnter={handleSend}
            disabled={!currentSession || sending}
            size="large"
          />
          <Button
            theme="primary"
            onClick={handleSend}
            loading={sending}
            disabled={!currentSession || !input.trim()}
            size="large"
          >
            发送
          </Button>
        </div>
      </div>
    </div>
  )
}
