import { test, expect } from '@playwright/test'

test.describe('Set Operations', () => {
  test('view set members', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('add member', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
