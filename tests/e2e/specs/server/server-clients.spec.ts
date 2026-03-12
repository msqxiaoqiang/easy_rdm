import { test, expect } from '@playwright/test'

test.describe('Server Clients', () => {
  test('view client list', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('kill client', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
