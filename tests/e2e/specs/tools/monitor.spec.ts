import { test, expect } from '@playwright/test'

test.describe('Monitor', () => {
  test('start monitor', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
