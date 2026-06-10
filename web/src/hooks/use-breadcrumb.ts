import { create } from 'zustand';

export type BreadcrumbItem = {
  title: string;
  path?: string;
};

type BreadcrumbState = {
  breadcrumbs: BreadcrumbItem[];
  setBreadcrumbs: (breadcrumbs: BreadcrumbItem[]) => void;
};

export const useBreadcrumb = create<BreadcrumbState>((set) => ({
  breadcrumbs: [],
  setBreadcrumbs: (breadcrumbs) => set({ breadcrumbs }),
}));
