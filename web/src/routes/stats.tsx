import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import client from '@/api/client'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import dayjs from 'dayjs'
import { PageHeader } from '@/components/page-header'

interface StatsData {
  tokens: Array<{ date: string; count: number }>
  requests: Array<{ date: string; count: number }>
  costs: Array<{ date: string; cost: number }>
  behavior: Array<{ model: string; requests: number; tokens: number; cost: number }>
}

export const Route = createFileRoute('/stats')({
  component: StatsPage,
})

function StatsPage() {
  const today = dayjs().format('YYYY-MM-DD')
  const [startDate, setStartDate] = useState(dayjs().subtract(7, 'day').format('YYYY-MM-DD'))
  const [endDate, setEndDate] = useState(today)

  const [activeTab, setActiveTab] = useState('tokens')
  const tabs = [
    { value: 'tokens', label: 'Token 统计', api: '/stats/tokens' },
    { value: 'requests', label: '请求统计', api: '/stats/requests' },
    { value: 'costs', label: '费用统计', api: '/stats/costs' },
    { value: 'behavior', label: '使用行为', api: '/stats/behavior' },
  ]

  const currentApi = tabs.find((t) => t.value === activeTab)?.api

  const { data, isLoading } = useQuery({
    queryKey: ['stats', activeTab, startDate, endDate],
    queryFn: () =>
      client.get(currentApi!, {
        params: { start_date: startDate, end_date: endDate },
      }),
    enabled: !!currentApi,
  })

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:py-6">
      <div className="px-4 lg:px-6">
        <PageHeader
          title="统计分析"
          description="查看系统的使用统计和趋势"
          tabs={tabs.map((tab) => ({
            title: tab.label,
            active: activeTab === tab.value,
            onClick: () => setActiveTab(tab.value),
          }))}
        />
      </div>
      {/* Date Range */}
      <Card className="mx-4 lg:mx-6">
        <CardContent>
          <div className="flex items-center gap-4">
            <div className="space-y-1">
              <Label htmlFor="start-date">开始日期</Label>
              <Input
                id="start-date"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
              />
            </div>
            <span className="mt-6 text-muted-foreground">至</span>
            <div className="space-y-1">
              <Label htmlFor="end-date">结束日期</Label>
              <Input
                id="end-date"
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tab Content */}
      {tabs.map((tab) =>
        activeTab === tab.value ? (
          <Card key={tab.value} className="mx-4 lg:mx-6">
            <CardHeader>
              <CardTitle>{tab.label}</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="space-y-2">
                  {[...Array(5)].map((_, i) => (
                    <Skeleton key={i} className="h-10 w-full" />
                  ))}
                </div>
              ) : tab.value === 'behavior' ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>模型</TableHead>
                      <TableHead>请求数</TableHead>
                      <TableHead>Token 数</TableHead>
                      <TableHead>费用</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(data as unknown as StatsData['behavior'])?.map((item, i) => (
                      <TableRow key={i}>
                        <TableCell className="font-medium">{item.model}</TableCell>
                        <TableCell>{item.requests}</TableCell>
                        <TableCell>{item.tokens}</TableCell>
                        <TableCell>${item.cost.toFixed(4)}</TableCell>
                      </TableRow>
                    ))}
                    {(!(data as unknown as StatsData['behavior']) || (data as unknown as StatsData['behavior']).length === 0) && (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center text-muted-foreground">
                          暂无数据
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>日期</TableHead>
                      <TableHead>
                        {tab.value === 'tokens' ? 'Token 数' : tab.value === 'requests' ? '请求数' : '费用 ($)'}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(data as unknown as Array<{ date: string; count: number; cost?: number }>)?.map((item, i) => (
                      <TableRow key={i}>
                        <TableCell>{item.date}</TableCell>
                        <TableCell>
                          {tab.value === 'costs' ? `$${(item.cost ?? 0).toFixed(4)}` : item.count}
                        </TableCell>
                      </TableRow>
                    ))}
                    {(!data || (data as unknown as unknown[]).length === 0) && (
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
        ) : null
      )}
    </div>
  )
}
