import type { ReactNode } from 'react';
import { toast } from 'sonner';
import type { API } from '@/typings';

export type OptionsItem = { label: string | React.ReactNode; value: string | number; text?: string; };
export type GroupOptionsItem = { label: string | React.ReactNode; options: OptionsItem[]; };

export namespace UI {
  export async function tips(task: Promise<API.ResponseStruct>, msg: string = '操作成功'): Promise<boolean> {
    try {
      const { errCode, errMsg } = await task;
      if (errCode === 0) {
        toast.success(msg);
        return true;
      } else {
        toast.error('操作失败', { description: errMsg });
      }
    } catch (error) {
      toast.error('执行失败', { description: error as string });
    }
    return false;
  }

  export function toOptions(labels: Record<string, any>, render?: (value: any) => ReactNode): OptionsItem[] {
    return Object.keys(labels).map((key) => {
      return { label: render ? render(labels[key]) : labels[key], value: key, text: labels[key] };
    });
  }

  export function toOptions2<T>(entries: T[], key: keyof T, title?: keyof T, render?: (value: T) => ReactNode): OptionsItem[] {
    const label: keyof T = title || ('title' as any);
    return entries.map((entry) => {
      return {
        label: render ? render(entry) : entry[label],
        value: entry[key],
        text: entry[label],
      } as OptionsItem;
    });
  }
}
