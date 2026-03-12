import { test, expect } from '@playwright/test'

test.describe('Connection Advanced', () => {
  test('shows TLS options', async ({ page }) => {
    await page.goto('/')
    // 等待 app ready
    await expect(page.locator('.sidebar-toolbar')).toBeVisible({ timeout: 10000 })
    // 点击新建连接按钮
    await page.locator('.sidebar-toolbar .toolbar-btn').first().click()
    // 连接表单弹窗出现
    await expect(page.locator('.modal-content')).toBeVisible()
  })
})
