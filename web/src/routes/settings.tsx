import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import toast from 'react-hot-toast'
import { PageHeader } from '@/components/page-header'

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
})

function SettingsPage() {
  const handleSave = () => {
    toast('功能开发中，敬请期待', { icon: '🚧' })
  }

  const [activeTab, setActiveTab] = useState('general')

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:py-6">
      <div className="px-4 lg:px-6">
        <PageHeader
          title="系统设置"
          description="配置系统的基本参数"
          tabs={[
            { title: '通用设置', active: activeTab === 'general', onClick: () => setActiveTab('general') },
            { title: '邮件通知', active: activeTab === 'email', onClick: () => setActiveTab('email') },
          ]}
        />
      </div>

      {activeTab === 'general' && (
        <Card className="mx-4 lg:mx-6">
          <CardContent className="space-y-4 pt-6">
              <div className="space-y-2">
                <Label htmlFor="site-name">站点名称</Label>
                <Input id="site-name" defaultValue="LLM Gateway" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="default-model">默认模型</Label>
                <Input id="default-model" defaultValue="gpt-4o-mini" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="max-tokens">单次请求最大 Token</Label>
                <Input id="max-tokens" type="number" defaultValue="4096" />
              </div>
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>允许注册</Label>
                  <p className="text-sm text-muted-foreground">是否允许新用户注册</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>自动审核</Label>
                  <p className="text-sm text-muted-foreground">新用户注册后自动审核通过</p>
                </div>
                <Switch />
              </div>
              <Button onClick={handleSave}>保存设置</Button>
            </CardContent>
          </Card>
      )}

      {activeTab === 'email' && (
        <Card className="mx-4 lg:mx-6">
          <CardContent className="space-y-4 pt-6">
              <div className="space-y-2">
                <Label htmlFor="smtp-host">SMTP 服务器</Label>
                <Input id="smtp-host" placeholder="smtp.example.com" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtp-port">SMTP 端口</Label>
                <Input id="smtp-port" type="number" defaultValue="587" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtp-user">SMTP 用户名</Label>
                <Input id="smtp-user" placeholder="noreply@example.com" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtp-pass">SMTP 密码</Label>
                <Input id="smtp-pass" type="password" placeholder="••••••••" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="notify-email">通知邮箱</Label>
                <Input id="notify-email" type="email" placeholder="admin@example.com" />
              </div>
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>启用邮件通知</Label>
                  <p className="text-sm text-muted-foreground">是否启用邮件通知功能</p>
                </div>
                <Switch />
              </div>
              <Button onClick={handleSave}>保存设置</Button>
            </CardContent>
          </Card>
      )}
    </div>
  )
}
