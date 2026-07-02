import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Table, Button, Tag, MessagePlugin } from 'tdesign-react'
import { spaceApi } from '../../api/client'
import type { KnowledgeSpace, SpaceMember } from '../../types'

export default function SpaceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [space, setSpace] = useState<KnowledgeSpace | null>(null)
  const [members, setMembers] = useState<SpaceMember[]>([])

  useEffect(() => {
    if (!id) return
    spaceApi.get(id).then(res => setSpace(res.data.space)).catch((err) => { console.error('API error:', err) })
    spaceApi.listMembers(id).then(res => setMembers(res.data.members || [])).catch((err) => { console.error('API error:', err) })
  }, [id])

  if (!space) return <div style={{ padding: 24 }}>加载中...</div>

  const memberColumns = [
    { title: '用户ID', colKey: 'user_id', width: 300 },
    { title: '角色', colKey: 'role', width: 120, cell: ({ row }: any) => (
      <Tag theme={row.role === 'owner' ? 'danger' : row.role === 'editor' ? 'warning' : 'default'}>
        {row.role === 'owner' ? '所有者' : row.role === 'editor' ? '编辑者' : '查看者'}
      </Tag>
    )},
    { title: '加入时间', colKey: 'created_at', width: 200, cell: ({ row }: any) =>
      new Date(row.created_at).toLocaleString()
    },
  ]

  return (
    <div style={{ padding: 24 }}>
      {/* Space Header */}
      <Card style={{ marginBottom: 24 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <h2 style={{ margin: 0 }}>{space.icon || '📚'} {space.name}</h2>
            <p style={{ color: '#999', margin: '8px 0 0' }}>{space.description || '暂无描述'}</p>
          </div>
          <Tag theme="primary" variant="outline">
            {space.visibility === 'private' ? '🔒 私密' : space.visibility === 'team' ? '👥 团队' : '🏢 企业'}
          </Tag>
        </div>
      </Card>

      {/* Members */}
      <Card title="空间成员" actions={
        <Button size="small" theme="primary" disabled title="即将推出">+ 邀请成员</Button>
      }>
        <Table
          data={members}
          columns={memberColumns}
          rowKey="id"
          bordered
          stripe
        />
      </Card>
    </div>
  )
}
