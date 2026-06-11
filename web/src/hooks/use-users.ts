import { useQuery } from '@tanstack/react-query';
import { userService, type User } from '@/services/user';

const USERS_QUERY_KEY = ['users'];

export function useUsers() {
  const { data: users = [], ...rest } = useQuery<User[]>({
    queryKey: USERS_QUERY_KEY,
    queryFn: async () => {
      const result = await userService.search({});
      return result.dataSet ?? [];
    },
  });

  return { users, ...rest };
}
