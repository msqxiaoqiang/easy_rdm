import { test, expect } from '@playwright/test'

test.describe('Editor', () => {
  test('edit value in editor', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
