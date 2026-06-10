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
import { auditService, type AuditLog } from '@/services/audit';

export const Route = createFileRoute('/audit-logs')({
  component: AuditLogsPage,
});

const PAGE_SIZE = 20;

function AuditLogsPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const { Modal, modalHandler, meta } = useModal<AuditLog>();

  useEffect(() => {
    setBreadcrumbs([{ title: '统计' }, { title: '审计日志' }]);
  }, []);

  const { data, isLoading } = useQuery({
    queryKey: ['audit-logs', page, statusFilter, modelFilter],
    queryFn: () =>
      auditService.search({
        page,
        pageSize: PAGE_SIZE,
        status: statusFilter || undefined,
        model: modelFilter || undefined,
      }),
  });

  const totalPages = Math.ceil((data?.total ?? 0) / PAGE_SIZE);

  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <PageHeader title="审计日志" description="查看所有 API 请求的审计记录" />

      {/* Filters */}
      <div className="flex items-center gap-4">
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value);
            setPage(1);
          }}
          className="border-input bg-background ring-ring h-9 rounded-md border px-3 text-sm"
        >
          <option value="">全部状态</option>
          <option value="success">成功</option>
          <option value="error">失败</option>
        </select>
        <input
          type="text"
          value={modelFilter}
          onChange={(e) => {
            setModelFilter(e.target.value);
            setPage(1);
          }}
          placeholder="按模型筛选..."
          className="border-input bg-background ring-ring h-9 w-48 rounded-md border px-3 text-sm"
        />
        <div className="text-muted-foreground ml-auto text-sm">
          共 {data?.total ?? 0} 条记录
        </div>
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="flex items-center justify-center p-8">
          <Loading size={32} />
        </div>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>时间</TableHead>
                <TableHead>模型</TableHead>
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
                  <TableCell colSpan={8} className="text-muted-foreground text-center">
                    暂无审计日志
                  </TableCell>
                </TableRow>
              ) : (
                (data?.list ?? []).map((log) => (
                  <TableRow key={log.id}>
                    <TableCell className="whitespace-nowrap text-xs">
                      {new Date(log.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="font-mono text-xs">{log.model_name}</TableCell>
                    <TableCell>
                      <Badge variant={log.status_code === 200 ? 'default' : 'destructive'}>
                        {log.status_code}
                      </Badge>
                    </TableCell>
                    <TableCell>{log.latency_ms}ms</TableCell>
                    <TableCell>
                      {log.prompt_tokens + log.completion_tokens}
                    </TableCell>
                    <TableCell>¥{log.cost.toFixed(4)}</TableCell>
                    <TableCell className="text-xs">{log.ip_address}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => modalHandler.open('审计日志详情', '', log)}
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

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            上一页
          </Button>
          <span className="text-muted-foreground text-sm">
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

      {/* Detail Modal */}
      <Modal>
        {meta && <AuditLogDetail log={meta} />}
      </Modal>
    </div>
  );
}

function AuditLogDetail({ log }: { log: AuditLog }) {
  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-2 md:grid-cols-2">
        <DetailRow label="Trace ID" value={log.trace_id} />
        <DetailRow label="模型" value={log.model_name} />
        <DetailRow label="状态码" value={String(log.status_code)} />
        <DetailRow label="延迟" value={`${log.latency_ms}ms`} />
        <DetailRow label="Prompt Tokens" value={String(log.prompt_tokens)} />
        <DetailRow label="Completion Tokens" value={String(log.completion_tokens)} />
        <DetailRow label="成本" value={`¥${log.cost.toFixed(4)}`} />
        <DetailRow label="IP 地址" value={log.ip_address} />
        <DetailRow label="User ID" value={String(log.user_id)} />
        <DetailRow label="API Key ID" value={String(log.api_key_id)} />
      </div>
      {log.error_message && (
        <div>
          <span className="text-muted-foreground text-sm">错误信息：</span>
          <pre className="bg-muted mt-1 overflow-auto rounded p-3 text-sm">{log.error_message}</pre>
        </div>
      )}
      {log.request_summary && (
        <div>
          <span className="text-muted-foreground text-sm">请求摘要：</span>
          <pre className="bg-muted mt-1 overflow-auto rounded p-3 text-sm max-h-40">
            {formatJson(log.request_summary)}
          </pre>
        </div>
      )}
      {log.response_summary && (
        <div>
          <span className="text-muted-foreground text-sm">响应摘要：</span>
          <pre className="bg-muted mt-1 overflow-auto rounded p-3 text-sm max-h-40">{log.response_summary}</pre>
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
