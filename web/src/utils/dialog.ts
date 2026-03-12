import { ref } from 'vue'

// ========== Confirm State ==========

export const confirmState = ref({
  visible: false,
  title: '',
  content: '',
  resolve: null as ((val: boolean) => void) | null,
})

export function resolveConfirm(value: boolean) {
  const { resolve } = confirmState.value
  confirmState.value.visible = false
  resolve?.(value)
}

export function gmConfirm(content: string, title?: string): Promise<boolean> {
  return new Promise((resolve) => {
    confirmState.value = {
      visible: true,
      title: title || '',
      content,
      resolve,
    }
  })
}

// ========== Prompt State ==========

export const promptState = ref({
  visible: false,
  title: '',
  label: '',
  value: '',
  placeholder: '',
  resolve: null as ((val: string | null) => void) | null,
})

export function resolvePrompt(value: string | null) {
  const { resolve } = promptState.value
  promptState.value.visible = false
  resolve?.(value)
}

export function gmPrompt(label: string, defaultValue = '', placeholder = ''): Promise<string | null> {
  return new Promise((resolve) => {
    promptState.value = {
      visible: true,
      title: label,
      label: '',
      value: defaultValue,
      placeholder,
      resolve,
    }
  })
}
