import { Route, Routes, Navigate } from 'react-router-dom'
import Layout from '@components/Layout'
import Login from '@pages/Login'
import Dashboard from '@pages/Dashboard'
import Gadgets from '@pages/Gadgets'
import AdminUsers from '@pages/AdminUsers'
import Files from '@pages/Files'
import Settings from '@pages/Settings'
import ProtectedRoute from '@utils/ProtectedRoute'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="gadgets" element={<Gadgets />} />
        <Route
          path="admin/users"
          element={
            <ProtectedRoute requiredRoles={["admin"]}>
              <AdminUsers />
            </ProtectedRoute>
          }
        />
        <Route path="files" element={<Files />} />
        <Route path="settings" element={<Settings />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App


