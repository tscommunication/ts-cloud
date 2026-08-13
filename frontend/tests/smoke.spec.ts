import { expect, test } from '@playwright/test'

test('renders the login page without runtime errors', async ({ page }) => {
  const runtimeErrors: string[] = []
  page.on('pageerror', (error) => runtimeErrors.push(error.message))

  await page.goto('/')

  await expect(page).toHaveURL(/\/login$/)
  await expect(page.getByRole('heading', { name: 'TS-Cloud' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible()
  expect(runtimeErrors).toEqual([])
})

test('renders a recovery page for an unknown route', async ({ page }) => {
  await page.goto('/this-route-does-not-exist')

  await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Go to dashboard' })).toBeVisible()
})
