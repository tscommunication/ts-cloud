export type DashboardView = 'standard' | 'portal' | 'noc' | 'architecture'

export const dashboardViews: Array<{
  value: DashboardView
  label: string
  description: string
}> = [
  { value: 'standard', label: 'TS-Cloud Standard', description: 'Default operational dashboard' },
  { value: 'portal', label: 'Customer-first View', description: 'Portal-inspired account and service focus' },
  { value: 'noc', label: 'NOC Command Center', description: 'Network and operations focus' },
  { value: 'architecture', label: 'Architecture View', description: 'ISP service and network topology focus' },
]
