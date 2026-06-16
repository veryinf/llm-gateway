import * as React from 'react';
import { LayoutDashboard, Users, Server, Cpu, ScrollText, Shield, Settings, Key, BarChart3, Blocks, Globe } from 'lucide-react';
import { Link, useLocation } from '@tanstack/react-router';
import { NavUser } from '@/components/nav-user';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar';

type MenuItem = {
  title: string;
  url: string;
  icon?: React.ComponentType;
};

const menuSet = {
  dashboard: [{ title: 'Dashboard', url: '/dashboard', icon: LayoutDashboard }],
  upstream: [
    { title: 'LLM 服务商', url: '/providers', icon: Server },
    { title: '服务商模型', url: '/upstream-models', icon: Blocks },
  ],
  downstream: [
    { title: '用户管理', url: '/users', icon: Users },
    { title: 'API Key 管理', url: '/keys', icon: Key },
    { title: '调用端模型', url: '/downstream-models', icon: Globe },
  ],
  routing: [
    { title: '模型路由', url: '/models', icon: Cpu },
    { title: '系统设置', url: '/settings', icon: Settings },
    { title: '请求记录', url: '/request-logs', icon: ScrollText },
    { title: '用量统计', url: '/usage-stats', icon: BarChart3 },
  ],
};

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild className="data-[slot=sidebar-menu-button]:!p-1.5">
              <Link to="/">
                <Shield className="!size-5" />
                <span className="text-base font-semibold">LLM Gateway</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              {menuSet.dashboard.map((item) => (
                <MenuItemRender key={item.url} item={item} />
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup className="group-data-[collapsible=icon]:hidden">
          <SidebarGroupLabel>路由</SidebarGroupLabel>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              {menuSet.routing.map((item) => (
                <MenuItemRender key={item.url} item={item} />
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup className="group-data-[collapsible=icon]:hidden">
          <SidebarGroupLabel>上游</SidebarGroupLabel>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              {menuSet.upstream.map((item) => (
                <MenuItemRender key={item.url} item={item} />
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup className="group-data-[collapsible=icon]:hidden">
          <SidebarGroupLabel>下游</SidebarGroupLabel>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              {menuSet.downstream.map((item) => (
                <MenuItemRender key={item.url} item={item} />
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
    </Sidebar>
  );
}

const activeClass =
  'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground active:bg-primary/90 active:text-primary-foreground min-w-8 duration-200 ease-linear';

function MenuItemRender({ item }: { item: MenuItem; }) {
  const location = useLocation();
  const isActive = location.pathname === item.url || location.pathname.startsWith(item.url + '/');
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        tooltip={item.title}
        asChild
        className={isActive ? activeClass : ''}
      >
        <Link to={item.url}>
          {item.icon && <item.icon />}
          <span>{item.title}</span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}
