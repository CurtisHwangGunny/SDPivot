import { useEffect, useState } from 'react'
import { Card, Row, Col, Statistic, Table, Loading } from 'tdesign-react'
import { usageApi } from '../../api/client'
import type { TokenUsageSummary } from '../../types'

export default function UsagePage() {
  const [summary, setSummary] = useState<TokenUsageSummary | null>(null)
  const [byModel, setByModel] = useState<any[]>([])

  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      usageApi.summary().then(res => setSummary(res.data.summary)).catch((err) => { console.error('API error:', err) }),
      usageApi.byModel().then(res => setByModel(res.data.models || [])).catch((err) => { console.error('API error:', err) }),
    ]).finally(() => setLoading(false))
  }, [])

  const modelColumns = [
    { title: '模型', colKey: 'model_id', width: 300 },
    { title: '总 Token', colKey: 'total_tokens', width: 150 },
    { title: '请求次数', colKey: 'request_count', width: 120 },
  ]

  return (
    <Loading loading={loading} size="large">
    <div style={{ padding: 24 }}>
      <h2 style={{ marginBottom: 24 }}>📊 用量统计</h2>

      {/* Summary Cards */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card><Statistic title="总 Token 消耗" value={summary?.total_tokens || 0} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="Prompt Tokens" value={summary?.total_prompt_tokens || 0} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="Completion Tokens" value={summary?.total_completion_tokens || 0} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="请求次数" value={summary?.request_count || 0} suffix="次" /></Card>
        </Col>
      </Row>

      {/* By Model Table */}
      <Card title="按模型统计">
        <Table
          data={byModel}
          columns={modelColumns}
          rowKey="model_id"
          bordered
          stripe
        />
      </Card>
    </div>
    </Loading>
  )
}
