import { test, expect } from '@playwright/test'

test.describe('Cluster Info', () => {
  test('view cluster info', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
