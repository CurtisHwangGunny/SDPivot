import { useEffect, useState } from 'react'
import { Card, Table, Button, Tag, Upload, Dialog, Form, Input, Select, MessagePlugin, Progress } from 'tdesign-react'
import { useParams } from 'react-router-dom'
import client from '../../api/client'

interface Document {
  id: string
  title: string
  file_name: string
  file_type: string
  file_size: number
  parse_status: string
  chunk_count: number
  embedding_status: string
  created_at: string
}

export default function DocumentsPage() {
  const { spaceId } = useParams<{ spaceId: string }>()
  const [docs, setDocs] = useState<Document[]>([])
  const [loading, setLoading] = useState(false)
  const [showUpload, setShowUpload] = useState(false)
  const [search, setSearch] = useState('')

  const loadDocs = () => {
    setLoading(true)
    client.get('/documents', { params: { space_id: spaceId, search } })
      .then(res => setDocs(res.data.documents || []))
      .catch((err) => { console.error('API error:', err) })
      .finally(() => setLoading(false))
  }
  useEffect(() => {
    const timer = setTimeout(() => { loadDocs() }, 300)
    return () => clearTimeout(timer)
  }, [spaceId, search])

  const handleUpload = (file: any) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('space_id', spaceId || '')
    client.post('/documents/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }).then(() => {
      MessagePlugin.success('上传成功')
      loadDocs()
    }).catch(err => {
      MessagePlugin.error(err?.response?.data?.error || '上传失败')
    })
    return false // Prevent default upload
  }

  const handleDelete = async (id: string) => {
    const confirmed = await Dialog.confirm({ header: '确认删除', body: '确定要删除此文档吗？删除后无法恢复。' })
    if (confirmed) {
      try {
        await client.delete(`/documents/${id}`)
        MessagePlugin.success('已删除')
        loadDocs()
      } catch (err: any) {
        MessagePlugin.error(err?.response?.data?.error || '删除失败')
      }
    }
  }

  const handleReparse = async (id: string) => {
    try {
      await client.post(`/documents/${id}/reparse`)
      MessagePlugin.success('重新解析已触发')
      loadDocs()
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '操作失败')
    }
  }

  const statusColor = (s: string) => {
    switch (s) {
      case 'completed': return 'success'
      case 'parsing': return 'warning'
      case 'failed': return 'danger'
      default: return 'default'
    }
  }

  const columns = [
    { title: '文件名', colKey: 'title', width: 300 },
    { title: '类型', colKey: 'file_type', width: 100, cell: ({ row }: any) => (
      <Tag>{row.file_type?.replace('.', '').toUpperCase()}</Tag>
    )},
    { title: '大小', colKey: 'file_size', width: 100, cell: ({ row }: any) => (
      <span>{(row.file_size / 1024).toFixed(1)} KB</span>
    )},
    { title: '解析状态', colKey: 'parse_status', width: 120, cell: ({ row }: any) => (
      <Tag theme={statusColor(row.parse_status)}>
        {row.parse_status === 'completed' ? '已完成' : row.parse_status === 'parsing' ? '解析中' : row.parse_status === 'failed' ? '失败' : '待处理'}
      </Tag>
    )},
    { title: '分块数', colKey: 'chunk_count', width: 80 },
    { title: '上传时间', colKey: 'created_at', width: 180, cell: ({ row }: any) =>
      new Date(row.created_at).toLocaleString()
    },
    { title: '操作', colKey: 'actions', width: 200, cell: ({ row }: any) => (
      <div style={{ display: 'flex', gap: 8 }}>
        <Button size="small" variant="outline" onClick={() => handleReparse(row.id)}>重新解析</Button>
        <Button size="small" variant="outline" theme="danger" onClick={() => handleDelete(row.id)}>删除</Button>
      </div>
    )},
  ]

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ margin: 0 }}>📄 文档管理</h2>
        <div style={{ display: 'flex', gap: 12 }}>
          <Input placeholder="搜索文档..." value={search} onChange={setSearch} style={{ width: 200 }} />
          <Upload beforeUpload={handleUpload} showUploadList={false}>
            <Button theme="primary">+ 上传文档</Button>
          </Upload>
        </div>
      </div>

      <Card>
        <Table
          data={docs}
          columns={columns}
          rowKey="id"
          loading={loading}
          bordered
          stripe
          pagination={{ pageSize: 20 }}
        />
      </Card>
    </div>
  )
}
