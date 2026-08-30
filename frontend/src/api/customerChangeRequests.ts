import apiClient from './client'

export type CustomerChangeType = 'BILLING_CYCLE' | 'PACKAGE' | 'LINE_SHIFT' | 'CLOSE'
export interface CustomerChangeRequest { ID:number; request_code:string; type:CustomerChangeType; status:string; customer_id:number; agent_id:number; reason:string; current_value:string; requested_value:string; rejection_reason:string; execution_error:string; CreatedAt:string; reviewed_at?:string; executed_at?:string }
export interface CustomerChangeRequestInput { customer_id:number; type:CustomerChangeType; reason:string; current_value:string; requested_value:string }
export interface CustomerChangeRequestOption { id:number; code:string; name:string }
export interface CustomerChangeRequestOptions { packages:CustomerChangeRequestOption[]; routers:CustomerChangeRequestOption[] }
export async function getCustomerChangeRequests(status=''):Promise<CustomerChangeRequest[]>{return (await apiClient.get<{requests:CustomerChangeRequest[]}>('/customer-change-requests',{params:{status}})).data.requests}
export async function getCustomerChangeRequestOptions():Promise<CustomerChangeRequestOptions>{return (await apiClient.get<CustomerChangeRequestOptions>('/customer-change-requests/options')).data}
export async function createCustomerChangeRequest(input:CustomerChangeRequestInput):Promise<CustomerChangeRequest>{return (await apiClient.post<CustomerChangeRequest>('/customer-change-requests',input)).data}
export async function approveCustomerChangeRequest(id:number):Promise<CustomerChangeRequest>{return (await apiClient.post<CustomerChangeRequest>(`/customer-change-requests/${id}/approve`,{})).data}
export async function rejectCustomerChangeRequest(id:number,reason:string):Promise<CustomerChangeRequest>{return (await apiClient.post<CustomerChangeRequest>(`/customer-change-requests/${id}/reject`,{reason})).data}
