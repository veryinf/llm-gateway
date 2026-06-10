import { Outlet, createRootRouteWithContext, useLocation, useNavigate } from '@tanstack/react-router';
import type { QueryClient } from '@tanstack/react-query';
import { AppSidebar } from '@/components/app-sidebar';
import { SiteHeader } from '@/components/site-header';
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { useAuth } from '@/hooks/use-auth';
import { Loading } from '@/components/loader';
import { getAuthMode } from '@/lib/request';

interface MyRouterContext {
  queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<MyRouterContext>()({
  component: RootComponent,
});

function RootComponent() {
  const location = useLocation();
  const navigate = useNavigate();
  const { isAuthenticated, loading } = useAuth();

  const publicPaths = ['/auth/login'];
  const isPublic = publicPaths.includes(location.pathname);
  const isSignatureMode = getAuthMode() === 'signature';

  if (loading) {
    return (
      <div className="flex h-dvh w-full items-center justify-center">
        <Loading size={40} />
      </div>
    );
  }

  if (isPublic) {
    return <Outlet />;
  }

  if (!isAuthenticated && !isSignatureMode) {
    navigate({ to: '/auth/login' });
    return null;
  }

  return (
    <SidebarProvider
      style={
        {
          '--sidebar-width': 'calc(var(--spacing) * 52)',
          '--header-height': 'calc(var(--spacing) * 12)',
        } as React.CSSProperties
      }
    >
      <AppSidebar variant="floating" />
      <SidebarInset>
        <SiteHeader />
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  );
}
