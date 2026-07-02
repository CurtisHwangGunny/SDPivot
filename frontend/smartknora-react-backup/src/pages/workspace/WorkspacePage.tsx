import { useEffect, useState } from 'react'
import { Card, Row, Col, Statistic, Button, Loading } from 'tdesign-react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../../stores/auth'
import { spaceApi, usageApi } from '../../api/client'
import type { KnowledgeSpace, TokenUsageSummary } from '../../types'

export default function WorkspacePage() {
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [spaces, setSpaces] = useState<KnowledgeSpace[]>([])
  const [usage, setUsage] = useState<TokenUsageSummary | null>(null)

  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      spaceApi.list().then(res => setSpaces(res.data.spaces || [])).catch((err) => { console.error('API error:', err) }),
      usageApi.summary().then(res => setUsage(res.data.summary)).catch((err) => { console.error('API error:', err) }),
    ]).finally(() => setLoading(false))
  }, [])

  return (
    <Loading loading={loading} size="large">
    <div style={{ padding: 24 }}>
      {/* Welcome Banner */}
      <Card style={{ marginBottom: 24, background: 'linear-gradient(135deg, #0052D9 0%, #4A90E2 100%)', color: '#fff' }}>
        <h2 style={{ margin: 0, fontSize: 24, fontWeight: 600 }}>
          👋 欢迎回来，{user?.username || '用户'}
        </h2>
        <p style={{ margin: '8px 0 0', opacity: 0.8 }}>
          随越·智枢 — 企业智能知识平台
        </p>
      </Card>

      {/* Quick Stats */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic title="知识空间" value={spaces.length} suffix="个" />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="总 Token 消耗" value={usage?.total_tokens || 0} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="请求次数" value={usage?.request_count || 0} suffix="次" />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Prompt Tokens" value={usage?.total_prompt_tokens || 0} />
          </Card>
        </Col>
      </Row>

      {/* Quick Actions */}
      <Card title="快速操作">
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
          <Button theme="primary" onClick={() => navigate('/spaces')}>
            📚 管理知识空间
          </Button>
          <Button theme="default" onClick={() => navigate('/org')}>
            🏢 企业管理
          </Button>
          <Button theme="default" onClick={() => navigate('/usage')}>
            📊 查看用量
          </Button>
          <Button theme="default" onClick={() => navigate('/settings')}>
            ⚙️ 个人设置
          </Button>
        </div>
      </Card>

      {/* Recent Spaces */}
      <Card title="最近的知识空间" style={{ marginTop: 24 }}>
        {spaces.length === 0 ? (
          <p style={{ color: '#999', textAlign: 'center', padding: 40 }}>
            暂无知识空间，去创建一个吧！
          </p>
        ) : (
          <Row gutter={16}>
            {spaces.slice(0, 6).map(space => (
              <Col key={space.id} span={8}>
                <Card
                  hoverShadow
                  style={{ cursor: 'pointer', marginBottom: 16 }}
                  onClick={() => navigate(`/spaces/${space.id}`)}
                >
                  <div style={{ fontSize: 16, fontWeight: 500 }}>{space.icon || '📚'} {space.name}</div>
                  <p style={{ color: '#999', fontSize: 13, marginTop: 8 }}>
                    {space.description || '暂无描述'}
                  </p>
                  <div style={{ fontSize: 12, color: '#bbb', marginTop: 8 }}>
                    {space.visibility === 'private' ? '🔒 私密' : space.visibility === 'team' ? '👥 团队' : '🏢 企业'}
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        )}
      </Card>
    </div>
    </Loading>
  )
}
