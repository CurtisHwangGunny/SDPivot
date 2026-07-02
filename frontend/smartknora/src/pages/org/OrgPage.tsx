import { useEffect, useState } from 'react'
import { Card, Button, Table, Dialog, Form, Input, Tag, MessagePlugin, Loading } from 'tdesign-react'
import { orgApi } from '../../api/client'
import type { Organization, OrgMember } from '../../types'

export default function OrgPage() {
  const [orgs, setOrgs] = useState<Array<{ organization: Organization; auth_status: string; days_remaining: number }>>([])
  const [showCreate, setShowCreate] = useState(false)
  const [showJoin, setShowJoin] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [joinCode, setJoinCode] = useState('')
  const [selectedOrg, setSelectedOrg] = useState<string | null>(null)
  const [members, setMembers] = useState<OrgMember[]>([])
  const [loading, setLoading] = useState(true)

  const loadOrgs = () => {
    orgApi.list().then(res => setOrgs(res.data.organizations || [])).catch((err) => { console.error('API error:', err) }).finally(() => setLoading(false))
  }

  useEffect(() => { loadOrgs() }, [])

  const handleCreate = async () => {
    if (!newName.trim()) { MessagePlugin.error('请输入企业名称'); return }
    try {
      await orgApi.create({ name: newName, description: newDesc })
      MessagePlugin.success('创建成功')
      setShowCreate(false); setNewName(''); setNewDesc('')
      loadOrgs()
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '创建失败')
    }
  }

  const handleJoin = async () => {
    if (!joinCode.trim()) { MessagePlugin.error('请输入邀请码'); return }
    try {
      await orgApi.join(joinCode)
      MessagePlugin.success('加入成功')
      setShowJoin(false); setJoinCode('')
      loadOrgs()
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '加入失败')
    }
  }

  const viewMembers = async (orgId: string) => {
    setSelectedOrg(orgId)
    try {
      const res = await orgApi.listMembers(orgId)
      setMembers(res.data.members || [])
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '加载成员失败')
    }
  }

  const statusColor = (s: string) => {
    switch (s) {
      case 'trial': return 'warning'
      case 'certified': return 'success'
      case 'expired': return 'danger'
      default: return 'default'
    }
  }

  const memberColumns = [
    { title: '用户ID', colKey: 'user_id', width: 300 },
    { title: '角色', colKey: 'role', width: 120, cell: ({ row }: any) => (
      <Tag theme={row.role === 'owner' ? 'danger' : row.role === 'admin' ? 'warning' : 'default'}>
        {row.role}
      </Tag>
    )},
    { title: '加入时间', colKey: 'joined_at', width: 200, cell: ({ row }: any) =>
      new Date(row.joined_at).toLocaleString()
    },
  ]

  return (
    <Loading loading={loading} size="large">
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ margin: 0 }}>🏢 企业管理</h2>
        <div style={{ display: 'flex', gap: 12 }}>
          <Button theme="default" onClick={() => setShowJoin(true)}>加入企业</Button>
          <Button theme="primary" onClick={() => setShowCreate(true)}>+ 创建企业</Button>
        </div>
      </div>

      {orgs.length === 0 ? (
        <Card style={{ textAlign: 'center', padding: 60 }}>
          <p style={{ fontSize: 48, margin: 0 }}>🏢</p>
          <p style={{ color: '#999', marginTop: 16 }}>您还没有加入任何企业</p>
        </Card>
      ) : (
        orgs.map(({ organization: org, auth_status, days_remaining }) => (
          <Card key={org.id} style={{ marginBottom: 16 }}
            actions={<Button size="small" onClick={() => viewMembers(org.id)}>查看成员</Button>}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <div style={{ fontSize: 18, fontWeight: 500 }}>{org.name}</div>
                <p style={{ color: '#999', margin: '4px 0 0', fontSize: 13 }}>{org.description || '暂无描述'}</p>
              </div>
              <div style={{ textAlign: 'right' }}>
                <Tag theme={statusColor(auth_status) as any}>
                  {auth_status === 'trial' ? '试用中' : auth_status === 'certified' ? '已认证' : auth_status === 'expired' ? '已过期' : auth_status}
                </Tag>
                {auth_status === 'trial' && (
                  <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                    剩余 {days_remaining} 天
                  </div>
                )}
              </div>
            </div>
          </Card>
        ))
      )}

      {/* Members Panel */}
      {selectedOrg && (
        <Card title="企业成员" style={{ marginTop: 24 }}
          actions={<Button variant="text" size="small" onClick={() => setSelectedOrg(null)}>关闭</Button>}>
          <Table data={members} columns={memberColumns} rowKey="id" bordered />
        </Card>
      )}

      {/* Create Dialog */}
      <Dialog header="创建企业" visible={showCreate} onConfirm={handleCreate} onClose={() => setShowCreate(false)}>
        <Form>
          <Form.FormItem label="企业名称" required>
            <Input placeholder="请输入企业名称" value={newName} onChange={setNewName} />
          </Form.FormItem>
          <Form.FormItem label="描述">
            <Input placeholder="企业描述（选填）" value={newDesc} onChange={setNewDesc} />
          </Form.FormItem>
        </Form>
      </Dialog>

      {/* Join Dialog */}
      <Dialog header="加入企业" visible={showJoin} onConfirm={handleJoin} onClose={() => setShowJoin(false)}>
        <Form>
          <Form.FormItem label="邀请码" required>
            <Input placeholder="请输入企业邀请码" value={joinCode} onChange={setJoinCode} />
          </Form.FormItem>
        </Form>
      </Dialog>
    </div>
    </Loading>
  )
}
