import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Form, Input, Button, MessagePlugin, Tabs } from 'tdesign-react'
import { useAuthStore } from '../../stores/auth'

export default function RegisterPage() {
  const navigate = useNavigate()
  const { register, isLoading } = useAuthStore()
  const [tab, setTab] = useState<string>('phone')
  const [phone, setPhone] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [nickname, setNickname] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (password !== confirmPassword) {
      MessagePlugin.error('两次密码输入不一致')
      return
    }
    if (password.length < 8) {
      MessagePlugin.error('密码至少需要8位')
      return
    }
    try {
      await register({
        phone: tab === 'phone' ? phone : undefined,
        email: tab === 'email' ? email : undefined,
        password,
        nickname: nickname || undefined,
      })
      MessagePlugin.success('注册成功')
      navigate('/workspace')
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '注册失败')
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
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <h1 style={{ fontSize: 28, fontWeight: 600, color: '#0052D9', margin: 0 }}>
            随越·智枢
          </h1>
          <p style={{ fontSize: 14, color: '#8B8B8B', margin: '8px 0 0' }}>
            创建您的账号
          </p>
        </div>

        <Tabs value={tab} onChange={setTab}>
          <Tabs.TabPanel value="phone" label="手机号注册">
            <Form onSubmit={handleSubmit} style={{ marginTop: 24 }}>
              <Form.FormItem>
                <Input placeholder="请输入手机号" value={phone} onChange={setPhone} size="large" />
              </Form.FormItem>
              <Form.FormItem>
                <Input placeholder="昵称（选填）" value={nickname} onChange={setNickname} size="large" />
              </Form.FormItem>
              <Form.FormItem>
                <Input type="password" placeholder="密码（至少8位）" value={password} onChange={setPassword} size="large" />
              </Form.FormItem>
              <Form.FormItem>
                <Input type="password" placeholder="确认密码" value={confirmPassword} onChange={setConfirmPassword} size="large" />
              </Form.FormItem>
              <Form.FormItem>
                <Button type="submit" theme="primary" size="large" block loading={isLoading}
                  disabled={!phone || !password || !confirmPassword}>
                  注册
                </Button>
              </Form.FormItem>
            </Form>
          </Tabs.TabPanel>

          <Tabs.TabPanel value="email" label="邮箱注册">
            <Form onSubmit={handleSubmit} style={{ marginTop: 24 }}>
              <Form.FormItem>
                <Input placeholder="请输入邮箱" value={email} onChange={setEmail} size="large" />
              </Form.FormItem>
              <Form.FormItem>
                <Input placeholder="昵称（选填）" value={nickname} onChange={setNickname} size="large" />
              </Form.FormItem>
              <Form.FormItem>
                <Input type="password" placeholder="密码（至少8位）" value={password} onChange={setPassword} size="large" />
              </Form.FormItem>
              <Form.FormItem>
                <Input type="password" placeholder="确认密码" value={confirmPassword} onChange={setConfirmPassword} size="large" />
              </Form.FormItem>
              <Form.FormItem>
                <Button type="submit" theme="primary" size="large" block loading={isLoading}
                  disabled={!email || !password || !confirmPassword}>
                  注册
                </Button>
              </Form.FormItem>
            </Form>
          </Tabs.TabPanel>
        </Tabs>

        <div style={{ textAlign: 'center', marginTop: 24, fontSize: 14 }}>
          <span style={{ color: '#999' }}>已有账号？</span>
          <Link to="/login" style={{ color: '#0052D9', marginLeft: 4 }}>立即登录</Link>
        </div>
      </div>
    </div>
  )
}
