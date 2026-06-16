import { request, type OptionsItem } from '@/lib';
import type { API } from '@/typings';
import { useQuery } from '@tanstack/react-query';

export interface User {
  uid: number;
  username: string;
  name: string;
  phone: string;
  department: string;
  role: 'admin' | 'user' | 'viewer';
  status: string;
  apiKeyCount: number;
}

export interface CreateUserParams {
  username: string;
  password: string;
  name?: string;
  phone?: string;
  department?: string;
  role?: 'admin' | 'user' | 'viewer';
}


export function useAllUsers() {
  const { data: allUsers = [], ...rest } = useQuery<User[]>({
    queryKey: ['all-users'],
    queryFn: async () => {
      const result = await userService.search({ pagination: { pageIndex: 1, pageSize: 10000 } });
      return result.dataSet ?? [];
    },
  });
  const allUserOptions = allUsers.map(u => ({
    label: `${u.name}(${u.username})`, value: u.uid
  } as OptionsItem));

  return { allUsers, allUserOptions, ...rest };
}

export const userService: API.Service<User> = {
  primaryKey: (entity) => entity.uid,
  title: (entity) => entity.username,

  async search(params) {
    const res = await request.post<API.DataSet<User>>('/user/search', params);
    return res.data;
  },

  async fetch(uid) {
    const res = await request.post<API.Data<User>>('/user/fetch', { uid });
    return res.data;
  },

  async add(params) {
    const res = await request.post<API.ResponseStruct>('/user/add', params);
    return res.data;
  },

  async update(uid, params) {
    const res = await request.post('/user/update', { uid: uid, ...params });
    return res.data;
  },

  async delete(id) {
    const res = await request.post('/user/remove', { uid: id });
    return res.data;
  },
};
