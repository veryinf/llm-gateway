import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { MonacoEditor } from '@/components/monaco-editor'
import client from '@/api/client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import dayjs from 'dayjs'
import { PageHeader } from '@/components/page-header'

interface AuditLog {
  id: number
  trace_id: string
  user_id: number
  api_key_id: number
  model_name: string
  request_summary: string
  response_summary: string
  error_message: string
  prompt_tokens: number
  completion_tokens: number
  status_code: number
  latency_ms: number
  cost: number
  ip_address: string
  user_agent: string
  created_at: string
}

interface AuditListResponse {
  list: AuditLog[]
  total: number
}

export const Route = createFileRoute('/audit')({
  component: AuditPage,
})

function AuditPage() {
  const [page, setPage] = useState(1)
  const [username, setUsername] = useState('')
  const [model, setModel] = useState('')
  const [statusCode, setStatusCode] = useState('')
  const [detailOpen, setDetailOpen] = useState(false)
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null)

  const { data, isLoading } = useQuery<AuditListResponse>({
    queryKey: ['audit-logs', page, username, model, statusCode],
    queryFn: () =>
      client.get('/audit/logs', {
        params: {
          page,
          pageSize: 15,
          username: username || undefined,
          model: model || undefined,
          status_code: statusCode || undefined,
        },
      }) as Promise<AuditListResponse>,
  })

  const totalPages = data ? Math.ceil(data.total / 15) : 0

  const handleViewDetail = (log: AuditLog) => {
    setSelectedLog(log)
    setDetailOpen(true)
  }

  const formatJson = (str: string) => {
    if (!str) return '{}'
    try {
      return JSON.stringify(JSON.parse(str), null, 2)
    } catch {
      return str
    }
  }

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:py-6">
      <div className="px-4 lg:px-6">
        <PageHeader title="审计日志" description="查看 API 调用记录和请求详情" />
      </div>
      <Card className="mx-4 lg:mx-6">
        <CardContent>
          {/* Filters */}
          <div className="flex flex-wrap items-end gap-4 mb-4">
            <div className="space-y-1">
              <label className="text-sm font-medium">用户名</label>
              <Input
                value={username}
                onChange={(e) => { setUsername(e.target.value); setPage(1) }}
                placeholder="筛选用户名"
                className="w-40"
              />
            </div>
            <div className="space-y-1">
              <label className="text-sm font-medium">模型</label>
              <Input
                value={model}
                onChange={(e) => { setModel(e.target.value); setPage(1) }}
                placeholder="筛选模型"
                className="w-40"
              />
            </div>
            <div className="space-y-1">
              <label className="text-sm font-medium">状态码</label>
              <Select
                value={statusCode}
                onChange={(e) => { setStatusCode(e.target.value); setPage(1) }}
              >
                <option value="">全部</option>
                <option value="200">200</option>
                <option value="400">400</option>
                <option value="401">401</option>
                <option value="403">403</option>
                <option value="429">429</option>
                <option value="500">500</option>
              </Select>
            </div>
          </div>

          {isLoading ? (
            <div className="space-y-2">
              {[...Array(10)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>时间</TableHead>
                  <TableHead>用户</TableHead>
                  <TableHead>模型</TableHead>
                  <TableHead>状态码</TableHead>
                  <TableHead>延迟</TableHead>
                  <TableHead>费用</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.list?.map((log) => (
                  <TableRow key={log.id}>
                    <TableCell className="text-xs">
                      {dayjs(log.created_at).format('YYYY-MM-DD HH:mm:ss')}
                    </TableCell>
                    <TableCell>{log.user_id}</TableCell>
                    <TableCell>{log.model_name}</TableCell>
                    <TableCell>
                      <Badge
                        variant={log.status_code < 400 ? 'default' : log.status_code >= 500 ? 'destructive' : 'outline'}
                      >
                        {log.status_code}
                      </Badge>
                    </TableCell>
                    <TableCell>{log.latency_ms}ms</TableCell>
                    <TableCell>${(log.cost ?? 0).toFixed(4)}</TableCell>
                    <TableCell>
                      <Button variant="ghost" size="sm" onClick={() => handleViewDetail(log)}>
                        详情
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
                {(!data?.list || data.list.length === 0) && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
                      暂无日志
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-4">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                上一页
              </Button>
              <span className="text-sm text-muted-foreground">
                {page} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                下一页
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Detail Modal */}
      <Dialog open={detailOpen} onOpenChange={setDetailOpen} size="full">
        <DialogContent className="h-[80vh] flex flex-col overflow-hidden">
          <DialogHeader>
            <DialogTitle>审计日志详情</DialogTitle>
          </DialogHeader>
          {selectedLog && (
            <Tabs defaultValue="info" className="flex-1 flex flex-col min-h-0">
              <TabsList>
                <TabsTrigger value="info">基础信息</TabsTrigger>
                <TabsTrigger value="request">请求</TabsTrigger>
                <TabsTrigger value="response">响应</TabsTrigger>
              </TabsList>

              <TabsContent value="info" className="flex-1 overflow-y-auto mt-4">
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div><span className="text-muted-foreground">Trace ID</span><p className="font-mono text-xs">{selectedLog.trace_id}</p></div>
                  <div><span className="text-muted-foreground">用户 ID</span><p>{selectedLog.user_id}</p></div>
                  <div><span className="text-muted-foreground">模型</span><p>{selectedLog.model_name}</p></div>
                  <div><span className="text-muted-foreground">状态码</span><p><Badge variant={selectedLog.status_code < 400 ? 'default' : 'destructive'}>{selectedLog.status_code}</Badge></p></div>
                  <div><span className="text-muted-foreground">延迟</span><p>{selectedLog.latency_ms}ms</p></div>
                  <div><span className="text-muted-foreground">费用</span><p>${(selectedLog.cost ?? 0).toFixed(6)}</p></div>
                  <div><span className="text-muted-foreground">Token</span><p>P:{selectedLog.prompt_tokens} C:{selectedLog.completion_tokens}</p></div>
                  <div><span className="text-muted-foreground">时间</span><p>{dayjs(selectedLog.created_at).format('YYYY-MM-DD HH:mm:ss')}</p></div>
                  <div><span className="text-muted-foreground">IP</span><p>{selectedLog.ip_address}</p></div>
                  <div className="col-span-2"><span className="text-muted-foreground">UA</span><p className="text-xs break-all">{selectedLog.user_agent}</p></div>
                </div>
                {selectedLog.error_message && (
                  <div className="mt-4">
                    <span className="text-sm font-medium text-red-500">错误信息</span>
                    <pre className="mt-1 bg-red-50 dark:bg-red-900/20 p-3 rounded text-xs overflow-auto max-h-32">{selectedLog.error_message}</pre>
                  </div>
                )}
              </TabsContent>

              <TabsContent value="request" className="flex-1 mt-4 min-h-0">
                <MonacoEditor language="json" value={formatJson(selectedLog.request_summary)} />
              </TabsContent>

              <TabsContent value="response" className="flex-1 mt-4 min-h-0">
                <MonacoEditor language="json" value={formatJson(selectedLog.response_summary)} />
              </TabsContent>
            </Tabs>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
