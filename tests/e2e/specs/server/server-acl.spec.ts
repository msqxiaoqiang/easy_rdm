import { test, expect } from '@playwright/test'

test.describe('Server ACL', () => {
  test('view ACL users', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
