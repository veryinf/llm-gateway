import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo, useState } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { AlertCircle, ArrowDown, ArrowUp, ArrowUpDown, Search, Sparkles } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { statsQueryService, type StatsQueryRequest, type StatsQueryFilter } from '@/services/stats';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';

export const Route = createFileRoute('/usage-stats')({
  component: UsageStatsPage,
});

type SortDir = 'asc' | 'desc';

function getDefaultDateRange() {
  const end = new Date();
  const start = new Date();
  start.setDate(start.getDate() - 6);
  return {
    start: start.toISOString().split('T')[0],
    end: end.toISOString().split('T')[0],
  };
}

function UsageStatsPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs([{ title: '用量统计' }]);
  }, []);

  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <h2 className="text-2xl font-semibold tracking-tight">用量统计</h2>

      <Tabs defaultValue="users" className="gap-4">
        <TabsList>
          <TabsTrigger value="users">用户统计</TabsTrigger>
          <TabsTrigger value="providerModels">服务商模型统计</TabsTrigger>
        </TabsList>

        <TabsContent value="users">
          <UsersTab />
        </TabsContent>

        <TabsContent value="providerModels">
          <ProviderModelsTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function DateRangePicker({
  start,
  end,
  onChange,
}: {
  start: string;
  end: string;
  onChange: (next: { start: string; end: string }) => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <Input
        type="date"
        value={start}
        onChange={(e) => onChange({ start: e.target.value, end })}
        className="w-[160px]"
      />
      <span className="text-muted-foreground">-</span>
      <Input
        type="date"
        value={end}
        onChange={(e) => onChange({ start, end: e.target.value })}
        className="w-[160px]"
      />
    </div>
  );
}

function SortHeader({
  label,
  field,
  currentField,
  currentDir,
  onSort,
  align = 'left',
}: {
  label: string;
  field: string;
  currentField?: string;
  currentDir?: SortDir;
  onSort: (field: string) => void;
  align?: 'left' | 'right';
}) {
  const active = currentField === field;
  const Icon = !active ? ArrowUpDown : currentDir === 'asc' ? ArrowUp : ArrowDown;
  return (
    <TableHead
      className={align === 'right' ? 'cursor-pointer select-none text-right' : 'cursor-pointer select-none'}
      onClick={() => onSort(field)}
    >
      <span className={align === 'right' ? 'inline-flex items-center gap-1' : 'inline-flex items-center gap-1'}>
        {label}
        <Icon className={active ? 'text-foreground size-3.5' : 'text-muted-foreground/50 size-3.5'} />
      </span>
    </TableHead>
  );
}

function Pagination({
  page,
  size,
  total,
  onChange,
}: {
  page: number;
  size: number;
  total: number;
  onChange: (next: { page: number; size: number }) => void;
}) {
  const totalPages = Math.max(1, Math.ceil(total / size));
  if (total === 0) return null;
  return (
    <div className="flex items-center justify-between px-2 py-3 text-sm">
      <div className="text-muted-foreground">共 {total.toLocaleString()} 条</div>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={page <= 1}
          onClick={() => onChange({ page: page - 1, size })}
        >
          上一页
        </Button>
        <span className="text-muted-foreground tabular-nums">
          第 {page} / {totalPages} 页
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={page >= totalPages}
          onClick={() => onChange({ page: page + 1, size })}
        >
          下一页
        </Button>
      </div>
    </div>
  );
}

function ErrorAlert({ message }: { message?: string }) {
  return (
    <div className="border-destructive/40 bg-destructive/5 text-destructive flex items-start gap-3 rounded-md border p-3 text-sm">
      <AlertCircle className="mt-0.5 size-4 shrink-0" />
      <div>
        <div className="font-medium">无法加载统计数据</div>
        <div className="text-destructive/80 text-xs">{message ?? '请检查网络或后端服务是否正常运行。'}</div>
      </div>
    </div>
  );
}

function EmptyState({ onReset }: { onReset?: () => void }) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center justify-center gap-3 py-12 text-center">
        <div className="bg-muted text-muted-foreground flex size-12 items-center justify-center rounded-full">
          <Sparkles className="size-6" />
        </div>
        <div>
          <div className="text-base font-medium">该时间段还没有调用记录</div>
          <div className="text-muted-foreground mt-1 text-sm">调整日期范围或关键词后重试。</div>
        </div>
        {onReset && (
          <Button variant="outline" size="sm" onClick={onReset}>
            重置筛选
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

function TableSkeleton({ cols }: { cols: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: 8 }).map((_, i) => (
        <Skeleton key={i} className="h-9 w-full" />
      ))}
      <div className="text-muted-foreground text-xs">{cols} 列</div>
    </div>
  );
}

function buildDateFilters(start: string, end: string): StatsQueryFilter[] {
  const endNext = new Date(end);
  endNext.setDate(endNext.getDate() + 1);
  return [
    { field: 'hour', op: 'gte', value: `${start}T00:00:00` },
    { field: 'hour', op: 'lt', value: `${endNext.toISOString().split('T')[0]}T00:00:00` },
  ];
}

function UsersTab() {
  const [dateRange, setDateRange] = useState(getDefaultDateRange);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [size] = useState(20);
  const [sort, setSort] = useState('total_tokens');
  const [order, setOrder] = useState<SortDir>('desc');

  const filters = useMemo(() => {
    const base = buildDateFilters(dateRange.start, dateRange.end);
    if (keyword) {
      base.push({ field: 'user_model', op: 'like' as const, value: keyword });
    }
    return base;
  }, [dateRange, keyword]);

  const query = useMemo<StatsQueryRequest>(() => ({
    dimensions: ['user_id'],
    measures: ['request_count', 'prompt_tokens', 'completion_tokens', 'reasoning_tokens', 'total_tokens'],
    filters,
    sort: [{ field: sort, dir: order }],
    page,
    size,
  }), [filters, sort, order, page, size]);

  const { data, isLoading, isError, error, isFetching } = useQuery({
    queryKey: ['usage-users', query],
    queryFn: () => statsQueryService.query(query),
    placeholderData: keepPreviousData,
  });

  const onSort = (field: string) => {
    const colMap: Record<string, string> = {
      username: 'user_id',
      department: 'user_id',
      requestCount: 'request_count',
      promptTokens: 'prompt_tokens',
      completionTokens: 'completion_tokens',
      reasoningTokens: 'reasoning_tokens',
      totalTokens: 'total_tokens',
      modelCount: 'request_count',
      lastCallAt: 'request_count',
    };
    const mapped = colMap[field] ?? field;
    if (sort === mapped) {
      setOrder(order === 'asc' ? 'desc' : 'asc');
    } else {
      setSort(mapped);
      setOrder('desc');
    }
    setPage(1);
  };

  const rows = useMemo(() => {
    if (!data?.rows.length) return [];
    return data.rows.map((r) => ({
      userId: Number(r.user_id ?? 0),
      username: '',
      department: '',
      requestCount: Number(r.request_count ?? 0),
      promptTokens: Number(r.prompt_tokens ?? 0),
      completionTokens: Number(r.completion_tokens ?? 0),
      reasoningTokens: Number(r.reasoning_tokens ?? 0),
      totalTokens: Number(r.total_tokens ?? 0),
      modelCount: 0,
      lastCallAt: '',
    }));
  }, [data]);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <DateRangePicker start={dateRange.start} end={dateRange.end} onChange={setDateRange} />
        <div className="relative w-[260px]">
          <Search className="text-muted-foreground absolute left-2 top-1/2 size-4 -translate-y-1/2" />
          <Input
            placeholder="按用户 ID / 模型搜索"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value);
              setPage(1);
            }}
            className="pl-8"
          />
        </div>
        {isFetching && !isLoading && (
          <span className="text-muted-foreground text-xs">刷新中…</span>
        )}
      </div>

      {isError && <ErrorAlert message={(error as Error)?.message} />}

      {isLoading ? (
        <TableSkeleton cols={9} />
      ) : data && rows.length === 0 ? (
        <EmptyState onReset={() => { setKeyword(''); setDateRange(getDefaultDateRange()); }} />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <SortHeader label="用户 ID" field="username" currentField={sort} currentDir={order} onSort={onSort} />
                <TableHead>部门</TableHead>
                <SortHeader label="请求数" field="requestCount" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="提示" field="promptTokens" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="完成" field="completionTokens" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="推理" field="reasoningTokens" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="总 Token" field="totalTokens" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <TableHead className="text-right">模型数</TableHead>
                <TableHead>最近调用</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((u) => (
                <TableRow key={u.userId}>
                  <TableCell className="font-medium">#{u.userId}</TableCell>
                  <TableCell className="text-muted-foreground">-</TableCell>
                  <TableCell className="text-right tabular-nums">{u.requestCount.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums">{u.promptTokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums">{u.completionTokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums">{u.reasoningTokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums font-medium">{u.totalTokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums">-</TableCell>
                  <TableCell className="text-muted-foreground">-</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <Pagination
            page={page}
            size={size}
            total={data?.total ?? 0}
            onChange={({ page: p }) => setPage(p)}
          />
        </div>
      )}
    </div>
  );
}

function ProviderModelsTab() {
  const [dateRange, setDateRange] = useState(getDefaultDateRange);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [size] = useState(20);
  const [sort, setSort] = useState('total_tokens');
  const [order, setOrder] = useState<SortDir>('desc');

  const filters = useMemo(() => {
    const base = buildDateFilters(dateRange.start, dateRange.end);
    if (keyword) {
      base.push({ field: 'provider_model', op: 'like' as const, value: keyword });
    }
    return base;
  }, [dateRange, keyword]);

  const query = useMemo<StatsQueryRequest>(() => ({
    dimensions: ['provider_model'],
    measures: ['request_count', 'prompt_tokens', 'completion_tokens', 'total_tokens', 'avg_latency_ms', 'unique_users'],
    filters,
    sort: [{ field: sort, dir: order }],
    page,
    size,
  }), [filters, sort, order, page, size]);

  const { data, isLoading, isError, error, isFetching } = useQuery({
    queryKey: ['usage-provider-models', query],
    queryFn: () => statsQueryService.query(query),
    placeholderData: keepPreviousData,
  });

  const onSort = (field: string) => {
    const colMap: Record<string, string> = {
      providerTitle: 'provider_model',
      providerModel: 'provider_model',
      requestCount: 'request_count',
      promptTokens: 'prompt_tokens',
      completionTokens: 'completion_tokens',
      totalTokens: 'total_tokens',
      userCount: 'unique_users',
      avgLatencyMs: 'avg_latency_ms',
    };
    const mapped = colMap[field] ?? field;
    if (sort === mapped) {
      setOrder(order === 'asc' ? 'desc' : 'asc');
    } else {
      setSort(mapped);
      setOrder('desc');
    }
    setPage(1);
  };

  const rows = useMemo(() => {
    if (!data?.rows.length) return [];
    return data.rows.map((r) => ({
      providerId: 0,
      providerTitle: '',
      providerModel: String(r.provider_model ?? ''),
      requestCount: Number(r.request_count ?? 0),
      promptTokens: Number(r.prompt_tokens ?? 0),
      completionTokens: Number(r.completion_tokens ?? 0),
      totalTokens: Number(r.total_tokens ?? 0),
      userCount: Number(r.unique_users ?? 0),
      avgLatencyMs: Number(r.avg_latency_ms ?? 0),
    }));
  }, [data]);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <DateRangePicker start={dateRange.start} end={dateRange.end} onChange={setDateRange} />
        <div className="relative w-[260px]">
          <Search className="text-muted-foreground absolute left-2 top-1/2 size-4 -translate-y-1/2" />
          <Input
            placeholder="按模型名搜索"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value);
              setPage(1);
            }}
            className="pl-8"
          />
        </div>
        {isFetching && !isLoading && (
          <span className="text-muted-foreground text-xs">刷新中…</span>
        )}
      </div>

      {isError && <ErrorAlert message={(error as Error)?.message} />}

      {isLoading ? (
        <TableSkeleton cols={8} />
      ) : data && rows.length === 0 ? (
        <EmptyState onReset={() => { setKeyword(''); setDateRange(getDefaultDateRange()); }} />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <SortHeader label="Model" field="providerModel" currentField={sort} currentDir={order} onSort={onSort} />
                <SortHeader label="请求数" field="requestCount" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="提示" field="promptTokens" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="完成" field="completionTokens" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="总 Token" field="totalTokens" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="用户数" field="userCount" currentField={sort} currentDir={order} onSort={onSort} align="right" />
                <SortHeader label="平均延迟" field="avgLatencyMs" currentField={sort} currentDir={order} onSort={onSort} align="right" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((p) => (
                <TableRow key={p.providerModel}>
                  <TableCell className="font-medium">{p.providerModel}</TableCell>
                  <TableCell className="text-right tabular-nums">{p.requestCount.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums">{p.promptTokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums">{p.completionTokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums font-medium">{p.totalTokens.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums">{p.userCount}</TableCell>
                  <TableCell className="text-right tabular-nums">{Math.round(p.avgLatencyMs)} ms</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <Pagination
            page={page}
            size={size}
            total={data?.total ?? 0}
            onChange={({ page: p }) => setPage(p)}
          />
        </div>
      )}
    </div>
  );
}
