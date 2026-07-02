import { useState, useEffect } from 'react'
import { Card, Form, Input, Button, MessagePlugin } from 'tdesign-react'
import { userApi } from '../../api/client'
import { useAuthStore } from '../../stores/auth'

export default function SettingsPage() {
  const { user } = useAuthStore()
  const [nickname, setNickname] = useState(user?.username || '')
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  useEffect(() => { if (user?.username) setNickname(user.username) }, [user?.username])

  const handleUpdateProfile = async () => {
    try {
      await userApi.updateProfile({ nickname })
      MessagePlugin.success('资料更新成功')
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '更新失败')
    }
  }

  const handleChangePassword = async () => {
    if (newPassword !== confirmPassword) {
      MessagePlugin.error('两次密码输入不一致')
      return
    }
    if (newPassword.length < 8) {
      MessagePlugin.error('密码至少需要8位')
      return
    }
    try {
      await userApi.changePassword({ old_password: oldPassword, new_password: newPassword })
      MessagePlugin.success('密码修改成功')
      setOldPassword(''); setNewPassword(''); setConfirmPassword('')
    } catch (err: any) {
      MessagePlugin.error(err?.response?.data?.error || '修改失败')
    }
  }

  return (
    <div style={{ padding: 24, maxWidth: 600 }}>
      <h2 style={{ marginBottom: 24 }}>⚙️ 个人设置</h2>

      <Card title="基本信息" style={{ marginBottom: 24 }}>
        <Form>
          <Form.FormItem label="用户名">
            <Input value={user?.username || ''} disabled />
          </Form.FormItem>
          <Form.FormItem label="邮箱">
            <Input value={user?.email || ''} disabled />
          </Form.FormItem>
          <Form.FormItem label="昵称">
            <Input value={nickname} onChange={setNickname} placeholder="设置您的昵称" />
          </Form.FormItem>
          <Form.FormItem>
            <Button theme="primary" onClick={handleUpdateProfile}>保存修改</Button>
          </Form.FormItem>
        </Form>
      </Card>

      <Card title="修改密码">
        <Form>
          <Form.FormItem label="当前密码">
            <Input type="password" value={oldPassword} onChange={setOldPassword} placeholder="请输入当前密码" />
          </Form.FormItem>
          <Form.FormItem label="新密码">
            <Input type="password" value={newPassword} onChange={setNewPassword} placeholder="至少8位" />
          </Form.FormItem>
          <Form.FormItem label="确认新密码">
            <Input type="password" value={confirmPassword} onChange={setConfirmPassword} placeholder="再次输入新密码" />
          </Form.FormItem>
          <Form.FormItem>
            <Button theme="primary" onClick={handleChangePassword}
              disabled={!oldPassword || !newPassword || !confirmPassword}>
              修改密码
            </Button>
          </Form.FormItem>
        </Form>
      </Card>
    </div>
  )
}
