import { createBrowserRouter, Navigate } from 'react-router-dom'

import App from '../App'
import AdminLayout from '../layouts/AdminLayout'
import Login from '../pages/Auth/Login'
import Customers from '../pages/Customers'
import Dashboard from '../pages/Dashboard'
import FTP from '../pages/FTP'
import Invoices from '../pages/Invoices'
import Packages from '../pages/Packages'
import Payments from '../pages/Payments'
import Settings from '../pages/Settings'
import Subscriptions from '../pages/Subscriptions'
import Users from '../pages/Users'
import ProtectedRoute from './ProtectedRoute'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    children: [
      {
        path: 'login',
        element: <Login />,
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
                path: 'dashboard',
                element: <Dashboard />,
              },
              {
                path: 'customers',
                element: <Customers />,
              },
              {
                path: 'packages',
                element: <Packages />,
              },
              {
                path: 'subscriptions',
                element: <Subscriptions />,
              },
              {
                path: 'invoices',
                element: <Invoices />,
              },
              {
                path: 'payments',
                element: <Payments />,
              },
              {
                path: 'ftp',
                element: <FTP />,
              },
              {
                path: 'settings',
                element: <Settings />,
              },
              {
                path: 'users',
                element: <Users />,
              },
            ],
          },
        ],
      },
    ],
  },
])
