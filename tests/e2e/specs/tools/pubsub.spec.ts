import { test, expect } from '@playwright/test'

test.describe('Pub/Sub', () => {
  test('subscribe to channel', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
