import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import client from '@/api/client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Skeleton } from '@/components/ui/skeleton'
import toast from 'react-hot-toast'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import { PageHeader } from '@/components/page-header'

interface Provider {
  id: number
  name: string
  type: string
  base_url: string
  is_active: boolean
  priority: number
  rate_limit_qpm: number
  rate_limit_burst: number
  created_at: string
}

export const Route = createFileRoute('/providers')({
  component: ProvidersPage,
})

function ProvidersPage() {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    type: 'openai',
    base_url: '',
    api_key: '',
    rate_limit_qpm: 0,
    rate_limit_burst: 0,
  })

  const { data, isLoading } = useQuery<Provider[]>({
    queryKey: ['providers'],
    queryFn: () => client.get('/admin/providers') as Promise<Provider[]>,
  })

  const createMutation = useMutation({
    mutationFn: (data: typeof formData) => client.post('/admin/providers', data),
    onSuccess: () => {
      toast.success('创建成功')
      queryClient.invalidateQueries({ queryKey: ['providers'] })
      setDialogOpen(false)
      resetForm()
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number } & Partial<typeof formData>) =>
      client.put(`/admin/providers/${id}`, data),
    onSuccess: () => {
      toast.success('更新成功')
      queryClient.invalidateQueries({ queryKey: ['providers'] })
      setDialogOpen(false)
      resetForm()
    },
  })

  const toggleMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: boolean }) =>
      client.put(`/admin/providers/${id}/toggle`, { is_active: status }),
    onSuccess: () => {
      toast.success('状态更新成功')
      queryClient.invalidateQueries({ queryKey: ['providers'] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => client.delete(`/admin/providers/${id}`),
    onSuccess: () => {
      toast.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['providers'] })
    },
  })

  const resetForm = () => {
    setFormData({ name: '', type: 'openai', base_url: '', api_key: '', rate_limit_qpm: 0, rate_limit_burst: 0 })
    setEditingProvider(null)
  }

  const handleEdit = (provider: Provider) => {
    setEditingProvider(provider)
    setFormData({
      name: provider.name,
      type: provider.type,
      base_url: provider.base_url,
      api_key: '',
      rate_limit_qpm: provider.rate_limit_qpm,
      rate_limit_burst: provider.rate_limit_burst,
    })
    setDialogOpen(true)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (editingProvider) {
      updateMutation.mutate({ id: editingProvider.id, ...formData })
    } else {
      createMutation.mutate(formData)
    }
  }

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:py-6">
      <div className="px-4 lg:px-6">
        <PageHeader
          title="Provider 管理"
          description="管理 AI 服务提供商配置"
          actions={
            <Button onClick={() => { resetForm(); setDialogOpen(true) }}>
              <Plus className="h-4 w-4 mr-2" />
              添加 Provider
            </Button>
          }
        />
      </div>
      <Card className="mx-4 lg:mx-6">
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>API Base</TableHead>
                  <TableHead>限流</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.map((provider) => (
                  <TableRow key={provider.id}>
                    <TableCell className="font-medium">{provider.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{provider.type}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[200px] truncate">
                      {provider.base_url}
                    </TableCell>
                    <TableCell>
                      {provider.rate_limit_qpm > 0 ? (
                        <span className="text-xs text-muted-foreground">
                          {provider.rate_limit_qpm} QPM
                          {provider.rate_limit_burst > 0 && ` / ${provider.rate_limit_burst} burst`}
                        </span>
                      ) : (
                        <span className="text-xs text-muted-foreground">不限制</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Switch
                          checked={provider.is_active}
                          onCheckedChange={(checked) =>
                            toggleMutation.mutate({ id: provider.id, status: checked })
                          }
                        />
                        <span className="text-xs text-muted-foreground">
                          {provider.is_active ? '启用' : '禁用'}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>{provider.created_at}</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button variant="ghost" size="icon" onClick={() => handleEdit(provider)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => deleteMutation.mutate(provider.id)}>
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {(!data || data.length === 0) && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
                      暂无 Provider
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingProvider ? '编辑 Provider' : '添加 Provider'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">名称</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="例如：OpenAI"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="type">类型</Label>
              <Select
                id="type"
                value={formData.type}
                onChange={(e) => setFormData({ ...formData, type: e.target.value })}
              >
                <option value="openai">OpenAI</option>
                <option value="azure">Azure</option>
                <option value="anthropic">Anthropic</option>
                <option value="google">Google</option>
                <option value="custom">自定义</option>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="base_url">API Base URL</Label>
              <Input
                id="base_url"
                value={formData.base_url}
                onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
                placeholder="https://api.openai.com"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="api_key">API Key</Label>
              <Input
                id="api_key"
                type="password"
                value={formData.api_key}
                onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
                placeholder="sk-..."
                required
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="rate_limit_qpm">QPM 限制</Label>
                <Input
                  id="rate_limit_qpm"
                  type="number"
                  min={0}
                  value={formData.rate_limit_qpm}
                  onChange={(e) => setFormData({ ...formData, rate_limit_qpm: Number(e.target.value) })}
                  placeholder="0=不限制"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="rate_limit_burst">Burst 并发</Label>
                <Input
                  id="rate_limit_burst"
                  type="number"
                  min={0}
                  value={formData.rate_limit_burst}
                  onChange={(e) => setFormData({ ...formData, rate_limit_burst: Number(e.target.value) })}
                  placeholder="0=默认(QPM/10)"
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                取消
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                {editingProvider ? '保存' : '创建'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
