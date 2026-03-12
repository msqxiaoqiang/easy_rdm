import { type Page, type Locator } from '@playwright/test'

export class SettingsPage {
  readonly page: Page
  readonly modal: Locator

  constructor(page: Page) {
    this.page = page
    this.modal = page.locator('.settings-modal')
  }

  async setTheme(theme: string) {
    await this.modal.locator('select').first().selectOption(theme)
  }

  async setLanguage(lang: string) {
    await this.modal.locator('select').nth(1).selectOption(lang)
  }

  async save() {
    await this.modal.locator('button', { hasText: /save|保存/i }).click()
  }

  async close() {
    await this.modal.locator('.close-btn').click()
  }
}
