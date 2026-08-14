import apiClient from './client'

export type DataExportType = 'customers' | 'agent-users'
export type DataExportFormat = 'csv' | 'xlsx'

export async function downloadDataExport(type: DataExportType, format: DataExportFormat): Promise<void> {
  const response = await apiClient.get<Blob>('/data-exports', {
    params: { type, format },
    responseType: 'blob',
  })
  const disposition = response.headers['content-disposition'] as string | undefined
  const filename = disposition?.match(/filename="?([^";]+)"?/)?.[1] ?? `ts-cloud-${type}.${format}`
  const url = URL.createObjectURL(response.data)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}
