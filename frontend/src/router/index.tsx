import { createBrowserRouter, Navigate } from 'react-router-dom'

import App from '../App'
import AdminLayout from '../layouts/AdminLayout'
import Login from '../pages/Auth/Login'
import Customers from '../pages/Customers'
import Dashboard from '../pages/Dashboard'
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
            ],
          },
        ],
      },
    ],
  },
])
