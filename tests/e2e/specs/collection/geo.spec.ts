import { test, expect } from '@playwright/test'

test.describe('Geo Operations', () => {
  test('view geo members', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
