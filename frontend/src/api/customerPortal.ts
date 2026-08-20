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
