import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { statsService, type RequestStat } from '@/services/stats';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';
import { Loading } from '@/components/loader';

export const Route = createFileRoute('/dashboard')({
  component: DashboardPage,
});

function getDefaultDateRange() {
  const end = new Date();
  const start = new Date();
  start.setDate(start.getDate() - 7);
  return {
    start: start.toISOString().split('T')[0],
    end: end.toISOString().split('T')[0],
  };
}

function DashboardPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const [dateRange, setDateRange] = useState(getDefaultDateRange);

  useEffect(() => {
    setBreadcrumbs([{ title: 'Dashboard' }]);
  }, []);

  const { data: overview, isLoading: overviewLoading } = useQuery({
    queryKey: ['dashboard-overview', dateRange],
    queryFn: () => statsService.fetchOverview(dateRange.start, dateRange.end),
  });

  const { data: requestStats, isLoading: requestLoading } = useQuery({
    queryKey: ['dashboard-requests', dateRange],
    queryFn: () => statsService.fetchRequests(dateRange.start, dateRange.end),
  });

  if (overviewLoading || requestLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-8">
        <Loading size={32} />
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <div className="flex items-center gap-4">
        <h2 className="text-2xl font-semibold tracking-tight">Dashboard</h2>
        <div className="ml-auto flex items-center gap-2">
          <input
            type="date"
            value={dateRange.start}
            onChange={(e) => setDateRange((r) => ({ ...r, start: e.target.value }))}
            className="border-input bg-background ring-ring h-9 rounded-md border px-3 text-sm"
          />
          <span className="text-muted-foreground">-</span>
          <input
            type="date"
            value={dateRange.end}
            onChange={(e) => setDateRange((r) => ({ ...r, end: e.target.value }))}
            className="border-input bg-background ring-ring h-9 rounded-md border px-3 text-sm"
          />
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        <StatCard title="总请求数" value={overview?.total_requests} format="number" />
        <StatCard title="总 Token" value={overview?.total_tokens} format="number" />
        <StatCard title="总费用" value={overview?.total_cost} prefix="¥" format="decimal" />
        <StatCard title="平均延迟" value={overview?.avg_latency_ms} suffix="ms" format="number" />
        <StatCard title="成功率" value={overview?.success_rate} suffix="%" format="decimal" />
        <StatCard title="活跃用户" value={overview?.active_users} format="number" />
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>请求趋势</CardTitle>
          </CardHeader>
          <CardContent>
            <RequestsChart data={requestStats ?? []} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Top 模型</CardTitle>
          </CardHeader>
          <CardContent>
            <TopModelsChart data={overview?.top_models ?? []} />
          </CardContent>
        </Card>
      </div>
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

function RequestsChart({ data }: { data: RequestStat[] }) {
  if (!data.length) {
    return <div className="text-muted-foreground flex h-[300px] items-center justify-center text-sm">暂无数据</div>;
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={data}>
        <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
        <XAxis dataKey="date" tick={{ fontSize: 12 }} />
        <YAxis tick={{ fontSize: 12 }} />
        <Tooltip />
        <Line type="monotone" dataKey="request_count" name="总请求" stroke="hsl(var(--chart-1))" strokeWidth={2} dot={false} />
        <Line type="monotone" dataKey="success_count" name="成功" stroke="hsl(var(--chart-2))" strokeWidth={2} dot={false} />
        <Line type="monotone" dataKey="error_count" name="失败" stroke="hsl(var(--chart-5))" strokeWidth={2} dot={false} />
      </LineChart>
    </ResponsiveContainer>
  );
}

function TopModelsChart({ data }: { data: { model_name: string; count: number }[] }) {
  if (!data.length) {
    return <div className="text-muted-foreground flex h-[300px] items-center justify-center text-sm">暂无数据</div>;
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={data} layout="vertical">
        <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
        <XAxis type="number" tick={{ fontSize: 12 }} />
        <YAxis type="category" dataKey="model_name" tick={{ fontSize: 12 }} width={120} />
        <Tooltip />
        <Bar dataKey="count" name="请求数" fill="hsl(var(--chart-1))" radius={[0, 4, 4, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}
