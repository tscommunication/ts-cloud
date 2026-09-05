import { useState, type ReactNode } from 'react'

import { dashboardViews, type DashboardView } from './dashboardView'
import { DashboardSettingsContext } from './useDashboardSettings'

export function DashboardSettingsProvider({ children }: { children: ReactNode }) {
  const [dashboardView, updateDashboardView] = useState<DashboardView>(() => {
    const saved = localStorage.getItem('ts-cloud-dashboard-view')
    return dashboardViews.some((view) => view.value === saved) ? saved as DashboardView : 'standard'
  })

  const setDashboardView = (view: DashboardView) => {
    localStorage.setItem('ts-cloud-dashboard-view', view)
    updateDashboardView(view)
  }

  return <DashboardSettingsContext.Provider value={{ dashboardView, setDashboardView }}>{children}</DashboardSettingsContext.Provider>
}
