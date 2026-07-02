import { useState, useEffect } from 'react'
import { Card, Button, Input, Select, MessagePlugin, Tabs } from 'tdesign-react'
import client from '../../api/client'

const CATEGORIES = [
  { value: 'work_summary', label: '📝 工作总结' },
  { value: 'research_report', label: '📊 研究报告' },
  { value: 'project_plan', label: '📋 项目方案' },
  { value: 'meeting_notes', label: '📓 会议纪要' },
  { value: 'tech_doc', label: '📘 技术文档' },
  { value: 'business_plan', label: '📈 商业计划书' },
  { value: 'daily_report', label: '📅 周报日报' },
]

export default function WritingPage() {
  const [category, setCategory] = useState('work_summary')
  const [prompt, setPrompt] = useState('')
  const [content, setContent] = useState('')
  const [generating, setGenerating] = useState(false)
  const [title, setTitle] = useState('')
  const [activeTab, setActiveTab] = useState('write')

  const handleGenerate = async () => {
    if (!prompt.trim()) { MessagePlugin.error('请输入写作提示'); return }
    setGenerating(true)
    try {
      const res = await client.post('/writing/generate', { category, prompt })
      setContent(res.data.content)
      MessagePlugin.success('内容生成完成')
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '生成失败')
    } finally {
      setGenerating(false)
    }
  }

  const handleSave = async () => {
    if (!title.trim()) { MessagePlugin.error('请输入标题'); return }
    try {
      await client.post('/writing/drafts', { title, category, content })
      MessagePlugin.success('草稿已保存')
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '保存失败')
    }
  }

  return (
    <div style={{ padding: 24 }}>
      <h2 style={{ marginBottom: 24 }}>✍️ AI 写作助手</h2>

      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.TabPanel value="write" label="写作工作台">
          <div style={{ display: 'flex', gap: 24, marginTop: 16 }}>
            {/* Left panel - category & prompt */}
            <Card style={{ width: 320 }}>
              <h4 style={{ marginBottom: 16 }}>选择写作类别</h4>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 24 }}>
                {CATEGORIES.map(cat => (
                  <Button
                    key={cat.value}
                    variant={category === cat.value ? 'base' : 'outline'}
                    onClick={() => setCategory(cat.value)}
                    block
                  >
                    {cat.label}
                  </Button>
                ))}
              </div>

              <h4 style={{ marginBottom: 8 }}>写作提示</h4>
              <Input.Textarea
                placeholder="描述您想写的内容..."
                value={prompt}
                onChange={setPrompt}
                rows={4}
                style={{ marginBottom: 16 }}
              />

              <Button
                theme="primary"
                block
                onClick={handleGenerate}
                loading={generating}
              >
                🤖 AI 生成
              </Button>
            </Card>

            {/* Right panel - content editor */}
            <Card style={{ flex: 1 }}>
              <div style={{ marginBottom: 16 }}>
                <Input
                  placeholder="文档标题"
                  value={title}
                  onChange={setTitle}
                  size="large"
                  style={{ fontSize: 18 }}
                />
              </div>

              <Input.Textarea
                placeholder="AI 生成的内容将显示在这里，您也可以直接编辑..."
                value={content}
                onChange={setContent}
                rows={20}
                style={{ marginBottom: 16, fontFamily: 'monospace' }}
              />

              <div style={{ display: 'flex', gap: 12 }}>
                <Button theme="primary" onClick={handleSave}>💾 保存草稿</Button>
                <Button variant="outline" disabled title="即将推出">📤 导出 PDF</Button>
                <Button variant="outline" disabled title="即将推出">📤 导出 DOCX</Button>
                <Button variant="outline" disabled title="即将推出">📤 导出 Markdown</Button>
              </div>
            </Card>
          </div>
        </Tabs.TabPanel>

        <Tabs.TabPanel value="drafts" label="草稿箱">
          <DraftsList />
        </Tabs.TabPanel>
      </Tabs>
    </div>
  )
}

function DraftsList() {
  const [drafts, setDrafts] = useState<any[]>([])

  useEffect(() => {
    client.get('/writing/drafts').then(res => setDrafts(res.data.drafts || [])).catch((err) => { console.error('API error:', err) })
  }, [])

  return (
    <div style={{ marginTop: 16 }}>
      {drafts.length === 0 ? (
        <Card style={{ textAlign: 'center', padding: 60 }}>
          <p style={{ fontSize: 48 }}>📝</p>
          <p style={{ color: '#999' }}>暂无草稿</p>
        </Card>
      ) : (
        drafts.map(draft => (
          <Card key={draft.id} style={{ marginBottom: 12 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <div style={{ fontSize: 16, fontWeight: 500 }}>{draft.title || '无标题'}</div>
                <div style={{ fontSize: 13, color: '#999', marginTop: 4 }}>
                  {draft.category} · {new Date(draft.updated_at).toLocaleString()}
                </div>
              </div>
              <Button size="small" variant="outline" disabled title="即将推出">编辑</Button>
            </div>
          </Card>
        ))
      )}
    </div>
  )
}
