<template>
  <div ref="editorRef" class="json-code-editor" />
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { EditorState } from '@codemirror/state'
import {
  EditorView,
  keymap,
  lineNumbers,
  highlightActiveLine,
  highlightActiveLineGutter,
} from '@codemirror/view'
import { json, jsonParseLinter } from '@codemirror/lang-json'
import { defaultKeymap, indentWithTab } from '@codemirror/commands'
import { linter, lintGutter } from '@codemirror/lint'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorRef = ref<HTMLElement>()
let view: EditorView | null = null

const editorTheme = EditorView.theme({
  '&': {
    height: '100%',
    fontSize: '13px',
    backgroundColor: '#fafbfc',
  },
  '.cm-scroller': {
    fontFamily: "'SF Mono', Monaco, Consolas, monospace",
    lineHeight: '1.6',
  },
  '.cm-gutters': {
    backgroundColor: '#f7f8fa',
    color: '#86909c',
    borderRight: '1px solid #e5e6eb',
  },
  '.cm-activeLineGutter': {
    backgroundColor: '#eef3ff',
    color: '#165dff',
  },
  '.cm-activeLine': {
    backgroundColor: '#f2f6ff',
  },
  '.cm-content': {
    caretColor: '#165dff',
    padding: '8px 0',
  },
  '&.cm-focused .cm-cursor': {
    borderLeftColor: '#165dff',
  },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
    backgroundColor: '#d6e4ff !important',
  },
  '.cm-lintRange-error': {
    backgroundImage: 'none',
    borderBottom: '2px wavy #f53f3f',
  },
})

onMounted(() => {
  if (!editorRef.value) return

  view = new EditorView({
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        highlightActiveLineGutter(),
        lintGutter(),
        json(),
        linter(jsonParseLinter()),
        EditorView.lineWrapping,
        editorTheme,
        keymap.of([...defaultKeymap, indentWithTab]),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            emit('update:modelValue', update.state.doc.toString())
          }
        }),
      ],
    }),
    parent: editorRef.value,
  })
})

watch(
  () => props.modelValue,
  (value) => {
    if (!view) return
    const current = view.state.doc.toString()
    if (value !== current) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: value },
      })
    }
  },
)

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})
</script>

<style scoped>
.json-code-editor {
  height: 100%;
  min-height: 420px;
  border: 1px solid #e5e6eb;
  border-radius: 8px;
  overflow: hidden;
}

.json-code-editor :deep(.cm-editor) {
  height: 100%;
}

.json-code-editor :deep(.cm-editor.cm-focused) {
  outline: none;
  border-color: #165dff;
  box-shadow: 0 0 0 2px rgba(22, 93, 255, 0.1);
}
</style>
