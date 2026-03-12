<template>
  <div class="json-code-editor" ref="containerRef">
    <div class="cm-wrapper" ref="editorRef"></div>
    <div v-if="validationMsg" :class="['json-validation', validationOk ? 'valid' : 'invalid']">
      {{ validationMsg }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, shallowRef } from 'vue'
import { EditorView, keymap, placeholder as cmPlaceholder, lineNumbers } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { json } from '@codemirror/lang-json'
import { syntaxHighlighting, HighlightStyle, bracketMatching, foldGutter, foldKeymap } from '@codemirror/language'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { linter, lintGutter, type Diagnostic } from '@codemirror/lint'
import { tags } from '@lezer/highlight'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  modelValue: string
  readonly?: boolean
  placeholder?: string
}>(), {
  readonly: false,
  placeholder: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: []
}>()

const { t } = useI18n()
const containerRef = ref<HTMLElement>()
const editorRef = ref<HTMLElement>()
const view = shallowRef<EditorView>()
const validationMsg = ref('')
const validationOk = ref(true)

// Compartment for readonly toggling
const readonlyComp = new Compartment()

// Dark theme matching app CSS variables
const editorTheme = EditorView.theme({
  '&': {
    height: '100%',
    fontSize: 'var(--app-editor-font-size, var(--font-size-sm))',
    fontFamily: 'var(--font-family-mono)',
    backgroundColor: 'var(--color-bg-1)',
    color: 'var(--color-text-1)',
  },
  '.cm-content': {
    padding: 'var(--spacing-sm) 0',
    caretColor: 'var(--color-text-1)',
  },
  '.cm-cursor': {
    borderLeftColor: 'var(--color-text-1)',
  },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
    background: 'var(--color-primary-bg, rgba(64,128,255,0.2)) !important',
  },
  '.cm-gutters': {
    backgroundColor: 'var(--color-bg-2)',
    color: 'var(--color-text-4)',
    border: 'none',
    borderRight: '1px solid var(--color-border-1)',
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'var(--color-bg-3, rgba(255,255,255,0.05))',
  },
  '.cm-activeLine': {
    backgroundColor: 'var(--color-bg-3, rgba(255,255,255,0.03))',
  },
  '.cm-matchingBracket': {
    backgroundColor: 'var(--color-primary-bg, rgba(64,128,255,0.3))',
    outline: '1px solid var(--color-primary)',
  },
  '.cm-gutterElement': {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  '.cm-lineNumbers .cm-gutterElement': {
    justifyContent: 'flex-end',
    paddingRight: '8px',
    minWidth: '32px',
  },
  '.cm-foldGutter .cm-gutterElement': {
    padding: '0 2px',
    cursor: 'pointer',
    width: '16px',
  },
  '.cm-tooltip': {
    backgroundColor: 'var(--color-bg-2)',
    border: '1px solid var(--color-border-2)',
    color: 'var(--color-text-1)',
  },
  '.cm-lint-marker': {
    width: '8px',
    height: '8px',
  },
  '.cm-lintRange-error': {
    backgroundImage: 'none',
    textDecoration: 'underline wavy var(--color-error)',
  },
  '.cm-placeholder': {
    color: 'var(--color-text-4)',
  },
})

// Syntax highlight colors for JSON
const highlightStyle = HighlightStyle.define([
  { tag: tags.propertyName, color: '#4fc1ff' },      // key — 亮青蓝
  { tag: tags.string, color: '#f0a875' },              // string value — 亮橙
  { tag: tags.number, color: '#a8e06c' },              // number — 亮绿
  { tag: tags.bool, color: '#6cb6ff' },                // boolean — 亮蓝
  { tag: tags.null, color: '#d2a8ff' },                // null — 亮紫
  { tag: tags.punctuation, color: 'var(--color-text-3)' }, // brackets, commas
])

// JSON validation linter
function jsonLinter(): (view: EditorView) => Diagnostic[] {
  return (view: EditorView) => {
    const doc = view.state.doc.toString()
    if (!doc.trim()) {
      validationMsg.value = ''
      validationOk.value = true
      return []
    }
    try {
      JSON.parse(doc)
      validationMsg.value = t('format.jsonValid')
      validationOk.value = true
      return []
    } catch (e: any) {
      const msg = e.message || 'Invalid JSON'
      // Extract position from error message (e.g., "at position 42")
      const posMatch = msg.match(/position\s+(\d+)/i)
      const pos = posMatch ? parseInt(posMatch[1], 10) : 0
      const clampedPos = Math.min(pos, view.state.doc.length)
      validationMsg.value = msg
      validationOk.value = false
      return [{
        from: clampedPos,
        to: Math.min(clampedPos + 1, view.state.doc.length),
        severity: 'error',
        message: msg,
      }]
    }
  }
}

// Suppress changes when updating from props
let suppressUpdate = false

function createEditor() {
  if (!editorRef.value) return

  const extensions = [
    lineNumbers(),
    history(),
    foldGutter(),
    bracketMatching(),
    closeBrackets(),
    json(),
    syntaxHighlighting(highlightStyle),
    editorTheme,
    lintGutter(),
    linter(jsonLinter(), { delay: 300 }),
    keymap.of([
      ...closeBracketsKeymap,
      ...defaultKeymap,
      ...historyKeymap,
      ...foldKeymap,
    ]),
    readonlyComp.of(EditorState.readOnly.of(props.readonly)),
    EditorView.updateListener.of((update) => {
      if (update.docChanged && !suppressUpdate) {
        const val = update.state.doc.toString()
        emit('update:modelValue', val)
        emit('change')
      }
    }),
  ]

  if (props.placeholder) {
    extensions.push(cmPlaceholder(props.placeholder))
  }

  const state = EditorState.create({
    doc: props.modelValue,
    extensions,
  })

  view.value = new EditorView({
    state,
    parent: editorRef.value,
  })
}

// Sync modelValue changes from parent
watch(() => props.modelValue, (newVal) => {
  if (!view.value) return
  const current = view.value.state.doc.toString()
  if (newVal !== current) {
    suppressUpdate = true
    view.value.dispatch({
      changes: { from: 0, to: view.value.state.doc.length, insert: newVal },
    })
    suppressUpdate = false
  }
})

// Sync readonly changes
watch(() => props.readonly, (val) => {
  if (!view.value) return
  view.value.dispatch({
    effects: readonlyComp.reconfigure(EditorState.readOnly.of(val)),
  })
})

onMounted(() => {
  createEditor()
})

onBeforeUnmount(() => {
  view.value?.destroy()
})

defineExpose({
  /** Format the current content */
  format() {
    if (!view.value) return
    const doc = view.value.state.doc.toString()
    try {
      const formatted = JSON.stringify(JSON.parse(doc), null, 2)
      if (formatted !== doc) {
        view.value.dispatch({
          changes: { from: 0, to: view.value.state.doc.length, insert: formatted },
        })
      }
    } catch (_e) { /* invalid JSON, skip format */ }
  },
  getView: () => view.value,
})
</script>

<style scoped>
.json-code-editor {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.cm-wrapper {
  flex: 1;
  overflow: hidden;
}

.cm-wrapper :deep(.cm-editor) {
  height: 100%;
  outline: none;
}

.cm-wrapper :deep(.cm-scroller) {
  overflow: auto;
}

.json-validation {
  flex-shrink: 0;
  padding: 2px var(--spacing-md);
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  border-top: 1px solid var(--color-border-1);
}

.json-validation.valid {
  color: var(--color-success, #00b42a);
}

.json-validation.invalid {
  color: var(--color-error, #f53f3f);
}
</style>
