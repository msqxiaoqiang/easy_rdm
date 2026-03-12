import { type Page, type Locator } from '@playwright/test'

export class TabBarPage {
  readonly page: Page
  readonly tabList: Locator

  constructor(page: Page) {
    this.page = page
    this.tabList = page.locator('.tab-list')
  }

  getTab(name: string) {
    return this.tabList.locator('.tab-item', { hasText: name })
  }

  async switchTab(name: string) {
    await this.getTab(name).click()
  }

  async closeTab(name: string) {
    await this.getTab(name).locator('.tab-close').click()
  }

  async rightClickTab(name: string) {
    await this.getTab(name).click({ button: 'right' })
  }
}
