import { createContext, useContext } from 'react'

import type { DashboardView } from './dashboardView'

export const DashboardSettingsContext = createContext<{
  dashboardView: DashboardView
  setDashboardView: (view: DashboardView) => void
} | null>(null)

export function useDashboardSettings() {
  const context = useContext(DashboardSettingsContext)
  if (!context) throw new Error('useDashboardSettings must be used inside DashboardSettingsProvider')
  return context
}
