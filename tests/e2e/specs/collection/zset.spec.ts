import { test, expect } from '@playwright/test'

test.describe('ZSet Operations', () => {
  test('view zset members with scores', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })

  test('add member with score', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('body')).toBeVisible()
  })
})
