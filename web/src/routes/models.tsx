import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { modelService, type Model } from '@/services/model';
import { providerService } from '@/services/provider';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useQuery } from '@tanstack/react-query';

export const Route = createFileRoute('/models')({
  component: ModelsPage,
});

const pageInformation: PageInformation = {
  name: 'models',
  entityName: '模型',
  page: { title: '模型路由管理', description: '配置模型名称到 Provider 的路由映射' },
  breadcrumbs: [{ title: '管理' }, { title: '模型路由' }],
};

function ModelsPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const { data: providers } = useQuery({
    queryKey: ['providers-list'],
    queryFn: () => providerService.search({}),
  });

  const providerOptions = (providers?.dataSet ?? []).map((p: any) => ({
    label: `${p.name} (${p.type})`,
    value: String(p.id),
  }));

  const columns: ColumnDef<Model, any>[] = [
    {
      accessorKey: 'name',
      header: '模型名称',
      meta: { label: '模型名称', viewDetail: true },
    },
    {
      accessorKey: 'provider',
      header: 'Provider',
      meta: { label: 'Provider' },
      cell: ({ row }) => {
        const p = row.original.provider;
        return p ? `${p.name} (${p.type})` : '-';
      },
    },
    {
      accessorKey: 'is_active',
      header: '状态',
      meta: { label: '状态' },
      cell: ({ row }) => (
        <Badge variant={row.original.is_active ? 'default' : 'destructive'}>
          {row.original.is_active ? '启用' : '禁用'}
        </Badge>
      ),
    },
  ];

  return (
    <Page<Model>
      infomation={pageInformation}
      columns={columns}
      service={modelService}
      options={{ showSelectColumn: false }}
      formInitialValue={(_type, entity) => ({
        id: 0,
        provider_id: entity?.provider_id ?? 0,
        name: entity?.name ?? '',
        is_active: entity?.is_active ?? true,
        created_at: '',
        updated_at: '',
      })}
      renderViewAdd={(form) => (
        <div className="flex flex-col gap-4">
          <FormFieldInput form={form} name="name" title="模型名称" required placeholder="例如: gpt-4o, claude-3-opus" />
          <FormFieldSelect form={form} name="provider_id" title="Provider" options={providerOptions} required />
          <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用此模型路由" />
        </div>
      )}
      renderViewUpdate={(form, _entity) => (
        <div className="flex flex-col gap-4">
          <FormFieldInput form={form} name="name" title="模型名称" required />
          <FormFieldSelect form={form} name="provider_id" title="Provider" options={providerOptions} required />
          <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用此模型路由" />
        </div>
      )}
    />
  );
}
