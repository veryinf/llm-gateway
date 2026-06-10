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
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import toast from 'react-hot-toast'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import { PageHeader } from '@/components/page-header'

interface User {
  id: number
  username: string
  email: string
  department: string
  role: string
  is_active: boolean
  created_at: string
}

export const Route = createFileRoute('/users')({
  component: UsersPage,
})

function UsersPage() {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    department: '',
    role: 'user',
  })

  const { data, isLoading } = useQuery<User[]>({
    queryKey: ['users'],
    queryFn: () => client.get('/admin/users') as Promise<User[]>,
  })

  const createMutation = useMutation({
    mutationFn: (data: typeof formData) => client.post('/admin/users', data),
    onSuccess: () => {
      toast.success('创建成功')
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setDialogOpen(false)
      resetForm()
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number } & Partial<typeof formData>) =>
      client.put(`/admin/users/${id}`, data),
    onSuccess: () => {
      toast.success('更新成功')
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setDialogOpen(false)
      resetForm()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => client.delete(`/admin/users/${id}`),
    onSuccess: () => {
      toast.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })

  const resetForm = () => {
    setFormData({ username: '', email: '', password: '', department: '', role: 'user' })
    setEditingUser(null)
  }

  const handleEdit = (user: User) => {
    setEditingUser(user)
    setFormData({
      username: user.username,
      email: user.email,
      password: '',
      department: user.department,
      role: user.role,
    })
    setDialogOpen(true)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (editingUser) {
      updateMutation.mutate({ id: editingUser.id, ...formData })
    } else {
      createMutation.mutate(formData)
    }
  }

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:py-6">
      <div className="px-4 lg:px-6">
        <PageHeader
          title="用户管理"
          description="管理系统用户和配额"
          actions={
            <Button onClick={() => { resetForm(); setDialogOpen(true) }}>
              <Plus className="h-4 w-4 mr-2" />
              添加用户
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
                  <TableHead>用户名</TableHead>
                  <TableHead>邮箱</TableHead>
                  <TableHead>角色</TableHead>
                  <TableHead>部门</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell className="font-medium">{user.username}</TableCell>
                    <TableCell>{user.email}</TableCell>
                    <TableCell><Badge variant="outline">{user.role}</Badge></TableCell>
                    <TableCell>{user.department || '-'}</TableCell>
                    <TableCell>
                      <Badge variant={user.is_active ? 'default' : 'secondary'}>
                        {user.is_active ? '正常' : '禁用'}
                      </Badge>
                    </TableCell>
                    <TableCell>{user.created_at}</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button variant="ghost" size="icon" onClick={() => handleEdit(user)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => deleteMutation.mutate(user.id)}>
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {(!data || data.length === 0) && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
                      暂无数据
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingUser ? '编辑用户' : '添加用户'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                value={formData.username}
                onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">邮箱</Label>
              <Input
                id="email"
                type="email"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">
                密码{editingUser ? '（留空则不修改）' : ''}
              </Label>
              <Input
                id="password"
                type="password"
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                required={!editingUser}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="department">部门</Label>
              <Input
                id="department"
                value={formData.department}
                onChange={(e) => setFormData({ ...formData, department: e.target.value })}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                取消
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                {editingUser ? '保存' : '创建'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
