import { useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Menu, Button, Avatar } from 'tdesign-react'
import { LogoutIcon } from 'tdesign-icons-react'
import { useAuthStore } from '../stores/auth'

const menuItems = [
  { value: '/workspace', label: '🏠 工作台' },
  { value: '/spaces', label: '📚 知识空间' },
  { value: '/qa', label: '🤖 AI 问答' },
  { value: '/writing', label: '✍️ AI 写作' },
  { value: '/org', label: '🏢 企业管理' },
  { value: '/usage', label: '📊 用量统计' },
  { value: '/ops', label: '⚙️ 运营管理' },
  { value: '/settings', label: '👤 个人设置' },
]

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout, loadFromStorage } = useAuthStore()

  useEffect(() => {
    loadFromStorage()
  }, [loadFromStorage])

  const handleMenuClick = (value: string) => {
    navigate(value)
  }

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const activeMenu = menuItems.reduce((best, item) => {
    if (location.pathname.startsWith(item.value) && item.value.length > best.length) return item.value
    return best
  }, '')

  return (
    <div style={{ display: 'flex', height: '100vh', background: '#f5f5f5' }}>
      {/* Sidebar */}
      <div style={{
        width: 240,
        background: '#fff',
        borderRight: '1px solid #e8e8e8',
        display: 'flex',
        flexDirection: 'column',
      }}>
        {/* Brand */}
        <div style={{
          padding: '24px 20px',
          borderBottom: '1px solid #e8e8e8',
        }}>
          <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600, color: '#0052D9' }}>
            随越·智枢
          </h2>
          <p style={{ margin: '4px 0 0', fontSize: 12, color: '#999' }}>
            企业智能知识平台
          </p>
        </div>

        {/* Navigation */}
        <Menu
          value={activeMenu}
          onChange={handleMenuClick}
          style={{ flex: 1, borderRight: 'none' }}
        >
          {menuItems.map((item) => (
            <Menu.MenuItem key={item.value} value={item.value}>
              {item.label}
            </Menu.MenuItem>
          ))}
        </Menu>

        {/* User Info */}
        <div style={{
          padding: '16px 20px',
          borderTop: '1px solid #e8e8e8',
          display: 'flex',
          alignItems: 'center',
          gap: 12,
        }}>
          <Avatar size="small">
            {user?.username?.[0]?.toUpperCase() || 'U'}
          </Avatar>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 14, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {user?.username || '用户'}
            </div>
            <div style={{ fontSize: 12, color: '#999' }}>
              {user?.email || ''}
            </div>
          </div>
          <Button
            variant="text"
            size="small"
            icon={<LogoutIcon />}
            onClick={handleLogout}
          />
        </div>
      </div>

      {/* Main Content */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        <Outlet />
      </div>
    </div>
  )
}
