import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import client from '@/api/client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
import { Skeleton } from '@/components/ui/skeleton'
import toast from 'react-hot-toast'
import { Plus, Trash2, Copy, Eye, EyeOff } from 'lucide-react'
import { PageHeader } from '@/components/page-header'

interface ApiKey {
  id: number
  user_id: number
  name: string
  key_prefix: string
  is_active: boolean
  created_at: string
}

export const Route = createFileRoute('/api-keys')({
  component: ApiKeysPage,
})

function ApiKeysPage() {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [keyName, setKeyName] = useState('')
  const [newKey, setNewKey] = useState<{ name: string; raw_key: string } | null>(null)
  const [visibleKeys, setVisibleKeys] = useState<Set<number>>(new Set())

  const { data, isLoading } = useQuery<ApiKey[]>({
    queryKey: ['api-keys'],
    queryFn: () => client.get('/admin/api-keys') as Promise<ApiKey[]>,
  })

  const createMutation = useMutation({
    mutationFn: (data: { name: string }) =>
      client.post('/admin/users/1/api-keys', data) as Promise<{ api_key: { name: string }; raw_key: string }>,
    onSuccess: (data) => {
      toast.success('API Key 创建成功')
      setNewKey({ name: data.api_key.name, raw_key: data.raw_key })
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => client.delete(`/admin/api-keys/${id}`),
    onSuccess: () => {
      toast.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
    },
  })

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate({ name: keyName })
    setKeyName('')
  }

  const toggleKeyVisibility = (id: number) => {
    setVisibleKeys((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success('已复制到剪贴板')
  }

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:py-6">
      {newKey && (
        <div className="mx-4 lg:mx-6 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
          <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200 mb-2">
            新创建的 API Key（请立即保存，关闭后将无法再次查看）：
          </p>
          <div className="flex items-center gap-2">
            <code className="bg-yellow-100 dark:bg-yellow-900/40 px-2 py-1 rounded text-sm">
              {newKey.raw_key}
            </code>
            <Button variant="ghost" size="icon" onClick={() => copyToClipboard(newKey.raw_key)}>
              <Copy className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      <div className="px-4 lg:px-6">
        <PageHeader
          title="API Key 管理"
          description="管理 API 访问密钥"
          actions={
            <Button onClick={() => { setNewKey(null); setDialogOpen(true) }}>
              <Plus className="h-4 w-4 mr-2" />
              创建 API Key
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
                  <TableHead>Key 前缀</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <code className="text-xs">
                          {visibleKeys.has(key.id) ? key.key_prefix : key.key_prefix + '••••••••••••••••••••'}
                        </code>
                        <Button variant="ghost" size="icon" onClick={() => toggleKeyVisibility(key.id)}>
                          {visibleKeys.has(key.id) ? (
                            <EyeOff className="h-3 w-3" />
                          ) : (
                            <Eye className="h-3 w-3" />
                          )}
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => copyToClipboard(key.key_prefix)}>
                          <Copy className="h-3 w-3" />
                        </Button>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={key.is_active ? 'default' : 'secondary'}>
                        {key.is_active ? '启用' : '禁用'}
                      </Badge>
                    </TableCell>
                    <TableCell>{key.created_at}</TableCell>
                    <TableCell>
                      <Button variant="ghost" size="icon" onClick={() => deleteMutation.mutate(key.id)}>
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
                {(!data || data.length === 0) && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground">
                      暂无 API Key
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
            <DialogTitle>创建 API Key</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="keyName">Key 名称</Label>
              <Input
                id="keyName"
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                placeholder="例如：生产环境 Key"
                required
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                取消
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? '创建中...' : '创建'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
