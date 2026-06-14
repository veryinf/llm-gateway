import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo, useState } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch, FormFieldTextarea } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { modelService, type Model } from '@/services/model';
import { providerService, type Provider } from '@/services/provider';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useQuery } from '@tanstack/react-query';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { request } from '@/lib';
import type { API } from '@/typings';

export const Route = createFileRoute('/upstream-models')({
  component: UpstreamModelsPage,
});

const pageInformation: PageInformation = {
  name: 'upstream-models',
  entityName: '模型',
  page: { title: '服务商模型', description: '管理各 Provider 下的模型详细信息' },
  breadcrumbs: [{ title: '上游' }, { title: '服务商模型' }],
};

function formatTokens(value: number) {
  if (!value) return '-';
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(0)}K`;
  return String(value);
}

function formatPrice(value: number) {
  if (!value) return '-';
  return `$${value.toFixed(2)}`;
}

function UpstreamModelsPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const [providerFilter, setProviderFilter] = useState<string>('all');

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const { data: providers } = useQuery({
    queryKey: ['providers-list'],
    queryFn: () => providerService.search({}),
  });

  const providerOptions = (providers?.dataSet ?? []).map((p: Provider) => ({
    label: p.name,
    value: String(p.id),
  }));

  const dynamicPageName = useMemo(() => {
    return providerFilter === 'all' ? 'upstream-models' : `upstream-models-p${providerFilter}`;
  }, [providerFilter]);

  const filteredService = useMemo<API.Service<Model>>(
    () => ({
      ...modelService,
      async search(_params) {
        const query = providerFilter !== 'all' ? `?provider_id=${providerFilter}` : '';
        const res = await request.get<API.DataSet<Model>>(`/admin/models${query}`);
        return res.data;
      },
    }),
    [providerFilter],
  );

  const dynamicPageInfo = useMemo(
    () => ({ ...pageInformation, name: dynamicPageName }),
    [dynamicPageName],
  );

  const columns: ColumnDef<Model, any>[] = [
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
      accessorKey: 'provider',
      header: 'Provider',
      meta: { label: 'Provider', className: 'w-[140px]' },
      cell: ({ row }) => row.original.provider?.name ?? '-',
    },
    {
      accessorKey: 'api_type',
      header: 'API 类型',
      meta: { label: 'API 类型', className: 'w-[90px]' },
      cell: ({ row }) => (
        <Badge variant="outline">{row.original.api_type === 'anthropic' ? 'Anthropic' : 'OpenAI'}</Badge>
      ),
    },
    {
      accessorKey: 'max_context_tokens',
      header: '上下文',
      meta: { label: '上下文', className: 'w-[90px]' },
      cell: ({ row }) => formatTokens(row.original.max_context_tokens),
    },
    {
      accessorKey: 'max_output_tokens',
      header: '最大输出',
      meta: { label: '最大输出', className: 'w-[90px]' },
      cell: ({ row }) => formatTokens(row.original.max_output_tokens),
    },
    {
      accessorKey: 'tpm',
      header: 'TPM',
      meta: { label: 'TPM', className: 'w-[80px]' },
      cell: ({ row }) => (row.original.tpm ? formatTokens(row.original.tpm) : '-'),
    },
    {
      accessorKey: 'qpm',
      header: 'QPM',
      meta: { label: 'QPM', className: 'w-[80px]' },
      cell: ({ row }) => (row.original.qpm ? String(row.original.qpm) : '-'),
    },
    {
      accessorKey: 'input_price',
      header: '输入单价',
      meta: { label: '输入单价', className: 'w-[90px]' },
      cell: ({ row }) => formatPrice(row.original.input_price),
    },
    {
      accessorKey: 'output_price',
      header: '输出单价',
      meta: { label: '输出单价', className: 'w-[90px]' },
      cell: ({ row }) => formatPrice(row.original.output_price),
    },
    {
      accessorKey: 'capabilities',
      header: '能力',
      meta: { label: '能力', className: 'w-[180px]' },
      cell: ({ row }) => {
        const caps: string[] = [];
        if (row.original.is_chat) caps.push('聊天');
        if (row.original.is_completion) caps.push('补全');
        if (row.original.is_vision) caps.push('视觉');
        if (row.original.is_embedding) caps.push('嵌入');
        return (
          <div className="flex flex-wrap gap-1">
            {caps.length > 0
              ? caps.map((c) => (
                  <Badge key={c} variant="secondary" className="text-xs">
                    {c}
                  </Badge>
                ))
              : '-'}
          </div>
        );
      },
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

  const formInitialValue = (_type: string, entity?: Model) => ({
    id: 0,
    provider_id: entity?.provider_id ?? 0,
    name: entity?.name ?? '',
    api_type: entity?.api_type ?? 'openai',
    display_name: entity?.display_name ?? '',
    description: entity?.description ?? '',
    max_context_tokens: entity?.max_context_tokens ?? 0,
    max_output_tokens: entity?.max_output_tokens ?? 0,
    input_price: entity?.input_price ?? 0,
    output_price: entity?.output_price ?? 0,
    tpm: entity?.tpm ?? 0,
    qpm: entity?.qpm ?? 0,
    is_chat: entity?.is_chat ?? true,
    is_completion: entity?.is_completion ?? false,
    is_vision: entity?.is_vision ?? false,
    is_embedding: entity?.is_embedding ?? false,
    is_active: entity?.is_active ?? true,
    created_at: '',
    updated_at: '',
  });

  const apiTypeOptions = [
    { label: 'OpenAI', value: 'openai' },
    { label: 'Anthropic', value: 'anthropic' },
  ];

  const renderForm = (form: any, _entity?: Model) => (
    <div className="flex flex-col gap-4 max-h-[70vh] overflow-y-auto pr-2">
      <div className="text-sm font-medium text-muted-foreground">基础信息</div>
      <FormFieldInput form={form} name="name" title="模型名称" required placeholder="例如: gpt-4o, claude-3-opus" />
      <FormFieldInput form={form} name="display_name" title="展示名" placeholder="用户友好的显示名称" />
      <FormFieldSelect form={form} name="provider_id" title="Provider" options={providerOptions} required />
      <FormFieldSelect form={form} name="api_type" title="API 类型" options={apiTypeOptions} required />
      <FormFieldTextarea form={form} name="description" title="描述" placeholder="模型描述信息" rows={2} />

      <div className="text-sm font-medium text-muted-foreground border-t pt-4">容量与限制</div>
      <div className="grid grid-cols-2 gap-4">
        <FormFieldInput form={form} name="max_context_tokens" title="最大上下文 (tokens)" type="number" placeholder="128000" />
        <FormFieldInput form={form} name="max_output_tokens" title="最大输出 (tokens)" type="number" placeholder="4096" />
      </div>

      <div className="text-sm font-medium text-muted-foreground border-t pt-4">速率限制</div>
      <div className="grid grid-cols-2 gap-4">
        <FormFieldInput form={form} name="tpm" title="TPM (0=不限)" type="number" tips="Tokens Per Minute" />
        <FormFieldInput form={form} name="qpm" title="QPM (0=不限)" type="number" tips="Queries Per Minute" />
      </div>

      <div className="text-sm font-medium text-muted-foreground border-t pt-4">定价 (per 1M tokens)</div>
      <div className="grid grid-cols-2 gap-4">
        <FormFieldInput form={form} name="input_price" title="输入单价 ($)" type="number" placeholder="0.00" />
        <FormFieldInput form={form} name="output_price" title="输出单价 ($)" type="number" placeholder="0.00" />
      </div>

      <div className="text-sm font-medium text-muted-foreground border-t pt-4">能力标记</div>
      <div className="grid grid-cols-2 gap-4">
        <FormFieldSwitch form={form} name="is_chat" title="聊天补全" switchLabel="支持 Chat Completions" />
        <FormFieldSwitch form={form} name="is_completion" title="文本补全" switchLabel="支持 Text Completions" />
        <FormFieldSwitch form={form} name="is_vision" title="视觉输入" switchLabel="支持图像输入" />
        <FormFieldSwitch form={form} name="is_embedding" title="嵌入" switchLabel="支持 Embeddings" />
      </div>

      <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用此服务商模型" />
    </div>
  );

  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 px-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Provider 筛选:</span>
            <Select value={providerFilter} onValueChange={setProviderFilter}>
              <SelectTrigger className="w-[200px] h-8">
                <SelectValue placeholder="全部 Provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部 Provider</SelectItem>
                {providerOptions.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>
      <Page<Model>
        infomation={dynamicPageInfo}
        columns={columns}
        service={filteredService}
        options={{ showSelectColumn: false }}
        formInitialValue={formInitialValue}
        renderViewAdd={(form) => renderForm(form)}
        renderViewUpdate={(form, entity) => renderForm(form, entity)}
      />
    </div>
  );
}
