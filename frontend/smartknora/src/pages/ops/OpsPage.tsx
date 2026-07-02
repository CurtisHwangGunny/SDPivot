import { useEffect, useState } from 'react'
import { Card, Row, Col, Statistic, Table, Button, Tag, Tabs, Loading } from 'tdesign-react'
import client from '../../api/client'

export default function OpsPage() {
  const [stats, setStats] = useState<any>({})
  const [activeTab, setActiveTab] = useState('dashboard')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    client.get('/ops/dashboard').then(res => setStats(res.data)).catch((err) => { console.error('API error:', err) }).finally(() => setLoading(false))
  }, [])

  return (
    <Loading loading={loading} size="large">
    <div style={{ padding: 24 }}>
      <h2 style={{ marginBottom: 24 }}>🏢 运营管理</h2>

      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.TabPanel value="dashboard" label="📊 数据看板">
          <Row gutter={16} style={{ marginTop: 16 }}>
            <Col span={6}>
              <Card><Statistic title="企业数量" value={stats.tenant_count || 0} suffix="家" /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="用户数量" value={stats.user_count || 0} suffix="人" /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="文档数量" value={stats.document_count || 0} suffix="篇" /></Card>
            </Col>
            <Col span={6}>
              <Card><Statistic title="知识空间" value={stats.space_count || 0} suffix="个" /></Card>
            </Col>
          </Row>
        </Tabs.TabPanel>

        <Tabs.TabPanel value="tenants" label="🏢 企业管理">
          <TenantsList />
        </Tabs.TabPanel>

        <Tabs.TabPanel value="announcements" label="📢 公告管理">
          <AnnouncementsList />
        </Tabs.TabPanel>

        <Tabs.TabPanel value="audit" label="📋 操作日志">
          <AuditLogList />
        </Tabs.TabPanel>
      </Tabs>
    </div>
    </Loading>
  )
}

function TenantsList() {
  const [tenants, setTenants] = useState<any[]>([])

  useEffect(() => {
    client.get('/ops/tenants').then(res => setTenants(res.data.tenants || [])).catch((err) => { console.error('API error:', err) })
  }, [])

  const columns = [
    { title: '企业名称', colKey: 'name', width: 200 },
    { title: '认证状态', colKey: 'auth_status', width: 120, cell: ({ row }: any) => (
      <Tag theme={row.auth_status === 'certified' ? 'success' : row.auth_status === 'trial' ? 'warning' : 'danger'}>
        {row.auth_status === 'certified' ? '已认证' : row.auth_status === 'trial' ? '试用中' : row.auth_status}
      </Tag>
    )},
    { title: '创建时间', colKey: 'created_at', width: 180 },
    { title: '操作', colKey: 'actions', width: 200, cell: () => (
      <div style={{ display: 'flex', gap: 8 }}>
        <Button size="small" variant="outline" disabled title="即将推出">查看</Button>
        <Button size="small" variant="outline" theme="danger" disabled title="即将推出">禁用</Button>
      </div>
    )},
  ]

  return (
    <div style={{ marginTop: 16 }}>
      <Table data={tenants} columns={columns} rowKey="id" bordered />
    </div>
  )
}

function AnnouncementsList() {
  const [announcements, setAnnouncements] = useState<any[]>([])

  useEffect(() => {
    client.get('/ops/announcements').then(res => setAnnouncements(res.data.announcements || [])).catch((err) => { console.error('API error:', err) })
  }, [])

  return (
    <div style={{ marginTop: 16 }}>
      <Button theme="primary" style={{ marginBottom: 16 }} disabled title="即将推出">+ 发布公告</Button>
      {announcements.length === 0 ? (
        <Card style={{ textAlign: 'center', padding: 40 }}>
          <p style={{ color: '#999' }}>暂无公告</p>
        </Card>
      ) : (
        announcements.map(a => (
          <Card key={a.id} style={{ marginBottom: 12 }}>
            <div style={{ fontWeight: 500 }}>{a.title}</div>
            <div style={{ color: '#666', marginTop: 8, fontSize: 14 }}>{a.content}</div>
            <div style={{ color: '#999', marginTop: 8, fontSize: 12 }}>
              {new Date(a.created_at).toLocaleString()}
            </div>
          </Card>
        ))
      )}
    </div>
  )
}

function AuditLogList() {
  const [logs, setLogs] = useState<any[]>([])

  useEffect(() => {
    client.get('/ops/audit-log').then(res => setLogs(res.data.logs || [])).catch((err) => { console.error('API error:', err) })
  }, [])

  const columns = [
    { title: '时间', colKey: 'timestamp', width: 180 },
    { title: '用户', colKey: 'user_id', width: 150 },
    { title: '操作', colKey: 'action', width: 200 },
    { title: '详情', colKey: 'details', width: 300 },
  ]

  return (
    <div style={{ marginTop: 16 }}>
      <Table data={logs} columns={columns} rowKey="id" bordered />
    </div>
  )
}
