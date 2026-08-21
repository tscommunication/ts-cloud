import apiClient from "./client";

export interface CustomerPortalMe {
  id: number;
  customer_code: string;
  full_name: string;
  mobile: string;
  alt_mobile: string;
  email: string;
  date_of_birth?: string;
  joining_date?: string;
  occupation: string;
  company_name: string;
  designation: string;
  present_address: string;
  permanent_address: string;
  country: string;
  division: string;
  district: string;
  upazila: string;
  post_office: string;
  postal_code: string;
  road_or_area: string;
  village_or_holding: string;
  status: string;
  billing_day: number;
  activation_date?: string;
}

export interface CustomerPortalSubscription {
  id: number;
  subscription_code: string;
  package_id: number;
  activation_date: string;
  billing_day: number;
  next_billing_date: string;
  expiry_date: string;
  status: string;
  pppoe_username: string;
  pppoe_password?: string;
  last_payment_date?: string;
  last_paid_amount: number;
  due_amount: number;
}

export interface CustomerPortalInvoice {
  id: number;
  invoice_no: string;
  subscription_id: number;
  package_id: number;
  bill_month: number;
  bill_year: number;
  issue_date: string;
  due_date: string;
  package_price: number;
  discount: number;
  vat: number;
  total_amount: number;
  paid_amount: number;
  due_amount: number;
  status: string;
}

export interface CustomerPortalPayment {
  id: number;
  receipt_no: string;
  invoice_id: number;
  subscription_id: number;
  payment_date: string;
  amount: number;
  method: string;
  transaction_id: string;
  status: string;
}

export interface CustomerPortalTemporaryAccess {
  ID: number;
  status: string;
  starts_at: string;
  ends_at: string;
  granted_duration_seconds: number;
  promised_payment_at?: string;
  promised_amount: number;
  request_source: string;
  reason: string;
  settled_at?: string;
}

export interface CustomerPortalFTPEntitlement {
  id: number;
  subscription_id: number;
  username: string;
  home_directory: string;
  storage_quota_gb: number;
  status: string;
  last_login?: string;
  last_ip: string;
  total_upload_bytes: number;
  total_download_bytes: number;
  server_name: string;
  server_host: string;
  server_port: number;
}
export interface CustomerPortalServiceEntitlement { id:number; service_type:string; service_name:string; username:string; endpoint:string; status:string; expiry_at?:string; quota_gb:number; remarks:string; password_configured:boolean }

export async function getCustomerPortalMe(): Promise<CustomerPortalMe> {
  const response = await apiClient.get<CustomerPortalMe>("/customer-portal/me");

  return response.data;
}

export async function getCustomerPortalSubscriptions(): Promise<
  CustomerPortalSubscription[]
> {
  const response = await apiClient.get<CustomerPortalSubscription[]>(
    "/customer-portal/subscription",
  );

  return response.data;
}

export async function getCustomerPortalInvoices(): Promise<
  CustomerPortalInvoice[]
> {
  const response = await apiClient.get<CustomerPortalInvoice[]>(
    "/customer-portal/invoices",
  );

  return response.data;
}

export async function getCustomerPortalPayments(): Promise<
  CustomerPortalPayment[]
> {
  const response = await apiClient.get<CustomerPortalPayment[]>(
    "/customer-portal/payments",
  );

  return response.data;
}

export async function getCustomerPortalTemporaryAccess(): Promise<
  CustomerPortalTemporaryAccess[]
> {
  const response = await apiClient.get<{
    temporary_accesses: CustomerPortalTemporaryAccess[];
  }>("/customer-portal/temporary-access");

	return response.data.temporary_accesses;
}

export async function getCustomerPortalFTPEntitlements(): Promise<CustomerPortalFTPEntitlement[]> {
  const response = await apiClient.get<{ ftp_entitlements: CustomerPortalFTPEntitlement[] }>(
    "/customer-portal/ftp-entitlements",
  );
  return response.data.ftp_entitlements;
}
export async function getCustomerPortalServiceEntitlements(): Promise<CustomerPortalServiceEntitlement[]> {
  return (await apiClient.get<{entitlements:CustomerPortalServiceEntitlement[]}>('/customer-portal/service-entitlements')).data.entitlements;
}
