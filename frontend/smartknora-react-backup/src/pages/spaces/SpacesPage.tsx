import { useEffect, useState } from 'react'
import { Card, Button, Row, Col, Dialog, Form, Input, Select, MessagePlugin } from 'tdesign-react'
import { useNavigate } from 'react-router-dom'
import { spaceApi } from '../../api/client'
import type { KnowledgeSpace } from '../../types'

export default function SpacesPage() {
  const navigate = useNavigate()
  const [spaces, setSpaces] = useState<KnowledgeSpace[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [newVisibility, setNewVisibility] = useState('team')

  const loadSpaces = () => {
    spaceApi.list().then(res => setSpaces(res.data.spaces || [])).catch((err) => { console.error('API error:', err) })
  }

  useEffect(() => { loadSpaces() }, [])

  const handleCreate = async () => {
    if (!newName.trim()) { MessagePlugin.error('请输入空间名称'); return }
    try {
      await spaceApi.create({ name: newName, description: newDesc, visibility: newVisibility })
      MessagePlugin.success('创建成功')
      setShowCreate(false)
      setNewName(''); setNewDesc(''); setNewVisibility('team')
      loadSpaces()
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '创建失败')
    }
  }

  const handleDelete = async (id: string, name: string) => {
    const confirmed = await Dialog.confirm({ header: '确认删除', body: `确定要删除知识空间「${name}」吗？` })
    if (confirmed) {
      try {
        await spaceApi.delete(id)
        MessagePlugin.success('已删除')
        loadSpaces()
      } catch (err: any) {
        MessagePlugin.error(err?.response?.data?.error || '删除失败')
      }
    }
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ margin: 0 }}>📚 知识空间</h2>
        <Button theme="primary" onClick={() => setShowCreate(true)}>+ 创建空间</Button>
      </div>

      {spaces.length === 0 ? (
        <Card style={{ textAlign: 'center', padding: 60 }}>
          <p style={{ fontSize: 48, margin: 0 }}>📚</p>
          <p style={{ color: '#999', marginTop: 16 }}>还没有知识空间</p>
          <Button theme="primary" onClick={() => setShowCreate(true)}>创建第一个空间</Button>
        </Card>
      ) : (
        <Row gutter={16}>
          {spaces.map(space => (
            <Col key={space.id} span={8}>
              <Card
                hoverShadow
                style={{ cursor: 'pointer', marginBottom: 16 }}
                actions={
                  <Button variant="text" size="small" theme="danger"
                    onClick={(e) => { e.stopPropagation(); handleDelete(space.id, space.name) }}>
                    删除
                  </Button>
                }
              >
                <div onClick={() => navigate(`/spaces/${space.id}`)}>
                  <div style={{ fontSize: 18, fontWeight: 500 }}>{space.icon || '📚'} {space.name}</div>
                  <p style={{ color: '#999', fontSize: 13, marginTop: 8, minHeight: 40 }}>
                    {space.description || '暂无描述'}
                  </p>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 12 }}>
                    <span style={{ fontSize: 12, color: '#bbb' }}>
                      {space.visibility === 'private' ? '🔒 私密' : space.visibility === 'team' ? '👥 团队' : '🏢 企业'}
                    </span>
                    <span style={{ fontSize: 12, color: '#bbb' }}>
                      {new Date(space.created_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {/* Create Dialog */}
      <Dialog
        header="创建知识空间"
        visible={showCreate}
        onConfirm={handleCreate}
        onClose={() => setShowCreate(false)}
      >
        <Form>
          <Form.FormItem label="空间名称" required>
            <Input placeholder="请输入空间名称" value={newName} onChange={setNewName} />
          </Form.FormItem>
          <Form.FormItem label="描述">
            <Input placeholder="空间描述（选填）" value={newDesc} onChange={setNewDesc} />
          </Form.FormItem>
          <Form.FormItem label="可见性">
            <Select value={newVisibility} onChange={setNewVisibility} options={[
              { label: '🔒 私密（仅成员）', value: 'private' },
              { label: '👥 团队（组织内可见）', value: 'team' },
              { label: '🏢 企业（全企业可见）', value: 'org' },
            ]} />
          </Form.FormItem>
        </Form>
      </Dialog>
    </div>
  )
}
