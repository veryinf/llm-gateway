import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import client from '@/api/client'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { BarChart3, Activity, Coins, Users } from 'lucide-react'

interface OverviewData {
  total_requests: number
  total_tokens: number
  total_cost: number
  active_users: number
  top_models: Array<{
    model_name: string
    count: number
  }>
}

export const Route = createFileRoute('/dashboard')({
  component: DashboardPage,
})

function DashboardPage() {
  const { data, isLoading } = useQuery<OverviewData>({
    queryKey: ['dashboard-overview'],
    queryFn: () => client.get('/dashboard/overview') as Promise<OverviewData>,
  })

  const stats = [
    {
      title: '总请求数',
      value: data?.total_requests ?? 0,
      icon: Activity,
      description: '所有 API 请求总量',
      color: 'text-blue-600',
      bgColor: 'bg-blue-50',
    },
    {
      title: '总 Token',
      value: data?.total_tokens ?? 0,
      icon: BarChart3,
      description: 'Token 消耗总量',
      color: 'text-green-600',
      bgColor: 'bg-green-50',
    },
    {
      title: '总费用',
      value: data?.total_cost?.toFixed(4) ?? '0.0000',
      icon: Coins,
      description: '累计 API 费用',
      color: 'text-yellow-600',
      bgColor: 'bg-yellow-50',
    },
    {
      title: '活跃用户',
      value: data?.active_users ?? 0,
      icon: Users,
      description: '已注册用户数',
      color: 'text-purple-600',
      bgColor: 'bg-purple-50',
    },
  ]

  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
          <div className="hidden h-full flex-1 flex-col gap-8 pt-0 pb-0 p-8 md:flex">
            <div className="flex items-center justify-between gap-2">
              <div className="flex flex-col gap-1">
                <h2 className="text-2xl font-semibold tracking-tight">欢迎回来！</h2>
                <p className="text-muted-foreground">这里是您的 LLM Gateway 仪表板概览。</p>
              </div>
            </div>
          </div>

          {/* Stat Cards */}
          <div className="grid grid-cols-1 gap-4 px-4 sm:grid-cols-2 lg:grid-cols-4 lg:px-6">
            {stats.map((stat) => (
              <Card key={stat.title} className="relative overflow-hidden">
                <CardHeader className="pb-2">
                  <CardDescription className="flex items-center gap-1.5">
                    <stat.icon className={`size-4 ${stat.color}`} />
                    {stat.title}
                  </CardDescription>
                  <CardTitle className="text-2xl font-semibold tabular-nums">
                    {isLoading ? (
                      <span className="inline-block h-7 w-16 animate-pulse rounded bg-muted" />
                    ) : (
                      stat.value
                    )}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-muted-foreground text-xs">{stat.description}</p>
                </CardContent>
                <div
                  className={`absolute -right-4 -top-4 size-20 rounded-full opacity-10 ${stat.bgColor}`}
                />
              </Card>
            ))}
          </div>

          {/* Model Ranking */}
          <div className="px-4 lg:px-6">
            <Card>
              <CardHeader>
                <CardTitle>模型排行</CardTitle>
                <CardDescription>按请求量排序的模型使用情况</CardDescription>
              </CardHeader>
              <CardContent>
                {isLoading ? (
                  <div className="space-y-2">
                    {[...Array(5)].map((_, i) => (
                      <span key={i} className="block h-8 w-full animate-pulse rounded bg-muted" />
                    ))}
                  </div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>模型</TableHead>
                        <TableHead>请求数</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data?.top_models?.map((item) => (
                        <TableRow key={item.model_name}>
                          <TableCell className="font-medium">{item.model_name}</TableCell>
                          <TableCell>{item.count}</TableCell>
                        </TableRow>
                      ))}
                      {(!data?.top_models || data.top_models.length === 0) && (
                        <TableRow>
                          <TableCell colSpan={2} className="text-center text-muted-foreground">
                            暂无数据
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}
