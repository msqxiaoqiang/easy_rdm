import { test, expect } from '@playwright/test'
import { SidebarPage } from '../../pages/sidebar.page'

test.describe('Connection Group', () => {
  test('create group', async ({ page }) => {
    await page.goto('/')
    const sidebar = new SidebarPage(page)
    await sidebar.newGroupBtn.click()
    await expect(page.locator('.prompt-dialog, .group-input')).toBeVisible()
  })
})
