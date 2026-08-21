import apiClient from './client'
export interface ServiceEntitlement { id:number; customer_id:number; customer_code?:string; customer_name?:string; subscription_id?:number; service_type:'JELLYFIN'|'IPTV'|'CLOUD_STORAGE'; service_name:string; username:string; endpoint:string; status:string; expiry_at?:string; quota_gb:number; remarks:string; password_configured:boolean }
export interface ServiceEntitlementRequest { customer_id:number; subscription_id?:number; service_type:string; service_name:string; username:string; password:string; endpoint:string; status:string; expiry_at?:string; quota_gb:number; remarks:string }
export async function getServiceEntitlements(){ return (await apiClient.get<{entitlements:ServiceEntitlement[]}>('/service-entitlements')).data.entitlements }
export async function createServiceEntitlement(data:ServiceEntitlementRequest){ return (await apiClient.post<ServiceEntitlement>('/service-entitlements',data)).data }
export async function updateServiceEntitlement(id:number,data:ServiceEntitlementRequest){ return (await apiClient.put<ServiceEntitlement>(`/service-entitlements/${id}`,data)).data }
export async function deleteServiceEntitlement(id:number){ await apiClient.delete(`/service-entitlements/${id}`) }
