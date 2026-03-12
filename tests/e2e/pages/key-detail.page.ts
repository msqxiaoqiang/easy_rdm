import { type Page, type Locator } from '@playwright/test'

export class KeyDetailPage {
  readonly page: Page
  readonly typeBadge: Locator
  readonly keyName: Locator
  readonly ttlDisplay: Locator

  constructor(page: Page) {
    this.page = page
    this.typeBadge = page.locator('.type-badge')
    this.keyName = page.locator('.key-name-display')
    this.ttlDisplay = page.locator('.ttl-display')
  }

  async refresh() {
    await this.page.locator('.info-actions .action-btn').nth(0).click()
  }

  async copy() {
    await this.page.locator('.info-actions .action-btn').nth(1).click()
  }

  async rename() {
    await this.page.locator('.info-actions .action-btn').nth(2).click()
  }

  async setTTL() {
    await this.page.locator('.info-actions .action-btn').nth(3).click()
  }

  async delete() {
    await this.page.locator('.info-actions .action-btn.danger').click()
  }

  async switchFormat(format: string) {
    await this.page.locator('.view-select').first().selectOption(format)
  }

  async save() {
    await this.page.locator('.save-btn').click()
  }
}
