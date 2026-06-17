import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo } from 'react';
import { useStore } from '@tanstack/react-form';
import type { ColumnDef } from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import { Page, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch, FormFieldTagsInput } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Loading } from '@/components/loader';
import { providerService, fetchProviderModels, type Provider } from '@/services/provider';
import { providerModelService } from '@/services/provider-model';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { toast } from 'sonner';
import type { EasyFormApi } from '@/components/form/utils';

export const Route = createFileRoute('/providers')({
  component: ProvidersPage,
});

const pageInformation: PageInformation = {
  name: 'providers',
  entityName: '服务商',
  page: { title: '服务商管理', description: '管理 LLM 服务商配置' },
  breadcrumbs: [{ title: '上游' }, { title: 'LLM 服务商' }],
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

const columns: ColumnDef<Provider, any>[] = [
  {
    accessorKey: 'title',
    header: '名称',
    meta: { label: '名称', className: 'w-[160px]', viewDetail: true },
  },
  {
    accessorKey: 'baseUrl',
    header: 'Base URL',
    meta: { label: 'Base URL', className: 'w-[240px]' },
    cell: ({ row }) => <span className="font-mono text-xs">{row.original.baseUrl}</span>,
  },
  {
    accessorKey: 'protocol',
    header: '协议',
    meta: { label: '协议', className: 'w-[120px]' },
    cell: ({ row }) => (
      <div className="flex gap-1">
        {row.original.supportOpenai && <Badge variant="secondary">OpenAI</Badge>}
        {row.original.supportAnthropic && <Badge variant="secondary">Anthropic</Badge>}
      </div>
    ),
  },
  {
    accessorKey: 'preferredApi',
    header: '优先接口',
    meta: { label: '优先接口', className: 'w-[80px]' },
    cell: ({ row }) => <Badge variant="outline">{row.original.preferredApi === 'openai' ? 'OpenAI' : 'Anthropic'}</Badge>,
  },
  {
    accessorKey: 'modelCount',
    header: '模型数',
    meta: { label: '模型数', className: 'w-[80px]' },
    cell: ({ row }) => <Badge variant={(row.original.modelCount ?? 0) > 0 ? 'default' : 'secondary'}>{row.original.modelCount ?? 0}</Badge>,
  },
  {
    accessorKey: 'isActive',
    header: '状态',
    meta: { label: '状态', className: 'w-[70px]' },
    cell: ({ row }) => <Badge variant={row.original.isActive ? 'default' : 'destructive'}>{row.original.isActive ? '启用' : '禁用'}</Badge>,
  },
];

// ---------- ProviderForm (新增/编辑共用) ----------

function ProviderForm({ form, entity }: { form: EasyFormApi<any>; entity?: Provider; }) {
  const isEdit = !!entity;

  const v = useStore(form.store ?? form, (s: any) => s.values ?? s) as any;
  const supportOpenai = v?.supportOpenai ?? false;
  const supportAnthropic = v?.supportAnthropic ?? false;
  const baseUrl = v?.baseUrl ?? '';

  const preferredApiOptions = useMemo(() => {
    const opts: { label: string; value: string; }[] = [];
    if (supportOpenai) opts.push({ label: 'OpenAI', value: 'openai' });
    if (supportAnthropic) opts.push({ label: 'Anthropic', value: 'anthropic' });
    return opts;
  }, [supportOpenai, supportAnthropic]);

  // 编辑模式：加载已有模型
  useQuery({
    enabled: isEdit,
    queryKey: ['provider-models', entity?.providerId],
    queryFn: async () => {
      const result = await providerModelService.search({
        filters: [{ field: 'providerId', value: entity!.providerId }],
      });
      const models = (result.dataSet ?? []).map((m) => m.name);
      form.setFieldValue('models', models);
      return result.dataSet;
    },
  });

  useEffect(() => {
    if (preferredApiOptions.length > 0 && !preferredApiOptions.find((o) => o.value === v?.preferredApi)) {
      form.setFieldValue('preferredApi', preferredApiOptions[0].value);
    }
  }, [preferredApiOptions, v?.preferredApi, form]);

  async function handleFetchModels() {
    if (!baseUrl) { toast.error('请先填写 BaseURL'); return []; }
    try {
      const fetched = await fetchProviderModels(baseUrl, v?.apiKey || '');
      toast.success(`获取到 ${fetched.length} 个模型`);
      return fetched.map((m) => m.id);
    } catch { toast.error('获取模型列表失败'); return []; }
  }

  return (
    <div className="grid grid-cols-12 gap-4">
      <FormFieldInput className="col-span-12" form={form} name="title" title="名称" required placeholder="服务商名称" />
      <FormFieldInput className="col-span-12" form={form} name="baseUrl" title="Base URL" required placeholder="https://api.example.com" />
      <FormFieldInput className="col-span-12" form={form} name="apiKey" title="API Key" required={!isEdit} placeholder={isEdit ? '留空不修改' : 'sk-...'} type="password" />
      <div className='col-span-12 grid grid-cols-12 gap-4'>
        <FormFieldSwitch className="col-span-2" form={form} name="supportOpenai" title="OpenAI协议接入" switchLabel="支持" />
        {form.getFieldValue("supportOpenai") && (
          <FormFieldInput className="col-span-10" form={form} name="openaiBaseUrl" title="OpenAI BaseURL" placeholder={baseUrl ? `${baseUrl}/v1` : 'https://api.example.com/v1'} />
        )}
      </div>
      <div className='col-span-12  grid grid-cols-12 gap-4'>
        <FormFieldSwitch className="col-span-2" form={form} name="supportAnthropic" title="Anthropic协议接入" switchLabel="支持" />
        {form.getFieldValue("supportAnthropic") && (
          <FormFieldInput className="col-span-10" form={form} name="anthropicBaseUrl" title="Anthropic BaseURL" placeholder={baseUrl ? `${baseUrl}/anthropic/v1` : 'https://api.example.com/anthropic/v1'} />
        )}
      </div>
      <FormFieldSwitch className="col-span-2" form={form} name="isActive" title="启用当前服务商" switchLabel="启用" />
      {preferredApiOptions.length > 0 && (
        <FormFieldSelect className="col-span-4" form={form} name="preferredApi" title="优先接口" options={preferredApiOptions} />
      )}
      <FormFieldTagsInput
        className="col-span-12 border-t pt-4"
        form={form}
        name="models"
        title="上游模型"
        placeholder="输入模型名称后按 Enter 添加..."
        titleMore={
          <Button type="button" size="sm" variant="outline" onClick={async () => {
            const tags = await handleFetchModels();
            if (tags.length > 0) {
              const current: string[] = (form.getFieldValue('models') as string[] | undefined) ?? [];
              const existing = new Set(current);
              form.setFieldValue('models', [...current, ...tags.filter((t) => !existing.has(t))]);
            }
          }}>
            从 API 获取
          </Button>
        }
      />
    </div>
  );
}

// ---------- ProvidersPage ----------

function ProvidersPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  return (
    <Page<Provider>
      infomation={pageInformation}
      columns={columns}
      service={providerService}
      options={{ showSelectColumn: false }}
      renderViewDetail={(entity) => <ProviderDetail entity={entity} />}
      formInitialValue={(_type, entity) => ({
        providerId: entity?.providerId ?? 0,
        title: entity?.title ?? '',
        baseUrl: entity?.baseUrl ?? '',
        apiKey: '',
        supportOpenai: entity?.supportOpenai ?? true,
        openaiBaseUrl: entity?.openaiBaseUrl ?? '',
        supportAnthropic: entity?.supportAnthropic ?? true,
        anthropicBaseUrl: entity?.anthropicBaseUrl ?? '',
        preferredApi: entity?.preferredApi ?? 'openai',
        isActive: entity?.isActive ?? true,
        models: [],
      })}
      formAddValidator={(e) => {
        if (!e.supportOpenai && !e.supportAnthropic) { toast.error('请至少支持一种协议'); return false; }
        return true;
      }}
      formUpdateValidator={(e) => {
        if (!e.supportOpenai && !e.supportAnthropic) { toast.error('请至少支持一种协议'); return false; }
        return true;
      }}
      renderViewForm={(form, entity) => <ProviderForm form={form} entity={entity} />}
    />
  );
}

// ---------- ProviderDetail ----------

function ProviderDetail({ entity }: { entity: Provider; }) {
  const pid = entity.providerId;

  const { data: models = [], isLoading: modelsLoading } = useQuery({
    queryKey: ['provider-models', pid],
    queryFn: async () => {
      const result = await providerModelService.search({
        filters: [{ field: 'providerId', value: pid }],
      });
      return result.dataSet ?? [];
    },
  });

  return (
    <div className="flex flex-col gap-4">
      <Descriptions
        title="服务商信息"
        labelClassName="w-20"
        items={[
          { label: '名称', value: entity.title },
          { label: 'Base URL', value: <span className="font-mono text-xs">{entity.baseUrl}</span> },
          {
            label: '协议',
            value: (
              <div className="flex gap-1">
                {entity.supportOpenai && <Badge variant="secondary">OpenAI</Badge>}
                {entity.supportAnthropic && <Badge variant="secondary">Anthropic</Badge>}
              </div>
            ),
          },
          { label: '优先接口', value: <Badge variant="outline">{entity.preferredApi === 'openai' ? 'OpenAI' : 'Anthropic'}</Badge> },
          { label: '状态', value: <Badge variant={entity.isActive ? 'default' : 'destructive'}>{entity.isActive ? '启用' : '禁用'}</Badge> },
        ]}
      />

      <Card>
        <CardHeader>
          <CardTitle>上游模型 ({models.length})</CardTitle>
        </CardHeader>
        <CardContent>
          {modelsLoading ? <Loading size={20} /> : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>模型名称</TableHead>
                  <TableHead>展示名</TableHead>
                  <TableHead>上下文</TableHead>
                  <TableHead>最大输出</TableHead>
                  <TableHead>输入单价</TableHead>
                  <TableHead>输出单价</TableHead>
                  <TableHead>状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {models.length === 0 ? (
                  <TableRow><TableCell colSpan={7} className="text-muted-foreground text-center">暂无模型</TableCell></TableRow>
                ) : models.map((m) => (
                  <TableRow key={m.modelId}>
                    <TableCell className="font-mono text-xs">{m.name}</TableCell>
                    <TableCell>{m.displayName || '-'}</TableCell>
                    <TableCell>{formatTokens(m.maxContextTokens)}</TableCell>
                    <TableCell>{formatTokens(m.maxOutputTokens)}</TableCell>
                    <TableCell>{formatPrice(m.inputPrice)}</TableCell>
                    <TableCell>{formatPrice(m.outputPrice)}</TableCell>
                    <TableCell><Badge variant={m.isActive ? 'default' : 'destructive'}>{m.isActive ? '启用' : '禁用'}</Badge></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
