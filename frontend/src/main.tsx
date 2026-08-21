import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import axios from 'axios'

import './index.css'
import { router } from './router'
import { ThemeSettingsProvider } from './theme/ThemeSettingsProvider'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (failureCount, error) => {
        const status = axios.isAxiosError(error) ? error.response?.status : undefined
        return failureCount < 2 && (status === undefined || status >= 500)
      },
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeSettingsProvider>
        <RouterProvider router={router} />
      </ThemeSettingsProvider>
    </QueryClientProvider>
  </StrictMode>,
)
