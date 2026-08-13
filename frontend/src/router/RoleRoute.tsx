import { Navigate, Outlet } from 'react-router-dom'
import { getStoredUser } from '../api/auth'

interface RoleRouteProps {
  roles: string[]
}

export default function RoleRoute({ roles }: RoleRouteProps) {
  const role = getStoredUser()?.role

  if (!role || !roles.includes(role)) {
    return <Navigate to="/settings" replace />
  }

  return <Outlet />
}
