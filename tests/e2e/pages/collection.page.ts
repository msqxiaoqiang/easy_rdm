import { type Page, type Locator } from '@playwright/test'

export class CollectionPage {
  readonly page: Page
  readonly table: Locator
  readonly searchInput: Locator

  constructor(page: Page) {
    this.page = page
    this.table = page.locator('.collection-table')
    this.searchInput = page.locator('.collection-search input')
  }

  async addField() {
    await this.page.locator('.collection-toolbar button', { hasText: /add|添加/i }).click()
  }

  async filter(pattern: string) {
    await this.searchInput.fill(pattern)
    await this.searchInput.press('Enter')
  }

  getRow(text: string) {
    return this.table.locator('tr', { hasText: text })
  }

  async deleteRow(text: string) {
    await this.getRow(text).locator('.delete-btn').click()
  }
}
