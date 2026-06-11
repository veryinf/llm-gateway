import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Loading } from '@/components/loader';
import { PageHeader } from '@/components/page-header';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useModal } from '@/components/modal';
import { requestLogService, type RequestLogEntry } from '@/services/request-log';

export const Route = createFileRoute('/request-logs')({
  component: RequestLogsPage,
});

const PAGE_SIZE = 20;

function RequestLogsPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const { Modal, modalHandler, meta } = useModal<RequestLogEntry>();

  useEffect(() => {
    setBreadcrumbs([{ title: '请求记录' }]);
  }, []);

  const { data, isLoading } = useQuery({
    queryKey: ['request-logs', page, statusFilter, modelFilter],
    queryFn: () =>
      requestLogService.search({
        page,
        pageSize: PAGE_SIZE,
        status: statusFilter || undefined,
        model: modelFilter || undefined,
      }),
  });

  const totalPages = Math.ceil((data?.total ?? 0) / PAGE_SIZE);

  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 px-4">
          <PageHeader title="请求记录" description="查看所有 API 请求的详细记录" />

          <div className="flex items-center gap-4">
            <select
              value={statusFilter}
              onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
              className="border-input bg-background ring-ring h-9 rounded-md border px-3 text-sm"
            >
              <option value="">全部状态</option>
              <option value="success">成功</option>
              <option value="error">失败</option>
            </select>
            <input
              type="text"
              value={modelFilter}
              onChange={(e) => { setModelFilter(e.target.value); setPage(1); }}
              placeholder="按模型筛选..."
              className="border-input bg-background ring-ring h-9 w-48 rounded-md border px-3 text-sm"
            />
            <div className="text-muted-foreground ml-auto text-sm">
              共 {data?.total ?? 0} 条记录
            </div>
          </div>

          <div className="flex flex-1 flex-col gap-4">
            {isLoading ? (
              <div className="flex items-center justify-center p-8">
                <Loading size={32} />
              </div>
            ) : (
              <div className="overflow-hidden rounded-md border relative">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>时间</TableHead>
                      <TableHead>模型</TableHead>
                      <TableHead>类型</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>延迟</TableHead>
                      <TableHead>Token</TableHead>
                      <TableHead>成本</TableHead>
                      <TableHead>IP</TableHead>
                      <TableHead className="w-16">详情</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(data?.list ?? []).length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={9} className="text-muted-foreground text-center">
                          暂无请求记录
                        </TableCell>
                      </TableRow>
                    ) : (
                      (data?.list ?? []).map((log, i) => (
                        <TableRow key={`${log.trace_id}-${i}`}>
                          <TableCell className="whitespace-nowrap text-xs">
                            {new Date(log.created_at).toLocaleString()}
                          </TableCell>
                          <TableCell className="font-mono text-xs">{log.model_name}</TableCell>
                          <TableCell>
                            <Badge variant={log.is_stream ? 'secondary' : 'default'}>
                              {log.is_stream ? '流式' : '非流式'}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Badge variant={log.status_code === 200 ? 'default' : 'destructive'}>
                              {log.status_code}
                            </Badge>
                          </TableCell>
                          <TableCell>{log.latency_ms}ms</TableCell>
                          <TableCell>{log.prompt_tokens + log.completion_tokens}</TableCell>
                          <TableCell>¥{log.cost.toFixed(4)}</TableCell>
                          <TableCell className="text-xs">{log.ip_address}</TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => modalHandler.open('请求详情', '', log)}
                            >
                              查看
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            )}

            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 mt-auto">
                <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                  上一页
                </Button>
                <span className="text-muted-foreground text-sm">{page} / {totalPages}</span>
                <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
                  下一页
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>

      <Modal>
        {meta && <RequestLogDetail log={meta} />}
      </Modal>
    </div>
  );
}

function RequestLogDetail({ log }: { log: RequestLogEntry }) {
  const { data: chunks, isLoading: chunksLoading } = useQuery({
    queryKey: ['request-chunks', log.trace_id],
    queryFn: () => requestLogService.fetchChunks(log.trace_id),
    enabled: log.is_stream && log.is_detail,
  });

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-2 md:grid-cols-2">
        <DetailRow label="Trace ID" value={log.trace_id} />
        <DetailRow label="模型" value={log.model_name} />
        <DetailRow label="类型" value={log.is_stream ? '流式' : '非流式'} />
        <DetailRow label="状态码" value={String(log.status_code)} />
        <DetailRow label="延迟" value={`${log.latency_ms}ms`} />
        <DetailRow label="Prompt Tokens" value={String(log.prompt_tokens)} />
        <DetailRow label="Completion Tokens" value={String(log.completion_tokens)} />
        <DetailRow label="成本" value={`¥${log.cost.toFixed(4)}`} />
        <DetailRow label="IP 地址" value={log.ip_address} />
        <DetailRow label="User ID" value={String(log.user_id)} />
      </div>

      {log.error_message && (
        <div>
          <span className="text-muted-foreground text-sm">错误信息：</span>
          <pre className="bg-muted mt-1 overflow-auto rounded p-3 text-sm">{log.error_message}</pre>
        </div>
      )}

      {log.request_body && (
        <div>
          <span className="text-muted-foreground text-sm">请求 Body：</span>
          <pre className="bg-muted mt-1 overflow-auto rounded p-3 text-sm max-h-60">
            {formatJson(log.request_body)}
          </pre>
        </div>
      )}

      {log.response_body && (
        <div>
          <span className="text-muted-foreground text-sm">响应 Body：</span>
          <pre className="bg-muted mt-1 overflow-auto rounded p-3 text-sm max-h-60">
            {formatJson(log.response_body)}
          </pre>
        </div>
      )}

      {log.is_stream && log.is_detail && (
        <div>
          <span className="text-muted-foreground text-sm">流式响应：</span>
          {chunksLoading ? (
            <div className="mt-1 flex items-center gap-2">
              <Loading size={16} /> <span className="text-sm">加载中...</span>
            </div>
          ) : chunks && chunks.length > 0 ? (
            <>
              <div className="bg-muted mt-1 overflow-auto rounded p-3 text-sm max-h-40">
                {assembleStreamContent(chunks)}
              </div>
              <details className="mt-2">
                <summary className="text-muted-foreground cursor-pointer text-xs">
                  原始 Chunks（共 {chunks.length} 个）
                </summary>
                <div className="bg-muted mt-1 max-h-60 overflow-auto rounded p-3">
                  {chunks.slice(0, 100).map((chunk) => (
                    <pre key={chunk.chunk_index} className="mt-1 border-b border-gray-700 pb-1 text-xs break-all">
                      [{chunk.chunk_index}] {formatChunkData(chunk.chunk_data)}
                    </pre>
                  ))}
                  {chunks.length > 100 && (
                    <span className="text-muted-foreground mt-1 text-xs">
                      ... 还有 {chunks.length - 100} 个 chunk
                    </span>
                  )}
                </div>
              </details>
            </>
          ) : (
            <p className="text-muted-foreground mt-1 text-sm">无 chunk 数据</p>
          )}
        </div>
      )}
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="text-muted-foreground text-sm">{label}</span>
      <p className="text-sm break-all">{value}</p>
    </div>
  );
}

function formatJson(str: string): string {
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

function assembleStreamContent(chunks: { chunk_data: string }[]): string {
  const parts: string[] = [];
  for (const chunk of chunks) {
    try {
      const data = JSON.parse(chunk.chunk_data);
      const delta = data.choices?.[0]?.delta;
      if (delta?.content) {
        parts.push(delta.content);
      }
    } catch {
      // skip unparseable chunks
    }
  }
  return parts.join('');
}

function formatChunkData(raw: string): string {
  try {
    const data = JSON.parse(raw);
    const delta = data.choices?.[0]?.delta;
    if (delta?.content) {
      return `content: ${JSON.stringify(delta.content)}`;
    }
    if (delta?.role) {
      return `role: ${delta.role}`;
    }
    if (data.usage) {
      return `usage: prompt=${data.usage.prompt_tokens} completion=${data.usage.completion_tokens}`;
    }
    return JSON.stringify(data, null, 0);
  } catch {
    return raw;
  }
}
