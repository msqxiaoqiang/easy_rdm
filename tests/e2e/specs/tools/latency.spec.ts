import { test, expect } from '@playwright/test'

test.describe('Latency Diagnostics', () => {
  test('open latency view', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
