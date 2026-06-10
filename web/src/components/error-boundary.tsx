import { Component, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { AlertTriangle } from 'lucide-react'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  reset() {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback
      return <DefaultErrorFallback error={this.state.error} onReset={() => this.reset()} />
    }
    return this.props.children
  }
}

function DefaultErrorFallback({ error, onReset }: { error: Error | null; onReset: () => void }) {
  return (
    <div className="flex h-dvh w-full flex-col items-center justify-center gap-4 p-6 text-center">
      <AlertTriangle className="text-destructive h-12 w-12" />
      <h1 className="text-xl font-semibold">页面出错了</h1>
      <p className="text-muted-foreground max-w-md text-sm">
        {error?.message || '发生了未知错误，请尝试刷新页面'}
      </p>
      <div className="flex gap-3">
        <Button variant="outline" onClick={() => window.location.reload()}>
          刷新页面
        </Button>
        <Button onClick={onReset}>重试</Button>
      </div>
    </div>
  )
}
