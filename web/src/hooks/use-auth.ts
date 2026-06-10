import { useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchProfile, logout, type SessionUser } from '@/services/auth';

export function useAuth() {
  const token = localStorage.getItem('accessToken');
  const queryClient = useQueryClient();

  const { data: user, isLoading } = useQuery({
    queryKey: ['current-user'],
    queryFn: fetchProfile,
    enabled: !!token,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
    gcTime: 0,
    staleTime: Infinity,
  });

  return {
    isAuthenticated: !!user,
    user: user as SessionUser | null,
    loading: isLoading,
    refresh() {
      return queryClient.invalidateQueries({ queryKey: ['current-user'] });
    },
    logout() {
      logout();
      queryClient.setQueryData(['current-user'], null);
      queryClient.removeQueries({ queryKey: ['current-user'] });
    },
  };
}
