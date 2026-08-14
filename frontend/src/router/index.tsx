import { createBrowserRouter, Navigate } from 'react-router-dom'

import App from '../App'
import AdminLayout from '../layouts/AdminLayout'
import ProtectedRoute from './ProtectedRoute'
import RoleRoute from './RoleRoute'
import RouteError from './RouteError'
import {
  Customers,
  Dashboard,
  FTP,
  Invoices,
  LazyRoute,
  Login,
  Packages,
  Payments,
  Organization,
  AgentCollections,
  NetworkRouters,
  PPPoESessions,
  Settings,
  Subscriptions,
  Users,
} from './LazyRoutes'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    errorElement: <RouteError />,
    children: [
      {
        path: 'login',
        element: <LazyRoute element={<Login />} />,
      },
      {
        element: <ProtectedRoute />,
        children: [
          {
            element: <AdminLayout />,
            children: [
              {
                index: true,
                element: <Navigate to="/dashboard" replace />,
              },
              {
                path: 'settings',
                element: <LazyRoute element={<Settings />} />,
              },
              {
                element: <RoleRoute roles={['superadmin', 'admin', 'agent']} />,
                children: [
                  {
                    path: 'dashboard',
                    element: <LazyRoute element={<Dashboard />} />,
                  },
                  { path: 'customers', element: <LazyRoute element={<Customers />} /> },
                  { path: 'agent-collections', element: <LazyRoute element={<AgentCollections />} /> },
                  { path: 'payments', element: <LazyRoute element={<Payments />} /> },
                ],
              },
              {
                element: <RoleRoute roles={['superadmin', 'admin']} />,
                children: [
                  {
                    path: 'organization',
                    element: <LazyRoute element={<Organization />} />,
                  },
				  { path: 'network/routers', element: <LazyRoute element={<NetworkRouters />} /> },
				  { path: 'network/pppoe-sessions', element: <LazyRoute element={<PPPoESessions />} /> },
                  {
                    path: 'packages',
                    element: <LazyRoute element={<Packages />} />,
                  },
                  {
                    path: 'subscriptions',
                    element: <LazyRoute element={<Subscriptions />} />,
                  },
                  {
                    path: 'invoices',
                    element: <LazyRoute element={<Invoices />} />,
                  },
                  {
                    path: 'ftp',
                    element: <LazyRoute element={<FTP />} />,
                  },
                  {
                    path: 'users',
                    element: <LazyRoute element={<Users />} />,
                  },
                ],
              },
            ],
          },
        ],
      },
      {
        path: '*',
        loader: () => {
          throw new Response('Not Found', { status: 404 })
        },
      },
    ],
  },
])
