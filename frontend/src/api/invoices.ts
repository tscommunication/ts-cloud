import apiClient from './client'

export interface Invoice {
  id: number
  invoice_no: string
  subscription_id: number
  customer_id: number
  package_id: number
  bill_month: number
  bill_year: number
  issue_date: string
  due_date: string
  package_price: number
  discount: number
  vat: number
  total_amount: number
  paid_amount: number
  due_amount: number
  status: string
  remarks: string
}

export interface CreateInvoiceRequest {
  subscription_id: number
  bill_month: number
  bill_year: number
  issue_date: string
  due_date: string
  package_price: number
  discount: number
  vat: number
  remarks: string
}

export async function getInvoices(): Promise<Invoice[]> {
  const response = await apiClient.get<Invoice[]>(
    '/invoices',
  )

  return response.data
}

export async function getInvoice(
  id: number,
): Promise<Invoice> {
  const response = await apiClient.get<Invoice>(
    `/invoices/${id}`,
  )

  return response.data
}

export async function createInvoice(
  data: CreateInvoiceRequest,
): Promise<Invoice> {
  const response = await apiClient.post<Invoice>(
    '/invoices',
    data,
  )

  return response.data
}

export async function updateInvoice(
  id: number,
  data: CreateInvoiceRequest,
): Promise<Invoice> {
  const response = await apiClient.put<Invoice>(
    `/invoices/${id}`,
    data,
  )

  return response.data
}

export async function deleteInvoice(
  id: number,
): Promise<void> {
  await apiClient.delete(`/invoices/${id}`)
}
