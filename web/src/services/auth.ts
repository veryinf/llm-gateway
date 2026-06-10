import { request } from '@/lib';
import type { API } from '@/typings';

export interface SessionUser {
  id: number;
  username: string;
  name: string;
  phone: string;
  department: string;
  role: 'admin' | 'user' | 'viewer';
  is_active: boolean;
}

export interface LoginParams {
  username: string;
  password: string;
}

export async function login(params: LoginParams): Promise<void> {
  const res = await request.post<API.SingleResponse<{ token: string }>>('/admin/login', params);
  if (res.data.code !== 0) {
    throw new Error(res.data.msg || '登录失败');
  }
  localStorage.setItem('accessToken', res.data.data!.token);
}

export async function fetchProfile(): Promise<SessionUser | null> {
  if (!localStorage.getItem('accessToken')) {
    return null;
  }
  try {
    const res = await request.get<API.SingleResponse<SessionUser>>('/admin/profile');
    if (res.data.code !== 0) {
      return null;
    }
    return res.data.data!;
  } catch {
    return null;
  }
}

export function logout(): void {
  localStorage.removeItem('accessToken');
}
