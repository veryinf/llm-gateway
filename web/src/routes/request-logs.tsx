import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import JsonView from '@uiw/react-json-view';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Loading } from '@/components/loader';
import { PageHeader } from '@/components/page-header';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useModal } from '@/components/modal';
import { Descriptions, type DescriptionsItem } from '@/components/descriptions';
import {
  requestLogService,
  requestDetailService,
  requestChunkService,
  type RequestLog,
  type RequestDetail,
  type RequestChunk,
} from '@/services/request-log';
import type { API } from '@/typings';

export const Route = createFileRoute('/request-logs')({
  component: RequestLogsPage,
});

const PAGE_SIZE = 20;

const apiTypeLabels: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
};

const passthroughLabels: Record<string, string> = {
  none: '正常路由',
  user: '用户级透传',
  provider: '提供商级透传',
};

function RequestLogsPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const { Modal, modalHandler, meta } = useModal<{ log: RequestLog; detail?: RequestDetail }>();

  useEffect(() => {
    setBreadcrumbs([{ title: '请求记录' }]);
  }, []);

  const searchParams: API.SearchParams = {
    pagination: { index: page, size: PAGE_SIZE },
    filters: [
      ...(statusFilter ? [{ field: 'status_code', value: statusFilter === 'success' ? 200 : 500 }] : []),
      ...(modelFilter ? [{ field: 'user_model', value: modelFilter }] : []),
    ],
  };

  const { data, isLoading } = useQuery({
    queryKey: ['request-logs', page, statusFilter, modelFilter],
    queryFn: () => requestLogService.search(searchParams),
  });

  const logs = data?.dataSet ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / PAGE_SIZE);

  const handleViewDetail = async (log: RequestLog) => {
    let detail: RequestDetail | undefined;
    if (log.isDetail) {
      detail = await requestDetailService.fetch(log.traceId);
    }
    modalHandler.open('请求详情', '', { log, detail });
  };

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
              共 {total} 条记录
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
                      <TableHead>用户模型</TableHead>
                      <TableHead>提供商模型</TableHead>
                      <TableHead>用户 API</TableHead>
                      <TableHead>提供商 API</TableHead>
                      <TableHead>摘要</TableHead>
                      <TableHead>类型</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>耗时</TableHead>
                      <TableHead>Token</TableHead>
                      <TableHead>IP</TableHead>
                      <TableHead className="w-16">详情</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {logs.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={12} className="text-muted-foreground text-center">
                          暂无请求记录
                        </TableCell>
                      </TableRow>
                    ) : (
                      logs.map((log, i) => (
                        <TableRow key={`${log.traceId}-${i}`}>
                          <TableCell className="whitespace-nowrap text-xs">
                            {new Date(log.createdAt).toLocaleString()}
                          </TableCell>
                          <TableCell className="font-mono text-xs">{log.userModel}</TableCell>
                          <TableCell className="font-mono text-xs">{log.providerModel}</TableCell>
                          <TableCell>
                            <Badge variant={log.userApiType === 'openai' ? 'default' : 'secondary'}>
                              {apiTypeLabels[log.userApiType] ?? log.userApiType}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Badge variant={log.providerApiType === 'openai' ? 'default' : 'secondary'}>
                              {apiTypeLabels[log.providerApiType] ?? log.providerApiType}
                            </Badge>
                          </TableCell>
                          <TableCell className="max-w-[200px] truncate text-xs">{log.summary || '-'}</TableCell>
                          <TableCell>
                            <Badge variant={log.isStream ? 'secondary' : 'default'}>
                              {log.isStream ? '流式' : '非流式'}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Badge variant={log.statusCode === 200 ? 'default' : 'destructive'}>
                              {log.statusCode}
                            </Badge>
                          </TableCell>
                          <TableCell>{log.duration}ms</TableCell>
                          <TableCell>
                            {log.promptTokens + log.completionTokens}
                            {log.cachedTokens > 0 && (
                              <span className="text-muted-foreground ml-1 text-xs">({log.cachedTokens} 缓存)</span>
                            )}
                          </TableCell>
                          <TableCell className="text-xs">{log.ipAddress}</TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleViewDetail(log)}
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
        {meta && <RequestLogDetail log={meta.log} detail={meta.detail} />}
      </Modal>
    </div>
  );
}

function RequestLogDetail({ log, detail }: { log: RequestLog; detail?: RequestDetail }) {
  const { data: chunks, isLoading: chunksLoading } = useQuery({
    queryKey: ['request-chunks', log.traceId],
    queryFn: () => requestChunkService.fetch(log.traceId),
    enabled: log.isStream && log.isDetail,
  });

  const basicItems: DescriptionsItem[] = [
    { label: 'Trace ID', value: <code className="text-xs">{log.traceId}</code> },
    { label: '时间', value: new Date(log.createdAt).toLocaleString() },
    { label: '用户模型', value: <code className="text-xs">{log.userModel}</code> },
    { label: '提供商模型', value: <code className="text-xs">{log.providerModel}</code> },
    { label: '用户 API', value: apiTypeLabels[log.userApiType] ?? log.userApiType },
    { label: '提供商 API', value: apiTypeLabels[log.providerApiType] ?? log.providerApiType },
    { label: '透传级别', value: passthroughLabels[log.passthroughLevel] ?? log.passthroughLevel },
    { label: '类型', value: log.isStream ? '流式' : '非流式' },
    { label: '状态码', value: String(log.statusCode) },
    { label: '耗时', value: `${log.duration}ms` },
    { label: '详细日志', value: log.isDetail ? '是' : '否' },
    { label: '摘要', value: log.summary || '-', span: 2 },
  ];

  const tokenItems: DescriptionsItem[] = [
    { label: 'Prompt', value: String(log.promptTokens) },
    { label: 'Completion', value: String(log.completionTokens) },
    { label: 'Reasoning', value: String(log.reasoningTokens) },
    { label: 'Cached', value: String(log.cachedTokens) },
    { label: 'Total', value: String(log.totalTokens) },
  ];

  const otherItems: DescriptionsItem[] = [
    { label: 'IP 地址', value: log.ipAddress },
    { label: 'User Agent', value: log.userAgent || '-', span: 2 },
    { label: 'User ID', value: String(log.userId) },
    { label: 'API Key ID', value: String(log.apiKeyId) },
  ];

  return (
    <Tabs defaultValue="info" className="w-full">
      <TabsList>
        <TabsTrigger value="info">基本信息</TabsTrigger>
        <TabsTrigger value="request">请求</TabsTrigger>
        <TabsTrigger value="response">响应</TabsTrigger>
        {log.isStream && <TabsTrigger value="chunks">流式响应</TabsTrigger>}
      </TabsList>

      <TabsContent value="info" className="mt-4 space-y-4">
        <Descriptions title="请求信息" items={basicItems} labelClassName="w-24" />
        <Descriptions title="Token 用量" items={tokenItems} labelClassName="w-24" />
        <Descriptions title="其他信息" items={otherItems} labelClassName="w-24" />
        {log.errorMessage && (
          <div>
            <h4 className="font-medium mb-2">错误信息</h4>
            <pre className="bg-muted overflow-auto rounded p-3 text-sm">{log.errorMessage}</pre>
          </div>
        )}
      </TabsContent>

      <TabsContent value="request" className="mt-4">
        {detail?.request ? (
          <div className="space-y-4">
            <div>
              <h4 className="text-sm font-medium mb-2">请求内容</h4>
              <pre className="bg-muted overflow-auto rounded p-4 text-sm max-h-[300px]">
                {detail.request}
              </pre>
            </div>
            {detail.requestRaw && (
              <div>
                <h4 className="text-sm font-medium mb-2">原始请求</h4>
                <div className="bg-muted overflow-auto rounded p-4 max-h-[300px]">
                  <JsonViewer data={detail.requestRaw} />
                </div>
              </div>
            )}
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">无请求详情（需要开启详细日志记录）</p>
        )}
      </TabsContent>

      <TabsContent value="response" className="mt-4">
        <div className="space-y-4">
          {detail?.reasoning && (
            <div>
              <h4 className="text-sm font-medium mb-2">推理过程</h4>
              <pre className="bg-muted overflow-auto rounded p-4 text-sm max-h-[200px]">
                {detail.reasoning}
              </pre>
            </div>
          )}
          {detail?.response && (
            <div>
              <h4 className="text-sm font-medium mb-2">消息正文</h4>
              <pre className="bg-muted overflow-auto rounded p-4 text-sm max-h-[300px]">
                {detail.response}
              </pre>
            </div>
          )}
          {detail?.responseRaw && (
            <div>
              <h4 className="text-sm font-medium mb-2">原始数据</h4>
              <div className="bg-muted overflow-auto rounded p-4 max-h-[300px]">
                <JsonViewer data={detail.responseRaw} />
              </div>
            </div>
          )}
          {!detail?.response && !detail?.reasoning && (
            <p className="text-muted-foreground text-sm">无响应详情（需要开启详细日志记录）</p>
          )}
        </div>
      </TabsContent>

      {log.isStream && (
        <TabsContent value="chunks" className="mt-4">
          <StreamChunksContent chunks={chunks} isLoading={chunksLoading} />
        </TabsContent>
      )}
    </Tabs>
  );
}

function StreamChunksContent({ chunks, isLoading }: { chunks?: RequestChunk[]; isLoading: boolean }) {
  const { Modal: ChunkModal, modalHandler: chunkModalHandler, meta: chunkMeta } = useModal<RequestChunk>();

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 p-4">
        <Loading size={16} /> <span className="text-sm">加载中...</span>
      </div>
    );
  }

  if (!chunks || chunks.length === 0) {
    return <p className="text-muted-foreground text-sm">无 chunk 数据</p>;
  }

  const chunkTypeLabels: Record<string, string> = {
    message: '消息',
    reasoning: '推理',
    usage: '用量',
    done: '结束',
    other: '其他',
  };

  const chunkTypeVariants: Record<string, 'default' | 'secondary' | 'outline'> = {
    message: 'default',
    reasoning: 'secondary',
    usage: 'outline',
    done: 'outline',
    other: 'outline',
  };

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-16">#</TableHead>
            <TableHead className="w-20">类型</TableHead>
            <TableHead>内容预览</TableHead>
            <TableHead className="w-20">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {chunks.slice(0, 100).map((chunk) => (
            <TableRow key={chunk.index}>
              <TableCell className="text-muted-foreground">{chunk.index}</TableCell>
              <TableCell>
                <Badge variant={chunkTypeVariants[chunk.type] ?? 'outline'}>
                  {chunkTypeLabels[chunk.type] ?? chunk.type}
                </Badge>
              </TableCell>
              <TableCell className="font-mono text-xs max-w-[500px] truncate" title={chunk.data}>
                {chunk.data.length > 100 ? chunk.data.substring(0, 100) + '...' : chunk.data}
              </TableCell>
              <TableCell>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => chunkModalHandler.open(`Chunk #${chunk.index}`, undefined, chunk)}
                >
                  查看
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {chunks.length > 100 && (
        <p className="text-muted-foreground text-xs mt-2">
          ... 还有 {chunks.length - 100} 个 chunk
        </p>
      )}

      <ChunkModal type="dialog" className="max-w-[80vw] max-h-[80vh]" title={chunkMeta ? `Chunk #${chunkMeta.index} - ${chunkTypeLabels[chunkMeta.type] ?? chunkMeta.type}` : ''}>
        {chunkMeta && (
          <div className="bg-muted overflow-auto rounded p-4 max-h-[70vh]">
            <JsonViewer data={chunkMeta.data} />
          </div>
        )}
      </ChunkModal>
    </>
  );
}

function JsonViewer({ data }: { data: string }) {
  try {
    const parsed = JSON.parse(data);
    return (
      <JsonView
        value={parsed}
        collapsed={false}
        displayDataTypes={false}
        style={{ backgroundColor: 'transparent', fontSize: '13px' }}
      />
    );
  } catch {
    return <pre className="text-sm break-all">{data}</pre>;
  }
}
