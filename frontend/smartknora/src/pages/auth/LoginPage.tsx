import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Form, Input, Button, MessagePlugin, Tabs } from 'tdesign-react'
import { useAuthStore } from '../../stores/auth'

export default function LoginPage() {
  const navigate = useNavigate()
  const { login, loginByEmail, isLoading } = useAuthStore()
  const [tab, setTab] = useState<string>('phone')
  const [phone, setPhone] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      if (tab === 'phone') {
        await login(phone, password)
      } else {
        await loginByEmail(email, password)
      }
      MessagePlugin.success('登录成功')
      navigate('/workspace')
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '登录失败')
    }
  }

  return (
    <div style={{
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      height: '100vh',
      background: '#f5f5f5',
    }}>
      <div style={{
        width: 420,
        background: '#fff',
        borderRadius: 16,
        padding: '40px 32px',
        boxShadow: '0 4px 16px rgba(0,0,0,0.08)',
      }}>
        {/* Brand */}
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <h1 style={{ fontSize: 28, fontWeight: 600, color: '#0052D9', margin: 0 }}>
            随越·智枢
          </h1>
          <p style={{ fontSize: 14, color: '#8B8B8B', margin: '8px 0 0' }}>
            企业智能知识平台 · AI 驱动
          </p>
        </div>

        {/* Login Tabs */}
        <Tabs value={tab} onChange={setTab}>
          <Tabs.TabPanel value="phone" label="手机号登录">
            <Form onSubmit={handleSubmit} style={{ marginTop: 24 }}>
              <Form.FormItem>
                <Input
                  placeholder="请输入手机号"
                  value={phone}
                  onChange={setPhone}
                  size="large"
                />
              </Form.FormItem>
              <Form.FormItem>
                <Input
                  type="password"
                  placeholder="请输入密码"
                  value={password}
                  onChange={setPassword}
                  size="large"
                />
              </Form.FormItem>
              <Form.FormItem>
                <Button
                  type="submit"
                  theme="primary"
                  size="large"
                  block
                  loading={isLoading}
                  disabled={!phone || !password}
                >
                  登录
                </Button>
              </Form.FormItem>
            </Form>
          </Tabs.TabPanel>

          <Tabs.TabPanel value="email" label="邮箱登录">
            <Form onSubmit={handleSubmit} style={{ marginTop: 24 }}>
              <Form.FormItem>
                <Input
                  placeholder="请输入邮箱"
                  value={email}
                  onChange={setEmail}
                  size="large"
                />
              </Form.FormItem>
              <Form.FormItem>
                <Input
                  type="password"
                  placeholder="请输入密码"
                  value={password}
                  onChange={setPassword}
                  size="large"
                />
              </Form.FormItem>
              <Form.FormItem>
                <Button
                  type="submit"
                  theme="primary"
                  size="large"
                  block
                  loading={isLoading}
                  disabled={!email || !password}
                >
                  登录
                </Button>
              </Form.FormItem>
            </Form>
          </Tabs.TabPanel>

          <Tabs.TabPanel value="wechat" label="微信扫码" disabled>
            <div style={{ padding: '40px 0', textAlign: 'center', color: '#999' }}>
              微信扫码登录将在 Phase 2 开放
            </div>
          </Tabs.TabPanel>
        </Tabs>

        {/* Footer */}
        <div style={{ textAlign: 'center', marginTop: 24, fontSize: 14 }}>
          <span style={{ color: '#999' }}>还没有账号？</span>
          <Link to="/register" style={{ color: '#0052D9', marginLeft: 4 }}>
            立即注册
          </Link>
        </div>
      </div>
    </div>
  )
}
