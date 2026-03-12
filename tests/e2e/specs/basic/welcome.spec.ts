import { test, expect } from '@playwright/test'
import fs from 'fs'
import path from 'path'

const dataDir = path.resolve(__dirname, '../../../../backend/data')

test.describe('Welcome Page', () => {
  test.beforeEach(() => {
    fs.writeFileSync(path.join(dataDir, 'connections.json'), '[]')
    fs.writeFileSync(path.join(dataDir, 'session.json'), '{}')
  })

  test('shows welcome page when no connections', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('.welcome-page')).toBeVisible({ timeout: 10000 })
  })

  test('shows create connection button', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('.welcome-page .btn-primary')).toBeVisible({ timeout: 10000 })
  })
})
