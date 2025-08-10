import { Navigate, useLocation } from 'react-router-dom'
import useAuthStore from '@store/auth'
import { PropsWithChildren } from 'react'

export default function ProtectedRoute({ children, requiredRoles }: PropsWithChildren<{ requiredRoles?: string[] }>) {
  const { token, roles } = useAuthStore()
  const location = useLocation()
  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }
  if (requiredRoles && !requiredRoles.some(r => roles.includes(r))) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}


