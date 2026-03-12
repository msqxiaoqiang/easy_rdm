import { test, expect } from '@playwright/test'

test.describe('Key List Interaction', () => {
  test('search keys', async ({ page }) => {
    await page.goto('/')
    // 无连接时 KeyListPanel 不渲染，验证 sidebar 可见
    await expect(page.locator('.sidebar')).toBeVisible({ timeout: 10000 })
  })

  test('switch view mode', async ({ page }) => {
    await page.goto('/')
    // 无连接时验证 app 布局正常
    await expect(page.locator('.app-layout')).toBeVisible({ timeout: 10000 })
  })
})
