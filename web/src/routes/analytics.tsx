import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { statsService } from '@/services/stats';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';
import { Loading } from '@/components/loader';

export const Route = createFileRoute('/analytics')({
  component: AnalyticsPage,
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

function AnalyticsPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const [dateRange, setDateRange] = useState(getDefaultDateRange);

  useEffect(() => {
    setBreadcrumbs([{ title: '统计' }, { title: '统计分析' }]);
  }, []);

  const { data: tokens, isLoading: tokensLoading } = useQuery({
    queryKey: ['analytics-tokens', dateRange],
    queryFn: () => statsService.fetchTokens(dateRange.start, dateRange.end),
  });

  const { data: costs, isLoading: costsLoading } = useQuery({
    queryKey: ['analytics-costs', dateRange],
    queryFn: () => statsService.fetchCosts(dateRange.start, dateRange.end),
  });

  const { data: behavior, isLoading: behaviorLoading } = useQuery({
    queryKey: ['analytics-behavior', dateRange],
    queryFn: () => statsService.fetchBehavior(dateRange.start, dateRange.end),
  });

  const isLoading = tokensLoading || costsLoading || behaviorLoading;

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-8">
        <Loading size={32} />
      </div>
    );
  }

  // Aggregate token usage by model
  const tokenByModel = aggregateByField(tokens ?? [], 'model_name', 'total_tokens');

  // Aggregate costs by model
  const costByModel = aggregateByField(costs ?? [], 'model_name', 'total_cost');

  // Top users by usage
  const topUsers = aggregateByField(behavior ?? [], 'username', 'count').slice(0, 10);

  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <div className="flex items-center gap-4">
        <h2 className="text-2xl font-semibold tracking-tight">统计分析</h2>
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

      {/* Token Usage Summary */}
      <Card>
        <CardHeader>
          <CardTitle>Token 用量（按模型）</CardTitle>
        </CardHeader>
        <CardContent>
          {tokenByModel.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={tokenByModel}>
                <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
                <XAxis dataKey="key" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} />
                <Tooltip />
                <Bar dataKey="value" name="Token 用量" fill="hsl(var(--chart-1))" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="text-muted-foreground flex h-[300px] items-center justify-center text-sm">暂无数据</div>
          )}
        </CardContent>
      </Card>

      {/* Cost by Model */}
      <Card>
        <CardHeader>
          <CardTitle>费用分布（按模型）</CardTitle>
        </CardHeader>
        <CardContent>
          {costByModel.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={costByModel}>
                <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
                <XAxis dataKey="key" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} />
                <Tooltip formatter={(v: number) => `¥${v.toFixed(2)}`} />
                <Bar dataKey="value" name="费用" fill="hsl(var(--chart-2))" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="text-muted-foreground flex h-[300px] items-center justify-center text-sm">暂无数据</div>
          )}
        </CardContent>
      </Card>

      {/* Top Users */}
      <Card>
        <CardHeader>
          <CardTitle>Top 用户</CardTitle>
        </CardHeader>
        <CardContent>
          {topUsers.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>排名</TableHead>
                  <TableHead>用户</TableHead>
                  <TableHead>请求数</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {topUsers.map((item, idx) => (
                  <TableRow key={item.key}>
                    <TableCell>{idx + 1}</TableCell>
                    <TableCell>{item.key}</TableCell>
                    <TableCell>{item.value.toLocaleString()}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="text-muted-foreground text-center text-sm">暂无数据</div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function aggregateByField<T>(data: T[], field: keyof T, valueField: keyof T): { key: string; value: number }[] {
  const map = new Map<string, number>();
  for (const item of data) {
    const k = String(item[field] ?? '');
    const v = Number(item[valueField] ?? 0);
    map.set(k, (map.get(k) ?? 0) + v);
  }
  return Array.from(map.entries())
    .map(([key, value]) => ({ key, value }))
    .sort((a, b) => b.value - a.value);
}
