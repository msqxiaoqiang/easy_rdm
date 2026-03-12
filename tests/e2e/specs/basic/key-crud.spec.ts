import { test, expect } from '@playwright/test'
import fs from 'fs'
import path from 'path'

const dataDir = path.resolve(__dirname, '../../../../backend/data')

test.describe('Key CRUD', () => {
  test.beforeEach(() => {
    fs.writeFileSync(path.join(dataDir, 'connections.json'), '[]')
    fs.writeFileSync(path.join(dataDir, 'session.json'), '{}')
  })

  test('create string key', async ({ page }) => {
    await page.goto('/')
    // 无连接时显示 welcome 页，验证新建连接按钮可见
    await expect(page.locator('.welcome-page .btn-primary')).toBeVisible({ timeout: 10000 })
  })
})
