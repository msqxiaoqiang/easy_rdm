import { test, expect } from '@playwright/test'

test.describe('Sentinel Info', () => {
  test('view sentinel info', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
