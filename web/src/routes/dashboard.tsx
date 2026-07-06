import { createFileRoute, Link } from '@tanstack/react-router';
import { useEffect, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AlertCircle, ArrowRight, Sparkles } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { statsQueryService, type StatsQueryRequest } from '@/services/stats';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

export const Route = createFileRoute('/dashboard')({
  component: DashboardPage,
});

function getDefaultDateRange() {
  const end = new Date();
  const start = new Date();
  start.setDate(start.getDate() - 6);
  return {
    start: start.toISOString().split('T')[0],
    end: end.toISOString().split('T')[0],
  };
}

function buildDateFilters(start: string, end: string) {
  const endNext = new Date(end);
  endNext.setDate(endNext.getDate() + 1);
  return [
    { field: 'hour', op: 'gte' as const, value: `${start}T00:00:00` },
    { field: 'hour', op: 'lt' as const, value: `${endNext.toISOString().split('T')[0]}T00:00:00` },
  ];
}

function DashboardPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs([{ title: 'Dashboard' }]);
  }, []);

  const dateRange = getDefaultDateRange();
  const filters = buildDateFilters(dateRange.start, dateRange.end);

  // 总览统计
  const overviewQuery: StatsQueryRequest = {
    dimensions: [],
    measures: ['request_count', 'total_tokens', 'avg_latency_ms', 'unique_users', 'success_count'],
    filters,
  };

  // 请求趋势（按小时）
  const requestsQuery: StatsQueryRequest = {
    dimensions: ['hour'],
    measures: ['request_count', 'success_count', 'error_count'],
    filters,
    sort: [{ field: 'hour', dir: 'asc' }],
    size: 200,
  };

  // Top 服务商
  const topProvidersQuery: StatsQueryRequest = {
    dimensions: ['provider_model'],
    measures: ['request_count', 'total_tokens', 'unique_users'],
    filters,
    sort: [{ field: 'request_count', dir: 'desc' }],
    size: 10,
  };

  // Top 用户
  const topUsersQuery: StatsQueryRequest = {
    dimensions: ['user_id'],
    measures: ['request_count', 'total_tokens'],
    filters,
    sort: [{ field: 'request_count', dir: 'desc' }],
    size: 10,
  };

  const { data: overviewData, isLoading: overviewLoading, isError: overviewError } = useQuery({
    queryKey: ['dashboard-overview', dateRange],
    queryFn: () => statsQueryService.query(overviewQuery),
  });

  const { data: requestsData, isLoading: requestsLoading, isError: requestsError } = useQuery({
    queryKey: ['dashboard-requests', dateRange],
    queryFn: () => statsQueryService.query(requestsQuery),
  });

  const { data: providersData } = useQuery({
    queryKey: ['dashboard-providers', dateRange],
    queryFn: () => statsQueryService.query(topProvidersQuery),
  });

  const { data: usersData } = useQuery({
    queryKey: ['dashboard-users', dateRange],
    queryFn: () => statsQueryService.query(topUsersQuery),
  });

  const overview = useMemo(() => {
    if (!overviewData?.rows.length) return null;
    const r = overviewData.rows[0];
    const totalRequests = Number(r.request_count ?? 0);
    const successCount = Number(r.success_count ?? 0);
    return {
      totalRequests,
      totalTokens: Number(r.total_tokens ?? 0),
      avgLatencyMs: Number(r.avg_latency_ms ?? 0),
      activeUsers: Number(r.unique_users ?? 0),
      successRate: totalRequests > 0 ? (successCount / totalRequests) * 100 : 0,
    };
  }, [overviewData]);

  const requestStats = useMemo(() => {
    if (!requestsData?.rows.length) return [];
    return requestsData.rows.map((r) => ({
      date: String(r.hour ?? '').replace('T', ' ').slice(0, 13) + ':00',
      requestCount: Number(r.request_count ?? 0),
      successCount: Number(r.success_count ?? 0),
      errorCount: Number(r.error_count ?? 0),
    }));
  }, [requestsData]);

  const topProviders = useMemo(() => {
    if (!providersData?.rows.length) return [];
    return providersData.rows.map((r) => ({
      providerModel: String(r.provider_model ?? ''),
      requestCount: Number(r.request_count ?? 0),
      totalTokens: Number(r.total_tokens ?? 0),
      userCount: Number(r.unique_users ?? 0),
    }));
  }, [providersData]);

  const topUsers = useMemo(() => {
    if (!usersData?.rows.length) return [];
    return usersData.rows.map((r) => ({
      userId: Number(r.user_id ?? 0),
      requestCount: Number(r.request_count ?? 0),
      totalTokens: Number(r.total_tokens ?? 0),
    }));
  }, [usersData]);

  const avgTokensPerRequest = useMemo(() => {
    if (!overview?.totalRequests) return 0;
    return Math.round(overview.totalTokens / overview.totalRequests);
  }, [overview]);

  const isLoading = overviewLoading || requestsLoading;
  const hasError = overviewError || requestsError;
  const isEmpty = !isLoading && !hasError && (overview?.totalRequests ?? 0) === 0;

  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <div className="flex items-center gap-4">
        <h2 className="text-2xl font-semibold tracking-tight">Dashboard</h2>
      </div>

      {hasError && (
        <div className="border-destructive/40 bg-destructive/5 text-destructive flex items-start gap-3 rounded-md border p-3 text-sm">
          <AlertCircle className="mt-0.5 size-4 shrink-0" />
          <div>
            <div className="font-medium">无法加载统计数据</div>
            <div className="text-destructive/80 text-xs">请检查网络或后端服务是否正常运行。</div>
          </div>
        </div>
      )}

      {isLoading ? (
        <DashboardSkeleton />
      ) : isEmpty ? (
        <EmptyState />
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
            <StatCard title="总请求数" value={overview?.totalRequests} format="number" />
            <StatCard title="总 Token" value={overview?.totalTokens} format="number" />
            <StatCard title="平均 Token/请求" value={avgTokensPerRequest} format="number" />
            <StatCard title="平均延迟" value={overview?.avgLatencyMs ? Math.round(overview.avgLatencyMs) : 0} suffix="ms" format="number" />
            <StatCard title="成功率" value={overview?.successRate ?? 0} suffix="%" format="decimal" />
            <StatCard title="活跃用户" value={overview?.activeUsers} format="number" />
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>请求趋势</CardTitle>
              </CardHeader>
              <CardContent>
                <RequestsChart data={requestStats} />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Top 服务商</CardTitle>
              </CardHeader>
              <CardContent>
                <TopProvidersChart data={topProviders} />
              </CardContent>
            </Card>

            <Card className="md:col-span-2">
              <CardHeader>
                <CardTitle>Top 用户</CardTitle>
              </CardHeader>
              <CardContent>
                <TopUsersTable data={topUsers} />
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}

function StatCard({
  title,
  value,
  prefix,
  suffix,
  format,
}: {
  title: string;
  value?: number;
  prefix?: string;
  suffix?: string;
  format: 'number' | 'decimal';
}) {
  const display = value != null ? (format === 'decimal' ? value.toFixed(2) : value.toLocaleString()) : '-';
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-muted-foreground text-sm">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">
          {prefix}
          {display}
          {suffix}
        </div>
      </CardContent>
    </Card>
  );
}

function DashboardSkeleton() {
  return (
    <>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i}>
            <CardHeader>
              <Skeleton className="h-4 w-20" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-7 w-24" />
            </CardContent>
          </Card>
        ))}
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-24" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-[300px] w-full" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-24" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-[300px] w-full" />
          </CardContent>
        </Card>
        <Card className="md:col-span-2">
          <CardHeader>
            <Skeleton className="h-5 w-24" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-[300px] w-full" />
          </CardContent>
        </Card>
      </div>
    </>
  );
}

function EmptyState() {
  return (
    <Card>
      <CardContent className="flex flex-col items-center justify-center gap-4 py-16 text-center">
        <div className="bg-muted text-muted-foreground flex size-12 items-center justify-center rounded-full">
          <Sparkles className="size-6" />
        </div>
        <div>
          <div className="text-base font-medium">该时间段还没有 LLM 调用记录</div>
          <div className="text-muted-foreground mt-1 text-sm">调整日期范围，或发起一次 Chat Completions 请求后回来看看。</div>
        </div>
        <div className="flex gap-2">
          <Button asChild variant="outline">
            <a href="https://platform.openai.com/docs/api-reference/chat" target="_blank" rel="noreferrer">
              查看 API 文档
            </a>
          </Button>
          <Button asChild>
            <Link to="/request-logs">
              查看请求记录
              <ArrowRight className="size-4" />
            </Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function RequestsChart({ data }: { data: { date: string; requestCount: number; successCount: number; errorCount: number }[] }) {
  if (!data.length) {
    return <div className="text-muted-foreground flex h-[300px] items-center justify-center text-sm">暂无数据</div>;
  }

  const tickInterval = Math.max(0, Math.floor(data.length / 8) - 1);

  const renderTick = (props: { x?: number; y?: number; payload?: { value?: string } }) => {
    const { x = 0, y = 0, payload } = props;
    const value = payload?.value ?? '';
    const parts = value.split(' ');
    const datePart = parts[0] ?? '';
    const hourPart = parts[1] ?? '';
    const monthDay = datePart.slice(5);
    return (
      <g transform={`translate(${x},${y})`}>
        <text y={-2} textAnchor="middle" fill="hsl(var(--foreground))" fontSize={11} fontWeight={500}>
          {monthDay}
        </text>
        <text y={12} textAnchor="middle" fill="hsl(var(--muted-foreground))" fontSize={10}>
          {hourPart}
        </text>
      </g>
    );
  };

  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={data} margin={{ top: 16, right: 16, left: 0, bottom: 16 }}>
        <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
        <XAxis
          dataKey="date"
          tick={renderTick as never}
          height={50}
          interval={tickInterval}
          tickLine={false}
        />
        <YAxis tick={{ fontSize: 12 }} allowDecimals={false} />
        <Tooltip
          labelFormatter={(label: string) => label.replace(' ', ' ')}
        />
        <Line type="monotone" dataKey="requestCount" name="总请求" stroke="hsl(var(--chart-1))" strokeWidth={2} dot={false} />
        <Line type="monotone" dataKey="successCount" name="成功" stroke="hsl(var(--chart-2))" strokeWidth={2} dot={false} />
        <Line type="monotone" dataKey="errorCount" name="失败" stroke="hsl(var(--chart-5))" strokeWidth={2} dot={false} />
      </LineChart>
    </ResponsiveContainer>
  );
}

function TopProvidersChart({ data }: { data: { providerModel: string; requestCount: number; totalTokens: number; userCount: number }[] }) {
  if (!data.length) {
    return <div className="text-muted-foreground flex h-[300px] items-center justify-center text-sm">暂无数据</div>;
  }

  const renderTick = (props: { x?: number; y?: number; payload?: { value?: string } }) => {
    const { x = 0, y = 0, payload } = props;
    const modelName = payload?.value ?? '';
    return (
      <g transform={`translate(${x},${y})`}>
        <text y={-2} textAnchor="middle" fill="hsl(var(--foreground))" fontSize={11} fontWeight={500}>
          {modelName}
        </text>
      </g>
    );
  };

  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={data} margin={{ top: 16, right: 16, left: 0, bottom: 24 }}>
        <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
        <XAxis
          dataKey="providerModel"
          tick={renderTick as never}
          height={60}
          interval={0}
          tickLine={false}
        />
        <YAxis tick={{ fontSize: 12 }} allowDecimals={false} />
        <Tooltip
          content={({ active, payload }) => {
            if (!active || !payload?.length) return null;
            const p = payload[0].payload as { providerModel: string; requestCount: number; totalTokens: number; userCount: number };
            return (
              <div className="rounded-md border bg-background p-2 text-xs shadow-sm">
                <div className="font-medium">{p.providerModel}</div>
                <div className="mt-1 grid grid-cols-2 gap-x-3 gap-y-0.5">
                  <span className="text-muted-foreground">请求数</span>
                  <span className="text-right">{p.requestCount.toLocaleString()}</span>
                  <span className="text-muted-foreground">总 Token</span>
                  <span className="text-right">{p.totalTokens.toLocaleString()}</span>
                  <span className="text-muted-foreground">用户数</span>
                  <span className="text-right">{p.userCount}</span>
                </div>
              </div>
            );
          }}
        />
        <Line
          type="monotone"
          dataKey="requestCount"
          name="请求数"
          stroke="hsl(var(--chart-2))"
          strokeWidth={2}
          dot={{ r: 4, fill: 'hsl(var(--chart-2))' }}
          activeDot={{ r: 6 }}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}

function TopUsersTable({ data }: { data: { userId: number; requestCount: number; totalTokens: number }[] }) {
  if (!data.length) {
    return <div className="text-muted-foreground flex h-[300px] items-center justify-center text-sm">暂无数据</div>;
  }

  return (
    <div className="max-h-[300px] overflow-auto">
      <Table>
        <TableHeader className="sticky top-0 bg-background">
          <TableRow>
            <TableHead>用户 ID</TableHead>
            <TableHead className="text-right">请求数</TableHead>
            <TableHead className="text-right">总 Token</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((u) => (
            <TableRow key={u.userId}>
              <TableCell className="font-medium">#{u.userId}</TableCell>
              <TableCell className="text-right tabular-nums">{u.requestCount.toLocaleString()}</TableCell>
              <TableCell className="text-right tabular-nums">{u.totalTokens.toLocaleString()}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
