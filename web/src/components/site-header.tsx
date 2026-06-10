import { useNavigate, useRouterState } from '@tanstack/react-router'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { LogOut, UserCircle } from 'lucide-react'
import { useAuth } from '@/hooks/use-auth'
import React from 'react'

const breadcrumbMap: Record<string, Array<{ title: string; path?: string }>> = {
  '/dashboard': [{ title: '仪表盘' }],
  '/users': [{ title: '管理', path: '/dashboard' }, { title: '用户管理' }],
  '/providers': [{ title: '管理', path: '/dashboard' }, { title: 'Provider 管理' }],
  '/api-keys': [{ title: '管理', path: '/dashboard' }, { title: 'API Key 管理' }],
  '/stats': [{ title: '概览', path: '/dashboard' }, { title: '统计分析' }],
  '/audit': [{ title: '管理', path: '/dashboard' }, { title: '审计日志' }],
  '/docs': [{ title: '其他', path: '/dashboard' }, { title: '接入文档' }],
  '/settings': [{ title: '其他', path: '/dashboard' }, { title: '系统设置' }],
}

export function SiteHeader() {
  const navigate = useNavigate()
  const { logout } = useAuth()
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname
  const breadcrumbs = breadcrumbMap[currentPath] || [{ title: 'LLM Gateway' }]

  const handleLogout = () => {
    logout()
    navigate({ to: '/login' })
  }

  return (
    <header className="flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator
          orientation="vertical"
          className="mx-2 data-[orientation=vertical]:h-4"
        />
        <Breadcrumb>
          <BreadcrumbList>
            {breadcrumbs.map((breadcrumb, index) => (
              <React.Fragment key={index}>
                <BreadcrumbItem className="hidden md:block">
                  {breadcrumb.path ? (
                    <BreadcrumbLink onClick={() => navigate({ to: breadcrumb.path! })}>
                      {breadcrumb.title}
                    </BreadcrumbLink>
                  ) : (
                    <BreadcrumbPage>{breadcrumb.title}</BreadcrumbPage>
                  )}
                </BreadcrumbItem>
                {breadcrumbs.length - 1 !== index && (
                  <BreadcrumbSeparator className="hidden md:block" />
                )}
              </React.Fragment>
            ))}
          </BreadcrumbList>
        </Breadcrumb>
        <div className="ml-auto flex items-center gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="rounded-full">
                <Avatar className="h-8 w-8">
                  <AvatarFallback>A</AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-56" align="end" sideOffset={4}>
              <div className="flex items-center gap-2 px-2 py-1.5">
                <Avatar className="h-8 w-8">
                  <AvatarFallback>A</AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">管理员</span>
                </div>
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem>
                <UserCircle />
                个人资料
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleLogout}>
                <LogOut />
                退出登录
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  )
}
