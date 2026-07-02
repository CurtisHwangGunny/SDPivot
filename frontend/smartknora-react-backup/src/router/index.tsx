import { createBrowserRouter, Navigate } from 'react-router-dom'
import MainLayout from '../layouts/MainLayout'
import LoginPage from '../pages/auth/LoginPage'
import RegisterPage from '../pages/auth/RegisterPage'
import WorkspacePage from '../pages/workspace/WorkspacePage'
import SpacesPage from '../pages/spaces/SpacesPage'
import SpaceDetailPage from '../pages/spaces/SpaceDetailPage'
import DocumentsPage from '../pages/spaces/DocumentsPage'
import OrgPage from '../pages/org/OrgPage'
import SettingsPage from '../pages/settings/SettingsPage'
import UsagePage from '../pages/usage/UsagePage'
import QAPage from '../pages/qa/QAPage'
import WritingPage from '../pages/writing/WritingPage'
import OpsPage from '../pages/ops/OpsPage'

import { useAuthStore } from '../stores/auth'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore()
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <RegisterPage />,
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <MainLayout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <Navigate to="/workspace" replace /> },
      { path: 'workspace', element: <WorkspacePage /> },
      { path: 'spaces', element: <SpacesPage /> },
      { path: 'spaces/:id', element: <SpaceDetailPage /> },
      { path: 'spaces/:spaceId/documents', element: <DocumentsPage /> },
      { path: 'qa', element: <QAPage /> },
      { path: 'writing', element: <WritingPage /> },
      { path: 'org', element: <OrgPage /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: 'usage', element: <UsagePage /> },
      { path: 'ops', element: <OpsPage /> },
    ],
  },
])
