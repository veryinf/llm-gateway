import { useRef, useEffect } from 'react'
import type * as Monaco from 'monaco-editor'

// Vite 原生 Worker 导入，确保版本匹配且 worker 完整加载
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'

if (typeof self !== 'undefined') {
  ;(self as any).MonacoEnvironment = {
    getWorker(_workerId: string, label: string) {
      switch (label) {
        case 'json':
          return new jsonWorker()
        case 'css':
        case 'scss':
        case 'less':
          return new cssWorker()
        case 'html':
        case 'handlebars':
        case 'razor':
          return new htmlWorker()
        case 'typescript':
        case 'javascript':
          return new tsWorker()
        default:
          return new editorWorker()
      }
    },
  }
}

let monacoLoading: Promise<typeof Monaco> | null = null

/** 格式化 JSON 字符串（带缩进），非 JSON 内容原样返回 */
function formatJSON(raw: string): string {
  try {
    const parsed = JSON.parse(raw)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return raw
  }
}

interface MonacoEditorProps {
  value: string
  language?: string
  className?: string
}

export function MonacoEditor({ value, language = 'json', className }: MonacoEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null)
  const modelRef = useRef<Monaco.editor.ITextModel | null>(null)

  useEffect(() => {
    let disposed = false
    if (!monacoLoading) {
      monacoLoading = import('monaco-editor')
    }
    monacoLoading.then((monaco) => {
      if (disposed || !containerRef.current) return

      const formatted = formatJSON(value)
      const model = monaco.editor.createModel(formatted, language)
      modelRef.current = model

      editorRef.current = monaco.editor.create(containerRef.current, {
        model,
        readOnly: true,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        lineNumbers: 'on',
        folding: true,
        foldingStrategy: 'auto',
        automaticLayout: true,
        bracketPairColorization: { enabled: true },
        guides: { bracketPairs: true },
        tabSize: 2,
        theme: 'vs-dark',
      })
    })

    return () => {
      disposed = true
      editorRef.current?.dispose()
      modelRef.current?.dispose()
      editorRef.current = null
      modelRef.current = null
    }
  }, [])

  useEffect(() => {
    if (modelRef.current) {
      const formatted = formatJSON(value)
      const currentValue = modelRef.current.getValue()
      if (currentValue !== formatted) {
        modelRef.current.setValue(formatted)
      }
    }
  }, [value])

  return <div ref={containerRef} className={className} style={{ height: '100%', width: '100%' }} />
}
