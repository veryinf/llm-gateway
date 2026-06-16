import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch, FormFieldTextarea } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { downstreamModelService, type DownstreamModel } from '@/services/downstream-model';
import { modelService, type Model } from '@/services/model';
import { providerService, type Provider } from '@/services/provider';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useQuery } from '@tanstack/react-query';

export const Route = createFileRoute('/downstream-models')({
  component: DownstreamModelsPage,
});

const pageInformation: PageInformation = {
  name: 'downstream-models',
  entityName: '模型',
  page: { title: '调用端模型', description: '配置暴露给用户的模型名称及其上游映射' },
  breadcrumbs: [{ title: '下游' }, { title: '调用端模型' }],
};

function DownstreamModelsPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const { data: upstreamModels } = useQuery({
    queryKey: ['provider-models-list'],
    queryFn: () => modelService.search({}),
  });

  const { data: providers } = useQuery({
    queryKey: ['providers-list'],
    queryFn: () => providerService.search({}),
  });

  const providerMap = useMemo(() => {
    const map: Record<number, Provider> = {};
    (providers?.dataSet ?? []).forEach((p: Provider) => {
      map[p.providerId] = p;
    });
    return map;
  }, [providers]);

  const upstreamModelOptions = useMemo(() => {
    return (upstreamModels?.dataSet ?? []).map((m: Model) => {
      const provider = providerMap[m.providerId];
      return {
        label: `${m.displayName || m.name} (${provider?.title ?? '未知'})`,
        value: String(m.modelId),
      };
    });
  }, [upstreamModels, providerMap]);

  const columns: ColumnDef<DownstreamModel, any>[] = [
    {
      accessorKey: 'name',
      header: '模型名称',
      meta: { label: '模型名称', className: 'w-[180px]', viewDetail: true },
    },
    {
      accessorKey: 'display_name',
      header: '展示名',
      meta: { label: '展示名', className: 'w-[140px]' },
      cell: ({ row }) => row.original.display_name || '-',
    },
    {
      accessorKey: 'upstream_model',
      header: '上游模型',
      meta: { label: '上游模型', className: 'w-[200px]' },
      cell: ({ row }) => {
        const um = row.original.upstream_model;
        if (!um) return '-';
        const provider = providerMap[um.providerId];
        return (
          <div className="flex items-center gap-1">
            <span>{um.displayName || um.name}</span>
            <Badge variant="outline" className="text-xs">
              {provider?.title ?? '未知'}
            </Badge>
          </div>
        );
      },
    },
    {
      accessorKey: 'description',
      header: '描述',
      meta: { label: '描述', className: 'w-[200px]' },
      cell: ({ row }) => row.original.description || '-',
    },
    {
      accessorKey: 'is_active',
      header: '状态',
      meta: { label: '状态', className: 'w-[70px]' },
      cell: ({ row }) => (
        <Badge variant={row.original.is_active ? 'default' : 'destructive'}>
          {row.original.is_active ? '启用' : '禁用'}
        </Badge>
      ),
    },
  ];

  return (
    <Page<DownstreamModel>
      infomation={pageInformation}
      columns={columns}
      service={downstreamModelService}
      options={{ showSelectColumn: false }}
      formInitialValue={(_type, entity) => ({
        id: 0,
        name: entity?.name ?? '',
        display_name: entity?.display_name ?? '',
        upstream_model_id: entity?.upstream_model_id ?? 0,
        description: entity?.description ?? '',
        is_active: entity?.is_active ?? true,
        created_at: '',
        updated_at: '',
      })}
      renderViewAdd={(form) => (
        <div className="flex flex-col gap-4">
          <FormFieldInput form={form} name="name" title="模型名称" required placeholder="例如: gpt-4, claude-3" />
          <FormFieldInput form={form} name="display_name" title="展示名" placeholder="用户友好的显示名称" />
          <FormFieldSelect form={form} name="upstream_model_id" title="上游模型" options={upstreamModelOptions} required />
          <FormFieldTextarea form={form} name="description" title="描述" placeholder="模型描述信息" rows={2} />
          <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用此调用端模型" />
        </div>
      )}
      renderViewUpdate={(form, _entity) => (
        <div className="flex flex-col gap-4">
          <FormFieldInput form={form} name="name" title="模型名称" required />
          <FormFieldInput form={form} name="display_name" title="展示名" />
          <FormFieldSelect form={form} name="upstream_model_id" title="上游模型" options={upstreamModelOptions} required />
          <FormFieldTextarea form={form} name="description" title="描述" rows={2} />
          <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用此调用端模型" />
        </div>
      )}
    />
  );
}
