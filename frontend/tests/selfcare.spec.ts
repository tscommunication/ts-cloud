import { expect, test, type Page } from '@playwright/test'

const customerUser = {
  id: 17,
  username: 'CUS-000017',
  role: 'customer',
  customer_id: 17,
}

async function authenticateAsCustomer(page: Page) {
  await page.addInitScript((user) => {
    localStorage.setItem('access_token', 'customer-test-token')
    localStorage.setItem('user', JSON.stringify(user))
  }, customerUser)
}

async function authenticateAsAdmin(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('access_token', 'admin-test-token')
    localStorage.setItem(
      'user',
      JSON.stringify({ id: 1, username: 'admin', role: 'admin' }),
    )
  })
}

async function mockCustomerPortal(page: Page) {
  const responses: Record<string, unknown> = {
    me: {
      id: 17,
      customer_code: 'CUS-000017',
      full_name: 'Rahim Uddin',
      mobile: '01700000017',
      email: 'rahim@example.com',
      status: 'ACTIVE',
      billing_day: 10,
      present_address: 'Magura',
    },
    subscription: [
      {
        id: 31,
        subscription_code: 'SUB-000031',
        package_id: 4,
        activation_date: '2026-01-10T00:00:00Z',
        billing_day: 10,
        next_billing_date: '2026-09-10T00:00:00Z',
        expiry_date: '2026-09-09T00:00:00Z',
        status: 'ACTIVE',
        pppoe_username: 'rahim-pppoe',
        last_payment_date: '2026-08-10T00:00:00Z',
        last_paid_amount: 800,
        due_amount: 200,
      },
    ],
    invoices: [
      {
        id: 44,
        invoice_no: 'INV-000044',
        subscription_id: 31,
        package_id: 4,
        bill_month: 8,
        bill_year: 2026,
        issue_date: '2026-08-01T00:00:00Z',
        due_date: '2026-08-10T00:00:00Z',
        package_price: 1000,
        discount: 0,
        vat: 0,
        total_amount: 1000,
        paid_amount: 800,
        due_amount: 200,
        status: 'PARTIAL',
      },
    ],
    payments: [
      {
        id: 52,
        receipt_no: 'RCP-000052',
        invoice_id: 44,
        subscription_id: 31,
        payment_date: '2026-08-10T00:00:00Z',
        amount: 800,
        method: 'CASH',
        transaction_id: 'TX-52',
        status: 'COMPLETED',
      },
    ],
  }

  await page.route('**/api/v1/customer-portal/**', async (route) => {
    const endpoint = new URL(route.request().url()).pathname.split('/').at(-1)
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(responses[endpoint || '']),
    })
  })
}

test('redirects an unauthenticated customer to the SelfCare login', async ({
  page,
}) => {
  await page.goto('/selfcare')

  await expect(page).toHaveURL(/\/selfcare\/login$/)
  await expect(
    page.getByRole('heading', { name: 'Customer SelfCare' }),
  ).toBeVisible()
})

test('renders customer account, subscription, invoice, and payment data', async ({
  page,
}) => {
  await authenticateAsCustomer(page)
  await mockCustomerPortal(page)

  await page.goto('/selfcare')

  await expect(page.getByText('Welcome, Rahim Uddin')).toBeVisible()
  await expect(page.getByText('SUB-000031')).toBeVisible()
  await expect(page.getByText('INV-000044')).toBeVisible()
  await expect(page.getByText('RCP-000052')).toBeVisible()
  await expect(page.getByText('rahim-pppoe')).toBeVisible()
})

test('redirects a customer away from the staff dashboard', async ({ page }) => {
  await authenticateAsCustomer(page)
  await mockCustomerPortal(page)

  await page.goto('/dashboard')

  await expect(page).toHaveURL(/\/selfcare$/)
  await expect(page.getByText('Welcome, Rahim Uddin')).toBeVisible()
})

test('redirects a staff user away from SelfCare', async ({ page }) => {
  await authenticateAsAdmin(page)

  await page.goto('/selfcare')

  await expect(page).toHaveURL(/\/login$/)
})

test('shows an API error and clears customer authentication on logout', async ({
  page,
}) => {
  await authenticateAsCustomer(page)
  await page.route('**/api/v1/customer-portal/**', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'Portal temporarily unavailable' }),
    })
  })

  await page.goto('/selfcare')

  await expect(page.getByText('Portal temporarily unavailable')).toBeVisible()
  await page.getByRole('button', { name: 'Sign Out' }).click()
  await expect(page).toHaveURL(/\/selfcare\/login$/)
  await expect(
    page.evaluate(() => ({
      token: localStorage.getItem('access_token'),
      user: localStorage.getItem('user'),
    })),
  ).resolves.toEqual({ token: null, user: null })
})
