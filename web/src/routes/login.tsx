import { useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useMutation } from '@tanstack/react-query'
import axios from 'axios'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LayoutGrid } from 'lucide-react'

export const Route = createFileRoute('/login')({
  component: LoginPage,
})

function LoginPage() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const loginMutation = useMutation({
    mutationFn: async (data: { username: string; password: string }) => {
      const res = await axios.post('/api/admin/login', data)
      return res.data
    },
    onSuccess: (data: { code: number; msg: string; data: { token: string } }) => {
      if (data.code === 0) {
        localStorage.setItem('token', data.data.token)
        toast.success('登录成功')
        navigate({ to: '/dashboard' })
      } else {
        toast.error(data.msg || '登录失败')
      }
    },
    onError: (error: unknown) => {
      if (axios.isAxiosError(error)) {
        toast.error(error.response?.data?.msg || '登录失败')
      } else {
        toast.error('网络错误')
      }
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    loginMutation.mutate({ username, password })
  }

  return (
    <div className="grid min-h-svh lg:grid-cols-2">
      <div className="bg-gradient-to-br from-primary/80 via-primary/60 to-primary/40 relative hidden lg:flex lg:flex-col lg:items-center lg:justify-center gap-6 p-12">
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute -right-20 -top-20 size-64 rounded-full bg-white/10" />
          <div className="absolute -bottom-16 -left-16 size-48 rounded-full bg-white/5" />
          <div className="absolute right-1/4 bottom-1/4 size-32 rounded-full bg-white/5" />
        </div>
        <div className="relative z-10 flex flex-col items-center gap-4 text-center">
          <div className="bg-white/20 text-white flex size-14 items-center justify-center rounded-2xl backdrop-blur-sm">
            <LayoutGrid className="size-7" />
          </div>
          <h1 className="text-3xl font-bold text-white">LLM Gateway</h1>
          <p className="text-white/80 max-w-sm text-sm leading-relaxed">
            统一 LLM API 网关，为团队提供标准化的大模型接入、用量统计与审计能力
          </p>
        </div>
      </div>
      <div className="flex flex-col gap-6 p-6 md:p-10">
        <div className="flex justify-center gap-2 lg:justify-start">
          <a href="#" className="flex items-center gap-2 font-medium">
            <div className="bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-md">
              <LayoutGrid className="size-4" />
            </div>
            LLM Gateway
          </a>
        </div>
        <div className="flex flex-1 items-center justify-center">
          <div className="w-full max-w-xs">
            <form onSubmit={handleSubmit} className="flex flex-col gap-6">
              <div className="flex flex-col items-center gap-1 text-center">
                <h1 className="text-2xl font-bold">欢迎回来</h1>
                <p className="text-muted-foreground text-sm text-balance">请输入您的账户信息登录管理后台</p>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="username">用户名</Label>
                <Input
                  id="username"
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="请输入用户名"
                  required
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="password">密码</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="请输入密码"
                  required
                />
              </div>
              <Button type="submit" className="w-full" disabled={loginMutation.isPending}>
                {loginMutation.isPending ? '登录中...' : '登录'}
              </Button>
            </form>
          </div>
        </div>
      </div>
    </div>
  )
}
